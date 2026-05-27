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
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
)

// CloudTrailS3Event holds the fields extracted from an EventBridge event whose
// source is a CloudTrail S3 data event (PutObject or CompleteMultipartUpload).
type CloudTrailS3Event struct {
	AccountID  string // AWS account ID where the S3 event occurred
	BucketName string
	ObjectKey  string
	Region     string // AWS region of the bucket
	EventName  string // e.g. PutObject, CompleteMultipartUpload
}

// cloudTrailDetail mirrors the relevant subset of the CloudTrail event detail
// that EventBridge delivers for "AWS API Call via CloudTrail" events.
type cloudTrailDetail struct {
	EventName         string `json:"eventName"`
	AWSRegion         string `json:"awsRegion"`
	EventSource       string `json:"eventSource"`
	RequestParameters struct {
		BucketName string `json:"bucketName"`
		Key        string `json:"key"`
	} `json:"requestParameters"`
}

// ParseCloudTrailS3Event extracts bucket, key, region, and account from the
// EventBridge event envelope. It returns an error for any event that does not
// represent an S3 object creation (so the Lambda can safely discard it without
// retrying).
func ParseCloudTrailS3Event(event events.CloudWatchEvent) (*CloudTrailS3Event, error) {
	var detail cloudTrailDetail
	if err := json.Unmarshal(event.Detail, &detail); err != nil {
		return nil, fmt.Errorf("unmarshal CloudTrail detail: %w", err)
	}

	if detail.EventSource != "s3.amazonaws.com" {
		return nil, fmt.Errorf("unexpected eventSource %q (want s3.amazonaws.com)", detail.EventSource)
	}

	if detail.RequestParameters.BucketName == "" {
		return nil, fmt.Errorf("missing requestParameters.bucketName in event from account %s", event.AccountID)
	}

	if detail.RequestParameters.Key == "" {
		return nil, fmt.Errorf("missing requestParameters.key in event from account %s", event.AccountID)
	}

	// CloudTrail URL-encodes object keys that contain special characters.
	key, err := url.QueryUnescape(detail.RequestParameters.Key)
	if err != nil {
		key = detail.RequestParameters.Key
	}

	return &CloudTrailS3Event{
		AccountID:  event.AccountID,
		BucketName: detail.RequestParameters.BucketName,
		ObjectKey:  key,
		Region:     detail.AWSRegion,
		EventName:  detail.EventName,
	}, nil
}
