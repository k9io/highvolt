/*
** Copyright (C) 2026 Key9, Inc <k9.io>
** Copyright (C) 2026 Champ Clark III <cclark@k9.io>
**
** This file is part of the HighVolt JSON analysis engine
**
** This program is free software: you can redistribute it and/or modify
** it under the terms of the GNU Affero General Public License as published by
** the Free Software Foundation, either version 3 of the License, or
** (at your option) any later version.
**
** This program is distributed in the hope that it will be useful
** but WITHOUT ANY WARRANTY; without even the implied warranty of
** MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
** GNU Affero General Public License for more details.
**
** You should have received a copy of the GNU Affero General Public License
** along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package processors

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/k9io/highvolt/cmd/highvolt-server/debug"
	"github.com/k9io/highvolt/cmd/highvolt-server/llm"
	"github.com/k9io/highvolt/cmd/highvolt-server/models"

	l "github.com/k9io/highvolt/internal/logger"

	"github.com/facette/natsort"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func Submit_PDF(file_data string) string {

	var ai_data string
	var page_one_ai string
	var tmp string

	models.ConfigMu.RLock()
	workDirBase := models.C.Export_Directories.Work
	tempFilePerm := models.C.Core.Temp_File_Perm
	pdfCmd := models.C.Export_Commands.PDF
	maxPDFPages := models.C.Core.Max_PDF_Pages
	cmdTimeout := time.Duration(models.C.Core.Export_Command_Timeout) * time.Second
	models.ConfigMu.RUnlock()

	/* PDF are split into multiple pages.  This is the file format for those pages */

	OUTFILE := "highvolt-pdf-%d.png"

	/* Create a temp work directory to extract the PDF -> PNG to */

	WorkDir, err := os.MkdirTemp(workDirBase, "highvolt-pdf-*")

	if err != nil {
		l.Logger(l.ERROR, "Error creating temp directory: %v", err)
		return `{"status":"failed","reason":"cannot creating temp directory","code":500}`
	}

	defer os.RemoveAll(WorkDir) /* Nuke it on close */

	/* Decode the base64 of the PDF into the "real" file */

	decode_pdf, err := base64.StdEncoding.DecodeString(file_data)

	if err != nil {
		l.Logger(l.ERROR, "Failed to decode PDF base64: %v", err)
		return `{"status":"failed","reason":"failed to decode pdf base64","code":500}`
	}

	/* The target of the new PDF file */

	INFILE := fmt.Sprintf("%s/%s", WorkDir, "highvolt-pdf-tmp.pdf")

	err = os.WriteFile(INFILE, decode_pdf, tempFilePerm)

	if err != nil {
		l.Logger(l.ERROR, "Failed to write PDF temp file: %v", err)
		return `{"status":"failed","reason":"failed to write pdf temp file","code":500}`
	}

	if debug.X.Submit == true {

		l.Logger(l.DEBUG, "Storing PDF to %s", INFILE)
	}

	/* Replace command "masks" to actual values */

	cmd := strings.ReplaceAll(pdfCmd, "%INFILE%", INFILE)
	cmd = strings.ReplaceAll(cmd, "%WORKDIR%", WorkDir)
	cmd = strings.ReplaceAll(cmd, "%OUTFILE%", OUTFILE)

	tmp = fmt.Sprintf("[0-%d]", maxPDFPages-1)
	cmd = strings.ReplaceAll(cmd, "%RANGE%", tmp)

	if debug.X.Submit == true {

		l.Logger(l.DEBUG, "Executing: %s", cmd)

	}

	/* Securely build out the command to execute */

	cmd_parts := strings.Fields(cmd)

	if len(cmd_parts) == 0 {
		l.Logger(l.ERROR, "PDF conversion command is empty or invalid.")
		return `{"status":"failed","reason":"invalid conversion command","code":500}`
	}

	head := cmd_parts[0]
	args := cmd_parts[1:]

	/* Execute the command with a timeout */

	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd_exec := exec.CommandContext(ctx, head, args...)

	out, err := cmd_exec.CombinedOutput()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			l.Logger(l.ERROR, "PDF conversion command timed out after %v", cmdTimeout)
			return `{"status":"failed","reason":"conversion command timed out","code":500}`
		}
		l.Logger(l.ERROR, "PDF conversion command failed: %v", err)
		return `{"status":"failed","reason":"Error executing conversion program","code":500}`
	}

	if debug.X.Submit == true {

		l.Logger(l.DEBUG, "Output from command: %s", string(out))

	}

	/* We now look through all the .png files the converter made. */

	pattern := WorkDir + "/*.png" // We always convert PDF -> PNG

	matches, err := filepath.Glob(pattern)

	if err != nil {
		l.Logger(l.ERROR, "Error with pattern: %v", err)
		return `{"status":"failed","reason":"Cannot glob files for PDF analysis","code":500}`
	}

	if len(matches) == 0 {
		l.Logger(l.WARN, "PDF conversion produced no pages.")
		return `{"status":"failed","reason":"PDF produced no pages","code":500}`
	}

	natsort.Sort(matches) // Sort, so we have the proper page order

	/* If the PDF is many pages,  we only inspect the first X amount.  The idea is,
	   we don't want to waste resources processing thousands of pages.  Our bet
	   is that if the PDF has sensitive data,  it will be in the first pages */

	if len(matches) >= maxPDFPages {

		l.Logger(l.NOTICE, "PDF has %d+ pages.", len(matches))

	} else {

		l.Logger(l.NOTICE, "PDF has %d pages.", len(matches))

	}

	for page_number, imagePath := range matches {

		l.Logger(l.INFO, "Analyzing %s from PDF.", imagePath)

		/* Hit max pages,  it's likely clean and we return page one data */

		if page_number >= maxPDFPages {

			l.Logger(l.WARN, "Hit max PDF pages.  Stopping analysis.")
			return page_one_ai
		}

		imageData, err := os.ReadFile(imagePath)

		if err != nil {

			l.Logger(l.ERROR, "Error reading image: %v", err)
			return `{"status":"failed","reason":"Cannot read file data","code":500}`

		}

		/* The images need to be converted to base64 so the LLM can handle the
		   submission */

		base64Image := base64.StdEncoding.EncodeToString(imageData)
		ai_data = llm.Submit_AI(base64Image, "image/png", "IMAGE")

		has_sensitive_data := gjson.Get(ai_data, "has_sensitive_data").Bool()

		/* Store page one results incase it comes up "clean".  Since we analyze
		   multiple pages,  if there is no sensitive information,  we use the
		   first pages results for the "files" results.  Seems to work best */

		if page_number == 0 {
			page_one_ai = ai_data
		}

		/* If we find sensitive information,  we add a "page" to the LLM analysis.
		   This way, the user knows what PDF and page tripped Highvolt. */

		if has_sensitive_data == true {

			l.Logger(l.NOTICE, "Found sensitive data at page %d", page_number)
			ai_data, _ = sjson.Set(ai_data, "page_number", page_number)

			if debug.X.Submit == true {

				l.Logger(l.DEBUG, "JSON from LLM: %s", ai_data)

			}

			return ai_data

		}

	}

	/* If clean, we set the analysis to whatever page 1 was */

	page_one_ai, _ = sjson.Set(page_one_ai, "page_number", 1)

	if debug.X.Submit == true {

		l.Logger(l.DEBUG, "No sensitive data found in the PDF")

	}

	return page_one_ai // Return status,  and return no sensitive information

}
