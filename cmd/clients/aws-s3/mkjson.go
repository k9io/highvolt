package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/k9io/highvolt/internal/device"

	"github.com/tidwall/sjson"
)

func Mk_JSON(md5_hex string, sha1_hex string, sha256_hex string, b64_data string, mimetype string, bucket string, file string) string {

	now := time.Now()
	highvolt_timestamp := now.Format(time.RFC3339)

	highvolt_json, _ := sjson.Set("", "mimetype", mimetype)
	highvolt_json, _ = sjson.Set(highvolt_json, "timestamp", highvolt_timestamp)
	highvolt_json, _ = sjson.Set(highvolt_json, "file_data", b64_data)
	highvolt_json, _ = sjson.Set(highvolt_json, "app", "aws-s3")
	highvolt_json, _ = sjson.Set(highvolt_json, "md5", md5_hex)
	highvolt_json, _ = sjson.Set(highvolt_json, "sha1", sha1_hex)
	highvolt_json, _ = sjson.Set(highvolt_json, "sha256", sha256_hex)

	full_path := fmt.Sprintf("s3://%s/%s", bucket, file)
	filename := filepath.Base(file)

	highvolt_json, _ = sjson.Set(highvolt_json, "full_path", full_path)
	highvolt_json, _ = sjson.Set(highvolt_json, "filename", filename)

	highvolt_json = device.Device_Info_JSON(highvolt_json)

	return highvolt_json

}
