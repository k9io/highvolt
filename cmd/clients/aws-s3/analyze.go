package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"encoding/base64"

	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"

	"github.com/k9io/highvolt/internal/define"
	"github.com/k9io/highvolt/internal/http_req"
	"github.com/k9io/highvolt/internal/jwt"
	l "github.com/k9io/highvolt/internal/logger"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/gabriel-vasile/mimetype"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func Analyze(ctx context.Context, client *s3.Client, accID, bucket string, obj s3Types.Object, bearerToken *string, registry HashRegistry, dbPath string) {

	s3Path := fmt.Sprintf("s3://%s/%s", bucket, *obj.Key)

	if registry[s3Path] {
		l.Logger(l.DEBUG, "Already seen %s (local cache). Skipping.", s3Path)
		return
	}

	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    obj.Key,
	})

	if err != nil {
		l.Logger(l.ERROR, "S3 GetObject Error: %v", err)
		return
	}
	defer resp.Body.Close()

	/* Set a 1mb buffer for mime detection */

	limit := 1024 * 1024

	if int64(limit) > *obj.Size {
		limit = int(*obj.Size)
	}

	header := make([]byte, limit)
	n, err := io.ReadFull(resp.Body, header)

	if err != nil && err != io.ErrUnexpectedEOF {
		l.Logger(l.ERROR, "Error reading object %s: %v", s3Path, err)
		return
	}

	/* Detect true MIME (Zips vs Office) */

	mtype := mimetype.Detect(header[:n])

	l.Logger(l.DEBUG, "mime: %s", mtype)

	for _, i := range Config.Core.MIME_Types {

		if i == mtype.String() {

			l.Logger(l.DEBUG, "Process MIME: %s", mtype)

			if *obj.Size > Config.Core.Max_File_Size {
				l.Logger(l.NOTICE, "Skipping %s: size %d exceeds max_file_size %d", s3Path, *obj.Size, Config.Core.Max_File_Size)
				registry[s3Path] = true
				SaveRegistry(registry, dbPath)
				return
			}

			var b64Payload strings.Builder

			sha256 := sha256.New()
			sha1 := sha1.New()
			md5 := md5.New()

			fullStream := io.MultiReader(bytes.NewReader(header[:n]), resp.Body)
			b64 := base64.NewEncoder(base64.StdEncoding, &b64Payload)

			multi := io.MultiWriter(sha256, sha1, md5, b64)

			// This single copy operation feeds all four destinations at once

			_, err = io.Copy(multi, fullStream)

			if err != nil {
				l.Logger(l.ERROR, "Error streaming object %s: %v", s3Path, err)
				return
			}

			b64.Close()

			sha256_hex := hex.EncodeToString(sha256.Sum(nil))

			highvolt_query, _ := sjson.Set("", "sha256", sha256_hex)

			results, status, err := http_req.HTTP(highvolt_query, Config.Highvolt.Query, "POST", *bearerToken)

			if err != nil {
				l.Logger(l.ERROR, "%v", err)
				return
			}

			if status == 401 {
				l.Logger(l.NOTICE, "Highvolt bearer token expired. Getting a new one.")
				*bearerToken = jwt.PAT_Auth("highvolt", Config.Highvolt.URL, define.HIGHVOLT_VERSION, Config.Highvolt.Pat, false)
				results, _, err = http_req.HTTP(highvolt_query, Config.Highvolt.Query, "POST", *bearerToken)
				if err != nil {
					l.Logger(l.ERROR, "%v", err)
					return
				}
			}

			code := gjson.Get(results, "code").Int()

			if code == 200 {
				l.Logger(l.NOTICE, "%s has already been processed. Skipping.", s3Path)
				registry[s3Path] = true
				SaveRegistry(registry, dbPath)
				return
			}

			l.Logger(l.NOTICE, "Submitting %s [%s] for analysis", s3Path, mtype.String())

			sha1_hex := hex.EncodeToString(sha1.Sum(nil))
			md5_hex := hex.EncodeToString(md5.Sum(nil))

			highvolt_json := Mk_JSON(md5_hex, sha1_hex, sha256_hex, b64Payload.String(), mtype.String(), bucket, *obj.Key)

			_, status, err = http_req.HTTP(highvolt_json, Config.Highvolt.Submit, "POST", *bearerToken)

			if err != nil {
				l.Logger(l.ERROR, "%v", err)
				return
			}

			if status == 401 {
				l.Logger(l.NOTICE, "Highvolt bearer token expired. Getting a new one.")
				*bearerToken = jwt.PAT_Auth("highvolt", Config.Highvolt.URL, define.HIGHVOLT_VERSION, Config.Highvolt.Pat, false)
				_, _, err = http_req.HTTP(highvolt_json, Config.Highvolt.Submit, "POST", *bearerToken)
				if err != nil {
					l.Logger(l.ERROR, "%v", err)
					return
				}
			}

			registry[s3Path] = true

			f, err := os.Create(dbPath)
			if err != nil {
				l.Logger(l.ERROR, "Cannot create registry file %s: %v", dbPath, err)
			} else {
				if err = gob.NewEncoder(f).Encode(registry); err != nil {
					l.Logger(l.ERROR, "Cannot save registry to %s: %v", dbPath, err)
				}
				f.Close()
			}

			return
		}
	}

	// MIME type not eligible — record so we don't download this object again
	registry[s3Path] = true
	SaveRegistry(registry, dbPath)

}
