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
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// GetSecret retrieves a secret string from AWS Secrets Manager.
// If the secret value is a JSON object containing a "JSONAIR_PAT" key, that
// value is returned. Otherwise the raw secret string is returned, allowing
// the secret to be stored either as {"JSONAIR_PAT":"value"} or as a bare string.
func GetSecret(secretName string) (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", fmt.Errorf("loading AWS config: %w", err)
	}

	client := secretsmanager.NewFromConfig(cfg)

	result, err := client.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return "", fmt.Errorf("GetSecretValue(%s): %w", secretName, err)
	}

	if result.SecretString == nil {
		return "", fmt.Errorf("secret %s has no string value (binary secrets are not supported)", secretName)
	}

	var secretMap map[string]string
	if json.Unmarshal([]byte(*result.SecretString), &secretMap) == nil {
		if pat, ok := secretMap["JSONAIR_PAT"]; ok {
			return pat, nil
		}
	}

	return *result.SecretString, nil
}
