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

package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/k9io/highvolt/internal/device"
	"github.com/tidwall/sjson"
)

func MkJSON(md5hex, sha1hex, sha256hex, b64Data, mimeType, bucket, key string) string {
	j, _ := sjson.Set("", "mimetype", mimeType)
	j, _ = sjson.Set(j, "timestamp", time.Now().Format(time.RFC3339))
	j, _ = sjson.Set(j, "file_data", b64Data)
	j, _ = sjson.Set(j, "app", "aws-s3-lambda")
	j, _ = sjson.Set(j, "md5", md5hex)
	j, _ = sjson.Set(j, "sha1", sha1hex)
	j, _ = sjson.Set(j, "sha256", sha256hex)
	j, _ = sjson.Set(j, "full_path", fmt.Sprintf("s3://%s/%s", bucket, key))
	j, _ = sjson.Set(j, "filename", filepath.Base(key))
	j = device.Device_Info_JSON(j)
	return j
}
