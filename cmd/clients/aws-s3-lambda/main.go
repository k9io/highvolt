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
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gabriel-vasile/mimetype"
	"github.com/k9io/highvolt/internal/define"
	"github.com/k9io/highvolt/internal/device"
)

var (
	baseCfg         aws.Config
	stsClient       *sts.Client
	lambdaAccountID string
	highvoltToken   string
)

func init() {
	mimetype.SetLimit(1024 * 1024)

	if err := device.Get_Device_Info(); err != nil {
		log.Printf("[WARN] Could not collect device info: %v", err)
	}

	LoadEnv()

	pat, err := GetSecret(Env.JSONAIR_SECRET_NAME)
	if err != nil {
		log.Fatalf("[ERROR] Failed to retrieve JSONAIR PAT from Secrets Manager: %v", err)
	}
	Env.JSONAIR_PAT = pat

	configJSON := GetConfigJSON()
	LoadConfig(configJSON)

	highvoltToken = patAuth("highvolt", Config.Highvolt.URL, define.HIGHVOLT_VERSION, Config.Highvolt.Pat)
	log.Println("[INFO] Highvolt bearer token obtained")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("[ERROR] AWS SDK config error: %v", err)
	}
	baseCfg = cfg
	stsClient = sts.NewFromConfig(cfg)

	// Determine this Lambda's own account ID so we can skip AssumeRole for
	// same-account buckets (the management account has no OrganizationAccountAccessRole).
	identity, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		log.Fatalf("[ERROR] STS GetCallerIdentity failed: %v", err)
	}
	lambdaAccountID = aws.ToString(identity.Account)

	log.Printf("[INFO] Highvolt AWS S3 Lambda initialized (account: %s)", lambdaAccountID)
}

// s3ClientForAccount returns an S3 client scoped to the given account and region.
// For the Lambda's own account it uses the execution role directly; for all
// other accounts it assumes the configured cross-account role.
func s3ClientForAccount(ctx context.Context, accountID, region string) (*s3.Client, error) {
	regionOpt := func(o *s3.Options) { o.Region = region }

	if accountID == lambdaAccountID {
		return s3.NewFromConfig(baseCfg, regionOpt), nil
	}

	roleARN := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, Env.CROSS_ACCOUNT_ROLE)
	creds := stscreds.NewAssumeRoleProvider(stsClient, roleARN)

	memberCfg := baseCfg
	memberCfg.Credentials = aws.NewCredentialsCache(creds)
	memberCfg.Region = region

	return s3.NewFromConfig(memberCfg, regionOpt), nil
}

func handler(ctx context.Context, event events.CloudWatchEvent) error {
	s3Evt, err := ParseCloudTrailS3Event(event)
	if err != nil {
		// Log and discard — a parse failure means the event is malformed or
		// is not an S3 object-creation event. Returning nil avoids Lambda retries.
		log.Printf("[WARN] Skipping unprocessable event: %v", err)
		return nil
	}

	log.Printf("[INFO] %s event: s3://%s/%s (account=%s region=%s)",
		s3Evt.EventName, s3Evt.BucketName, s3Evt.ObjectKey,
		s3Evt.AccountID, s3Evt.Region)

	s3c, err := s3ClientForAccount(ctx, s3Evt.AccountID, s3Evt.Region)
	if err != nil {
		log.Printf("[ERROR] Cannot create S3 client for account %s: %v", s3Evt.AccountID, err)
		return nil
	}

	Analyze(ctx, s3c, s3Evt.BucketName, s3Evt.ObjectKey, &highvoltToken)
	return nil
}

func main() {
	lambda.Start(handler)
}
