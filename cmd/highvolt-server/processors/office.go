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
	"strings"
	"time"

	"github.com/k9io/highvolt/cmd/highvolt-server/debug"
	"github.com/k9io/highvolt/cmd/highvolt-server/models"

	l "github.com/k9io/highvolt/internal/logger"
)

/****************************************************************************/
/* Submit_Office - This requires some special handling.  We use an external */
/* tool like LibreOffice to convert the document to a PDF.  The PDF is then */
/* passed to the PDF processors,  which breaks it apart into difference     */
/* images for analysis.  Annoying? Yes...                                   */
/****************************************************************************/

func Submit_Office(file_data string) string {

	var tmp string
	var ai_data string

	models.ConfigMu.RLock()
	workDirBase := models.C.Export_Directories.Work
	tempFilePerm := models.C.Core.Temp_File_Perm
	officeCmd := models.C.Export_Commands.Office
	cmdTimeout := time.Duration(models.C.Core.Export_Command_Timeout) * time.Second
	models.ConfigMu.RUnlock()

	OUTFILE := "highvolt-pdf-%d.pdf" /* File we will convert from office format -> PDF */

	if debug.X.Submit == true {

		l.Logger(l.DEBUG, "Using output file: %s", OUTFILE)

	}

	/* Create a "temp" directory to work out of */

	WorkDir, err := os.MkdirTemp(workDirBase, "highvolt-office-*")

	if err != nil {

		l.Logger(l.ERROR, "Error creating temp dir: %v", err)
		return `{"status":"failed","reason":"error creating temp directory","code":500}`
	}

	defer os.RemoveAll(WorkDir) /* Nuke it on close */

	/* Decode the base64 back into the original file */

	decode_office, err := base64.StdEncoding.DecodeString(file_data)

	if err != nil {

		l.Logger(l.ERROR, "Failed to decode office base64")
		return `{"status":"failed","reason":"failed to decode office base64","code":500}`
	}

	/* Honor the users temp work path and create the file */

	INFILE := fmt.Sprintf("%s/%s", WorkDir, "highvolt-office")

	err = os.WriteFile(INFILE, decode_office, tempFilePerm)

	if err != nil {

		l.Logger(l.ERROR, "Failed to write file: %v", err)
		return `{"status":"failed","reason":"failed to write infile for office conversion","code":500}`
	}

	/* Replace command "masks" to actual values.  This is for the command line tool */

	cmd := strings.ReplaceAll(officeCmd, "%INFILE%", INFILE)
	cmd = strings.ReplaceAll(cmd, "%WORKDIR%", WorkDir)
	cmd = strings.ReplaceAll(cmd, "%OUTFILE%", OUTFILE)

	if debug.X.Submit == true {

		l.Logger(l.DEBUG, "Executing: %s", cmd)

	}

	/* Securely setup the command for execution */

	cmd_parts := strings.Fields(cmd)

	if len(cmd_parts) == 0 {
		l.Logger(l.ERROR, "Office conversion command is empty or invalid.")
		return `{"status":"failed","reason":"invalid conversion command","code":500}`
	}

	head := cmd_parts[0]
	args := cmd_parts[1:]

	/* Call the command with a timeout */

	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd_exec := exec.CommandContext(ctx, head, args...)

	out, err := cmd_exec.CombinedOutput()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			l.Logger(l.ERROR, "Office conversion command timed out after %v", cmdTimeout)
			return `{"status":"failed","reason":"conversion command timed out","code":500}`
		}
		l.Logger(l.ERROR, "Error executing command: %v", err)
		return `{"status":"failed","reason":"Error executing conversion program","code":500}`
	}

	if debug.X.Submit == true {

		l.Logger(l.DEBUG, "Output from command: %s", string(out))

	}

	/* Write the new PDF */

	tmp = INFILE + ".pdf"

	imageData, err := os.ReadFile(tmp)

	if err != nil {
		l.Logger(l.ERROR, "Error reading image: %v", err)
		return `{"status":"failed","reason":"Cannot read file data","code":500}`
	}

	/* Encode the newly convert PDF to the PDF processor */

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	ai_data = Submit_PDF(base64Image) /* Office -> PDF -> PNG */

	return ai_data

}
