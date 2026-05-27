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

/* Configuration struct's */

package models

import (
	"io/fs"
	"sync"
)

/* Configuration from Redis */

type Config_Struct struct {

	JSON		   string 

	Core               Core_Struct               `json:"core"`
	HTTP               HTTP_Struct               `json:"http"`
	Syslog             Syslog_Struct             `json:"syslog"`
	Opensearch         Opensearch_Struct         `json:"opensearch"`
	LLM                LLM_Struct                `json:"llm"`
	Export_Directories Export_Directories_Struct `json:"export_directories"`
	Export_Commands    Export_Commands_Struct    `json:"export_commands"`
	HA_Proxy           HA_Proxy_Struct `json:"haproxy"`
}

type Core_Struct struct {
	API_Key                  string      `json:"api_key"`
	Minimum_Image_Size       int         `json:"minimum_image_size"`
	Max_PDF_Pages            int         `json:"max_pdf_pages"`
	Max_Workers              int         `json:"max_workers"`
	Max_Queue_Size           int         `json:"max_queue_size"`
	Queue_Directory          string      `json:"queue_directory"`
	MIME_Types               MIME_Struct `json:"mime_types"`
	Temp_File_Mode           string      `json:"temp_file_mode"` // octal string, e.g. "0600"
	Temp_File_Perm           fs.FileMode `json:"-"`              // parsed from Temp_File_Mode
	Max_Archive_Size         int64       `json:"max_archive_size"`          // max total extracted bytes
	Archive_Extract_Timeout  int         `json:"archive_extract_timeout"`   // seconds
	Max_Body_Size            int64       `json:"max_body_size"`             // max HTTP request body bytes
	Export_Command_Timeout   int         `json:"export_command_timeout"`    // seconds for pdf/office conversion commands
}

type HA_Proxy_Struct struct {
	Enabled	 bool	`json:"enabled"`
	Port	int	`json:"port"`
}

type MIME_Struct struct {
	Text    []string `json:"text"`
	Image   []string `json:"image"`
	PDF     []string `json:"pdf"`
	Office  []string `json:"office"`
	Archive []string `json:"archive"`
}

type HTTP_Struct struct {
	TLS    bool   `json:"tls"`
	Listen string `json:"listen"`
	Cert   string `json:"cert"`
	Key    string `json:"key"`
	Mode   string `json:"mode"`
}

type Syslog_Struct struct {
	Host  string `json:"host"`
	Proto string `json:"proto"`
}

type Opensearch_Struct struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	URL             string `json:"url"`
	Index           string `json:"index"`
	TLS_Skip_Verify bool   `json:"tls_skip_verify"`
}

type LLM_Struct struct {
	API_Key       string `json:"api_key"`
	URL           string `json:"url"`
	Model         string `json:"model"`
	System_Prompt string `json:"system_prompt"`
	User_Prompt   string `json:"user_prompt"`
	Timeout       int    `json:"timeout"` // seconds
}

type Export_Directories_Struct struct {
	Work    string `json:"work"`
	Archive string `json:"archive"`
}

type Export_Commands_Struct struct {
	PDF    string `json:"pdf"`
	Office string `json:"office"`
}

var C Config_Struct
var ConfigMu sync.RWMutex
