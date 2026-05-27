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

/* This does a POST request to the LLM.  It is formated to use the OpenAI
   API.  It should be compatible with ChatGPT (OpenAI),  Ollama,  vLLM,
   and Llama.CPP */

package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/k9io/highvolt/cmd/highvolt-server/debug"
	"github.com/k9io/highvolt/cmd/highvolt-server/models"

	l "github.com/k9io/highvolt/internal/logger"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var llmClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 90 * time.Second,
	},
}

type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type VisionRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	Format         string          `json:"format"`
	Stream         bool            `json:"stream"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

/* This struct is verify the output from the LLM */

type AnalysisResult struct {
	HasSensitiveData bool   `json:"has_sensitive_data"`
	Confidence       string `json:"confidence"`
	Reasoning        string `json:"reasoning"`
	Description      string `json:"description"`
	Status           string `json:"status"`
}

/**********************************************************************/
/* Submit_AI - Submits data to your LLM of choice (as long as it uses */
/* the OpenAI API!).  LLM's expect the file data to be in a Base64    */
/* format and POST'ed to the API                                      */
/**********************************************************************/

func Submit_AI(file_data string, mime_type string, mime_ret string) string {

	var analysis AnalysisResult
	var payload VisionRequest

	models.ConfigMu.RLock()
	apiUrl := fmt.Sprintf("%s/%s", models.C.LLM.URL, "chat/completions")
	timeout := time.Duration(models.C.LLM.Timeout) * time.Second
	apiKey := models.C.LLM.API_Key
	llmModel := models.C.LLM.Model
	systemPrompt := models.C.LLM.System_Prompt
	userPrompt := models.C.LLM.User_Prompt
	models.ConfigMu.RUnlock()

	if mime_ret == "TEXT" {

		var textErr error
		payload, textErr = AI_TextRequest(mime_type, file_data, llmModel, systemPrompt, userPrompt)
		if textErr != nil {
			l.Logger(l.ERROR, "Failed to build text request: %v", textErr)
			return `{"status":"failed","code":500}`
		}

	} else {

		payload = AI_VisionRequest(mime_type, file_data, llmModel, systemPrompt, userPrompt)

	}

	jsonPayload, err := json.Marshal(payload)

	if err != nil {
		l.Logger(l.ERROR, "Failed to marshal LLM request: %v", err)
		return `{"status":"failed","code":500}`
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", apiUrl, bytes.NewBuffer(jsonPayload))

	if err != nil {
		l.Logger(l.ERROR, "Failed to create LLM HTTP request: %v", err)
		return `{"status":"failed","code":500}`
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := llmClient.Do(req)

	if err != nil {

		l.Logger(l.WARN, "HTTP request error: %v", err)
		return `{"status":"failed","code":500}`

	}
	defer resp.Body.Close()

	const maxLLMResponseSize = 1024 * 1024 // 1 MB — LLM analysis JSON is never legitimately larger

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxLLMResponseSize+1))

	if err != nil {
		l.Logger(l.ERROR, "Failed to read LLM response body: %v", err)
		return `{"status":"failed","code":500}`
	}

	if int64(len(body)) > maxLLMResponseSize {
		l.Logger(l.ERROR, "LLM response body exceeded %d bytes", maxLLMResponseSize)
		return `{"status":"failed","code":500}`
	}

	llm_json := gjson.Get(string(body), "choices.0.message.content").String()

	if llm_json == "" {

		l.Logger(l.WARN, "Cannot find LLM return message.")
		return `{"status":"failed","code":500}`

	}

	/* Sometimes the LLM can be sloppy and add crap that is shouldn't.  This
	   does a little bit of "clean up" in case that happens */

	llm_json = cleanJSON(llm_json)
	llm_json = minifyJSON(llm_json)

	/* Add "success" flag */

	llm_json, _ = sjson.Set(llm_json, "status", "success")

	/* Decode as a sanity check to make sure we have the data we need */

	err = json.Unmarshal([]byte(llm_json), &analysis)

	if err != nil {

		l.Logger(l.ERROR, "Unable to decode LLM JSON. [%s]", err)
		return `{"status":"failed","code":500}`

	}

	l.Logger(l.NOTICE, "Sample sensitive data status: %v", analysis.HasSensitiveData)

	if debug.X.LLM == true {

		l.Logger(l.DEBUG, "JSON return from LLM: %s", llm_json)

	}

	return llm_json
}

/************************************************************************/
/* minifyJSON - This is a backup for when a LLM fails to structure data */
/* properly.  In some cases,  LLMs will add extra spaces and new lines. */
/* In case that happens, this cleans that up                            */
/************************************************************************/

func minifyJSON(input string) string {

	dst := &bytes.Buffer{}

	err := json.Compact(dst, []byte(input))

	if err != nil {
		l.Logger(l.ERROR, "LLM returned non-JSON object: %v", err)
		return input
	}

	return dst.String()
}

/************************************************************************/
/* cleanJSON - More cleanup for LLM output.  In some cases,  LLMs will  */
/* add back ticks and a "this is json" line.  This is here as a backup  */
/* to clean that crap up                                                */
/************************************************************************/

func cleanJSON(input string) string {

	/* Clean some crap sometimes add by LLMs */

	input = strings.TrimPrefix(input, "```json")
	input = strings.TrimPrefix(input, "```")
	input = strings.TrimSuffix(input, "```")

	return strings.TrimSpace(input)
}

func AI_VisionRequest(mime_type string, file_data string, llmModel string, systemPrompt string, userPrompt string) VisionRequest {

	dataURL := fmt.Sprintf("data:%s;base64,%s", mime_type, file_data)

	payload := VisionRequest{
		Model:          llmModel,
		Stream:         false,
		Format:         "json",
		ResponseFormat: &ResponseFormat{Type: "json_object"},
		Messages: []Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role: "user",
				Content: []ContentPart{
					{Type: "text", Text: userPrompt},
					{Type: "image_url", ImageURL: &ImageURL{URL: dataURL}},
				},
			},
		},
	}

	return payload
}

func AI_TextRequest(mime_type string, file_data string, llmModel string, systemPrompt string, userPrompt string) (VisionRequest, error) {

	decoded_text, err := base64.StdEncoding.DecodeString(file_data)

	if err != nil {
		return VisionRequest{}, fmt.Errorf("failed to decode base64 text: %w", err)
	}

	tmp := fmt.Sprintf("%s. Here is the text from the file: %s", userPrompt, string(decoded_text))

	payload := VisionRequest{
		Model:          llmModel,
		Stream:         false,
		Format:         "json",
		ResponseFormat: &ResponseFormat{Type: "json_object"},
		Messages: []Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role: "user",
				Content: []ContentPart{
					{Type: "text", Text: userPrompt},
					{Type: "text", Text: tmp},
				},
			},
		},
	}

	return payload, nil

}
