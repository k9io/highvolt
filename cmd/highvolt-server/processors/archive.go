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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/facette/natsort"
	"github.com/mholt/archives"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/k9io/highvolt/cmd/highvolt-server/debug"
	"github.com/k9io/highvolt/cmd/highvolt-server/helpers"
	"github.com/k9io/highvolt/cmd/highvolt-server/models"

	"github.com/k9io/highvolt/internal/util"

	l "github.com/k9io/highvolt/internal/logger"
)

/*******************************************************************/
/* Submit_Archive - An archive can be a .tar.gz, .zip,, etc file.  */
/* This will tear apart the archive and analyze its contents.      */
/*******************************************************************/

func Submit_Archive(file_data string) string {

	var ai_data string
	var file_list []string

	models.ConfigMu.RLock()
	workDirBase := models.C.Export_Directories.Work
	archiveDirBase := models.C.Export_Directories.Archive
	tempFilePerm := models.C.Core.Temp_File_Perm
	archiveExtractTimeout := models.C.Core.Archive_Extract_Timeout
	maxArchiveSize := models.C.Core.Max_Archive_Size
	models.ConfigMu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(archiveExtractTimeout)*time.Second)
	defer cancel()

	/* Create a temp work directory */

	WorkDir, err := os.MkdirTemp(workDirBase, "highvolt-archive-*")

	if err != nil {

		l.Logger(l.ERROR, "Error create temp WorkDir directory: %v. Analysis faile", err)
		return `{"status":"failed","reason":"Cannot create temp directory","code":500}`

	}

	defer os.RemoveAll(WorkDir) /* Nuke it on exit */

	/* Create the temp directory for the archive to be exploded in */

	WorkDirZip, err := os.MkdirTemp(archiveDirBase, "highvolt-archive-zip-*")

	if err != nil {

		l.Logger(l.ERROR, "Error create temp ZipDir directory: %v. Analysis faile", err)
		return `{"status":"failed","reason":"Cannot create temp zip directory","code":500}`

	}

	defer os.RemoveAll(WorkDirZip) /* Nuke it on exit */

	/* Decode the Base64 from our submission to get to the "real" file */

	decode_archive, err := base64.StdEncoding.DecodeString(file_data)

	if err != nil {

		l.Logger(l.WARN, "Failed to decode base64: %v", err)
		return `{"status":"failed","reason":"failed to decode base64","code":500}`

	}

	/* Honor the temp working directory and create a new archive file */

	INFILE := fmt.Sprintf("%s/%s", WorkDirZip, "highvolt-archive-tmp.zip")

	err = os.WriteFile(INFILE, decode_archive, tempFilePerm)

	if err != nil {

		l.Logger(l.ERROR, "Failed to write temp file: %v", err)
		return `{"status":"failed","reason":"failed to write temp file","code":500}`

	}

	if debug.X.Submit == true {

		l.Logger(l.DEBUG, "Storing archive to %s", INFILE)

	}

	/* Open the Zip file */

	zip_file, err := os.Open(INFILE)

	if err != nil {

		l.Logger(l.ERROR, "Cannot open archive file: %v", err)
		return `{"status":"failed","reason":"failed to open archive file","code":500}`

	}

	defer zip_file.Close()

	/* Identify the format */

	format, _, err := archives.Identify(ctx, INFILE, zip_file)

	if err != nil {

		l.Logger(l.ERROR, "Cannot identify archive type: %v", err)
		return `{"status":"failed","reason":"failed to identify archive file","code":500}`

	}

	/* Type-assert to see if it's an Extractor */

	ex, ok := format.(archives.Extractor)

	if !ok {

		l.Logger(l.WARN, "Format %s does not support extraction\n", format.Extension())
		return `{"status":"failed","reason":"Format does not support extraction","code":500}`

	}

	/* Define the Handler (The "callback" for each file in the archive) */

	var totalExtracted atomic.Int64

	handler := func(ctx context.Context, f archives.FileInfo) error {

		/* Create the full path for the file/directory */

		outputPath := filepath.Join(WorkDir, f.NameInArchive)

		/* Guard against path traversal attacks (e.g. ../../etc/passwd in archive) */

		cleanWork := filepath.Clean(WorkDir)
		if !strings.HasPrefix(filepath.Clean(outputPath), cleanWork+string(os.PathSeparator)) {
			l.Logger(l.WARN, "Skipping path traversal attempt in archive: %s", f.NameInArchive)
			return nil
		}

		if f.IsDir() {
			return os.MkdirAll(outputPath, 0700)
		}

		/* Ensure the parent directory exists */

		if err := os.MkdirAll(filepath.Dir(outputPath), 0700); err != nil {
			return err
		}

		/* Open the file within the archive */

		rc, err := f.Open()

		if err != nil {
			return err
		}

		defer rc.Close()

		/* Create the file on disk */

		dst, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, tempFilePerm)

		if err != nil {
			return err
		}

		defer dst.Close()

		/* Copy with a size limit to guard against zip bombs */

		remaining := maxArchiveSize - totalExtracted.Load()
		if remaining <= 0 {
			return errors.New("archive extraction size limit exceeded")
		}

		n, err := io.Copy(dst, io.LimitReader(rc, remaining+1))
		totalExtracted.Add(n)

		if totalExtracted.Load() > maxArchiveSize {
			return fmt.Errorf("archive extraction size limit of %d bytes exceeded", maxArchiveSize)
		}

		return err
	}

	/* Run the extraction */

	err = ex.Extract(ctx, zip_file, handler)

	if err != nil {

		l.Logger(l.WARN, "Extraction of archive files: %v", err)
		return `{"status":"failed","reason":"extraction of archive failed","code":500}`

	} else {

		l.Logger(l.INFO, "Extraction completed successfully.")
	}

	/* We now walk the archive's path for all "file" to analyze */

	err = filepath.WalkDir(WorkDir, func(path string, d os.DirEntry, err error) error {

		if err != nil {
			return err
		}

		if !d.IsDir() {

			/* Create a file list of the archives contents */

			file_list = append(file_list, path)

		}

		return nil
	})

	if err != nil {

		l.Logger(l.WARN, "Error walking path: %v", err)
		return `{"status":"failed","reason":"error walking archive path","code":500}`

	}

	natsort.Sort(file_list)

	total_files := len(file_list)

	/* Go through the files and submit the ones in our MIME list for analysis */

	for file_num, f := range file_list {

		mimetype := util.GetFileMagic(f)

		l.Logger(l.INFO, "[%d/%d] Processing file: %s, MIME Type: %s", file_num+1, total_files, f, mimetype)

		/* Is the MIME type in our list? */

		check_mime, mime_ret := helpers.Check_MIME(mimetype)

		/* If yes,  read in the file */

		if check_mime == true {

			imageData, err := os.ReadFile(f)

			if err != nil {

				l.Logger(l.ERROR, "Error reading file in archive: %v", err)

			}

			/* Base64 the file so the LLM can handle it */

			base64Image := base64.StdEncoding.EncodeToString(imageData)

			/* Submit the file, along with its MIME type for LLM analysis */

			ai_data = Submit_Data(base64Image, mimetype, mime_ret)

			if debug.X.Submit == true {

				l.Logger(l.DEBUG, "JSON returned from LLM: %s", ai_data)

			}

			/* Post analysis for the file,  lets check for sensitive data.  If there
			   is sensitive data, we don't need to go any futher.  We know the
			   archive has PII */

			has_sensitive_data := gjson.Get(ai_data, "has_sensitive_data").Bool()

			if has_sensitive_data == true {

				l.Logger(l.NOTICE, "Archive appears to have sensitive information.")

				relPath, _ := filepath.Rel(WorkDir, f)
				ai_data, _ = sjson.Set(ai_data, "file_in_zip", relPath)

				if debug.X.Submit == true {

					l.Logger(l.DEBUG, "JSON Returned from LLM: %s", ai_data)

				}

				return ai_data

			}

		} else {

			l.Logger(l.NOTICE, "Magic type: '%s' is not support. Skipping.", mimetype)
		}

	}

	/* Make it through all files,  we consider it clean */

	l.Logger(l.NOTICE, "Archive file appears to be clean.")

	return `{"has_sensitive_data":false,"confidence":"medium","reasoning":"None of the files in the archive contain sensitive data.","description":"No readable or identifiable sensitive information detected","status":"success","code":200}`

}
