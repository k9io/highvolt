package main

import (
	"context"
	"fmt"
	"os"

	"github.com/k9io/highvolt/internal/define"
	"github.com/k9io/highvolt/internal/device"
	"github.com/k9io/highvolt/internal/jwt"

	l "github.com/k9io/highvolt/internal/logger"

	"github.com/gabriel-vasile/mimetype"

	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type AWS_Struct struct {
	Name string
	ID   string
}

func init() {

	// Set 1MB limit so Office docs (Zips) can be deeply identified

	mimetype.SetLimit(1024 * 1024)

	device.Get_Device_Info()
}

func main() {

	var AWS []AWS_Struct
	var TMP_AWS AWS_Struct

	l.Init_Logger("local", "tcp") // Need config support for syslog

	l.Logger(l.INFO, "Firing up Highvolt AWS-S3")

	LoadEnv()

	l.Logger(l.INFO, "Loading config from %s/config", Env.JSONAIR_URL)

	JSONAIR_bearerToken := jwt.PAT_Auth("jsonair", Env.JSONAIR_URL, define.JSONAIR_VERSION, Env.JSONAIR_PAT, false)

	JSONAIR_config, JSONAIR_bearerToken := GetConfigJSON(JSONAIR_bearerToken)

	LoadConfig(JSONAIR_config)

	HIGHVOLT_bearerToken := jwt.PAT_Auth("highvolt", Config.Highvolt.URL, define.HIGHVOLT_VERSION, Config.Highvolt.Pat, false)

	l.Logger(l.DEBUG, "Highvolt bearer token obtained.")

	l.Init_Logger(Config.Syslog.Host, Config.Syslog.Proto) /* Reload logging based off config */

	dbPath := GetHighvoltDBFileName()
	registry := LoadRegistry(dbPath)

	l.Logger(l.INFO, "AWS-S3 SHA256 database: %s", dbPath)

	ctx := context.TODO()

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(Config.AWS.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			Config.AWS.Access_Key_ID,
			Config.AWS.Secret_Access_Key,
			"", // Session token - leave empty if not using temporary creds
		)),
	)

	if err != nil {
		l.Logger(l.ERROR, "AWS configuration error: %v", err)
		os.Exit(1)
	}

	orgClient := organizations.NewFromConfig(cfg)
	stsClient := sts.NewFromConfig(cfg)

	l.Logger(l.NOTICE, "Scanning Organization....")

	pag := organizations.NewListAccountsPaginator(orgClient, &organizations.ListAccountsInput{})

	for pag.HasMorePages() {

		page, err := pag.NextPage(ctx)

		if err != nil {
			l.Logger(l.ERROR, "Error listing accounts: %v", err)
			os.Exit(1)
		}

		l.Logger(l.NOTICE, "Accounts discovered.")

		for _, account := range page.Accounts {

			/* DEBUG: Can't goroutine this? */

			//                        processAccount(ctx, account, cfg, stsClient)

			l.Logger(l.INFO, "Name: %s,  ID: %s", *account.Name, *account.Id)

			flag := false

			for _, ignore := range Config.S3.Exclude_Users {

				if ignore == *account.Name {
					flag = true
				}

			}

			if flag == false {

				TMP_AWS.Name = *account.Name
				TMP_AWS.ID = *account.Id

				AWS = append(AWS, TMP_AWS)
			}

		}
	}

	/* Cycle through AWS users */

	for _, account := range AWS {

		l.Logger(l.DEBUG, "Account: %s (%s)", account.Name, account.ID)

		roleArn := fmt.Sprintf("arn:aws:iam::%s:role/OrganizationAccountAccessRole", account.ID)
		creds := stscreds.NewAssumeRoleProvider(stsClient, roleArn)

		memberCfg := cfg

		memberCfg.Credentials = aws.NewCredentialsCache(creds)
		s3Client := s3.NewFromConfig(memberCfg)

		buckets, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})

		if err != nil {
			l.Logger(l.ERROR, "Error accessing %s (%s): %v", account.Name, account.ID, err)
			continue
		}

		for _, b := range buckets.Buckets {

			bucketName := *b.Name

			l.Logger(l.DEBUG, "Bucket: %s", bucketName)

			objPag := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{Bucket: &bucketName})

			for objPag.HasMorePages() {

				page, err := objPag.NextPage(ctx)
				if err != nil {
					l.Logger(l.ERROR, "Error listing objects in bucket %s: %v", bucketName, err)
					break
				}

				for _, obj := range page.Contents {
					Analyze(ctx, s3Client, account.ID, bucketName, obj, &HIGHVOLT_bearerToken, registry, dbPath)
				}
			}
		}

	}

}
