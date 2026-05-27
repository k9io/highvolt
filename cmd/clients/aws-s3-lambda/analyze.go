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
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gabriel-vasile/mimetype"
	"github.com/k9io/highvolt/internal/define"
	"github.com/k9io/highvolt/internal/http_req"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func Analyze(ctx context.Context, client *s3.Client, bucket, key string, bearerToken *string) {
	s3Path := fmt.Sprintf("s3://%s/%s", bucket, key)

	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		log.Printf("[ERROR] S3 GetObject %s: %v", s3Path, err)
		return
	}
	defer resp.Body.Close()

	// Reject oversized files before reading. ContentLength is -1 when unknown;
	// in that case we let the stream proceed and rely on the MIME detection cap.
	if resp.ContentLength != nil && *resp.ContentLength > Config.Core.Max_File_Size {
		log.Printf("[NOTICE] %s: size %d exceeds max_file_size %d, skipping",
			s3Path, *resp.ContentLength, Config.Core.Max_File_Size)
		return
	}

	// Read up to 1MB for accurate MIME detection (handles ZIP-based formats like DOCX).
	limit := 1024 * 1024
	if resp.ContentLength != nil && *resp.ContentLength > 0 && *resp.ContentLength < int64(limit) {
		limit = int(*resp.ContentLength)
	}

	header := make([]byte, limit)
	n, err := io.ReadFull(resp.Body, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		log.Printf("[ERROR] Reading object header %s: %v", s3Path, err)
		return
	}

	mtype := mimetype.Detect(header[:n])
	log.Printf("[DEBUG] %s detected as %s", s3Path, mtype)

	mimeAllowed := false
	for _, allowed := range Config.Core.MIME_Types {
		if allowed == mtype.String() {
			mimeAllowed = true
			break
		}
	}

	if !mimeAllowed {
		log.Printf("[DEBUG] %s: MIME type %s not in allowlist, skipping", s3Path, mtype)
		return
	}

	// Stream the full object through all hash writers and the base64 encoder in one pass.
	var b64Payload strings.Builder
	sha256hash := sha256.New()
	sha1hash := sha1.New()
	md5hash := md5.New()

	fullStream := io.MultiReader(bytes.NewReader(header[:n]), resp.Body)
	b64enc := base64.NewEncoder(base64.StdEncoding, &b64Payload)
	multi := io.MultiWriter(sha256hash, sha1hash, md5hash, b64enc)

	if _, err = io.Copy(multi, fullStream); err != nil {
		log.Printf("[ERROR] Streaming object %s: %v", s3Path, err)
		return
	}
	b64enc.Close()

	sha256hex := hex.EncodeToString(sha256hash.Sum(nil))

	// Check whether Highvolt has already processed this content by SHA256.
	queryBody, _ := sjson.Set("", "sha256", sha256hex)
	results, status, err := http_req.HTTP(queryBody, Config.Highvolt.Query, "POST", *bearerToken)
	if err != nil {
		log.Printf("[ERROR] Highvolt query for %s: %v", s3Path, err)
		return
	}
	if status == 401 {
		log.Println("[NOTICE] Highvolt bearer token expired, refreshing")
		*bearerToken = patAuth("highvolt", Config.Highvolt.URL, define.HIGHVOLT_VERSION, Config.Highvolt.Pat)
		results, _, err = http_req.HTTP(queryBody, Config.Highvolt.Query, "POST", *bearerToken)
		if err != nil {
			log.Printf("[ERROR] Highvolt query after token refresh for %s: %v", s3Path, err)
			return
		}
	}

	if gjson.Get(results, "code").Int() == 200 {
		log.Printf("[NOTICE] %s already processed by Highvolt, skipping", s3Path)
		return
	}

	log.Printf("[NOTICE] Submitting %s [%s] for PII analysis", s3Path, mtype)

	sha1hex := hex.EncodeToString(sha1hash.Sum(nil))
	md5hex := hex.EncodeToString(md5hash.Sum(nil))

	payload := MkJSON(md5hex, sha1hex, sha256hex, b64Payload.String(), mtype.String(), bucket, key)

	_, status, err = http_req.HTTP(payload, Config.Highvolt.Submit, "POST", *bearerToken)
	if err != nil {
		log.Printf("[ERROR] Highvolt submit for %s: %v", s3Path, err)
		return
	}
	if status == 401 {
		log.Println("[NOTICE] Highvolt bearer token expired, refreshing")
		*bearerToken = patAuth("highvolt", Config.Highvolt.URL, define.HIGHVOLT_VERSION, Config.Highvolt.Pat)
		_, _, err = http_req.HTTP(payload, Config.Highvolt.Submit, "POST", *bearerToken)
		if err != nil {
			log.Printf("[ERROR] Highvolt submit after token refresh for %s: %v", s3Path, err)
			return
		}
	}

	log.Printf("[INFO] %s submitted successfully for PII analysis", s3Path)
}
