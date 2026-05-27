package config

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"time"

	"github.com/k9io/highvolt/cmd/highvolt-server/auth"
	"github.com/k9io/highvolt/cmd/highvolt-server/models"

	"github.com/k9io/highvolt/internal/define"
	"github.com/k9io/highvolt/internal/http_req"
	l "github.com/k9io/highvolt/internal/logger"
)

func GetConfigJSON(bearerToken string) (string, string) {

	var err error
	var config_json string
	var status_code int

	config_url := fmt.Sprintf("%s/api/%s/jsonair/config", models.Env.JSONAIR_URL, define.JSONAIR_VERSION)
	config_json_tmp := fmt.Sprintf(`{"type":"%s","name":"%s","decode":true}`, models.Env.JSONAIR_TYPE, models.Env.JSONAIR_NAME)

	for attempt := 0; attempt < 10; attempt++ {

		if attempt > 0 {
			wait := time.Duration(1<<uint(attempt-1)) * time.Second
			l.Logger(l.NOTICE, "Retrying config fetch in %v (attempt %d/10).", wait, attempt+1)
			time.Sleep(wait)
		}

		config_json, status_code, err = http_req.HTTP(config_json_tmp, config_url, "GET", bearerToken)

		if err != nil {
			l.Logger(l.ERROR, "%v", err)
			continue
		}

		if status_code == 401 {
			l.Logger(l.NOTICE, "Bearer Token expired. Getting a new one.")
			bearerToken = auth.PAT_Auth()
			continue
		}

		if status_code == 200 {
			return config_json, bearerToken
		}

		l.Logger(l.ERROR, "Got bad response %v.", status_code)
	}

	l.Logger(l.ERROR, "Failed to fetch config after 10 attempts.")
	os.Exit(1)
	return "", bearerToken
}

func Load_Config(configJSON string) {

	models.ConfigMu.Lock()
	defer models.ConfigMu.Unlock()

	models.C = models.Config_Struct{} // Reset struct

	/* Set some defaults */

	models.C.Syslog.Host = "local"

	models.C.Core.Minimum_Image_Size = 5128
	models.C.Core.Max_PDF_Pages = 50

	models.C.Core.Max_Workers = define.DEFAULT_MAX_WORKERS
	models.C.Core.Max_Queue_Size = 50

	models.C.Core.Temp_File_Mode = "0600"
	models.C.Core.Max_Archive_Size = 500 * 1024 * 1024 // 500 MB
	models.C.Core.Archive_Extract_Timeout = 300         // 5 minutes
	models.C.Core.Max_Body_Size = 1024 * 1024 * 1024   // 1 GB
	models.C.Core.Export_Command_Timeout = 120          // 2 minutes

	models.C.LLM.Timeout = 120

	models.C.HTTP.Mode = "production"
	models.C.HTTP.TLS = false

	models.C.Syslog.Host = "local"
	models.C.Syslog.Proto = "tcp"

	err := json.Unmarshal([]byte(configJSON), &models.C)

	if err != nil {

		l.Logger(l.ERROR, "Unable to decode JSON from highvolt config JSON. [%s]", err)
		os.Exit(1)

	}

	/* Parse temp file mode — stored as octal string (e.g. "0600") in config */

	modeVal, err := strconv.ParseUint(models.C.Core.Temp_File_Mode, 8, 32)

	if err != nil {
		l.Logger(l.ERROR, "Invalid 'core.temp_file_mode' value '%s': must be an octal string like \"0600\".", models.C.Core.Temp_File_Mode)
		os.Exit(1)
	}

	models.C.Core.Temp_File_Perm = fs.FileMode(modeVal)

	/* Sanity Check */

	if models.C.Core.Max_Workers == 0 {

			l.Logger(l.WARN, "'max_workers' not set.  Using default of %d", define.DEFAULT_MAX_WORKERS)
			models.C.Core.Max_Workers = define.DEFAULT_MAX_WORKERS

		}


	/* --- Core --- */

	if len(models.C.Core.MIME_Types.Image) == 0 {

		l.Logger(l.ERROR, "Missing 'core.mime_types.image'.")
		os.Exit(1)

	}

	if len(models.C.Core.MIME_Types.PDF) == 0 {

		l.Logger(l.ERROR, "Missing 'core.mime_types.pdf'.")
		os.Exit(1)

	}

	if len(models.C.Core.MIME_Types.Office) == 0 {

		l.Logger(l.ERROR, "Missing 'core.mime_types.office'.")
		os.Exit(1)

	}

	if len(models.C.Core.Queue_Directory) == 0 {

		l.Logger(l.ERROR, "Missing 'core.queue_directory.archive'.")
		os.Exit(1)

	}

	/* --- HTTP --- */

	if models.C.HTTP.Listen == "" {

		l.Logger(l.ERROR, "Missing 'http.listen'.")
		os.Exit(1)

	}

	if models.C.HTTP.TLS == true {

		if models.C.HTTP.Cert == "" {

			l.Logger(l.ERROR, "Missing 'http.cert'.")
			os.Exit(1)

		}

		if models.C.HTTP.Key == "" {

			l.Logger(l.ERROR, "Missing 'http.key'.")
			os.Exit(1)

		}

	}

	if models.C.Opensearch.Username == "" {

		l.Logger(l.ERROR, "Missing 'opensearch.username'.")
		os.Exit(1)

	}

	if models.C.Opensearch.Password == "" {

		l.Logger(l.ERROR, "Missing 'opensearch.password'.")
		os.Exit(1)

	}

	if models.C.Opensearch.URL == "" {

		l.Logger(l.ERROR, "Missing 'opensearch.url'.")
		os.Exit(1)

	}

	if models.C.Opensearch.Index == "" {

		l.Logger(l.ERROR, "Missing 'opensearch.index'.")
		os.Exit(1)

	}

	/* --- LLM --- */

	if models.C.LLM.API_Key == "" {

		l.Logger(l.ERROR, "Missing 'llm.api_key'.")
		os.Exit(1)

	}

	if models.C.LLM.URL == "" {

		l.Logger(l.ERROR, "Missing 'llm.url'.")
		os.Exit(1)

	}

	if models.C.LLM.Model == "" {

		l.Logger(l.ERROR, "Missing 'llm.model'.")
		os.Exit(1)

	}

	if models.C.LLM.System_Prompt == "" {

		l.Logger(l.ERROR, "Missing 'llm.system_prompt'.")
		os.Exit(1)

	}

	if models.C.LLM.User_Prompt == "" {

		l.Logger(l.ERROR, "Missing 'llm.user_prompt'.")
		os.Exit(1)

	}

	/* --- Export Directories --- */

	if models.C.Export_Directories.Work == "" {

		l.Logger(l.ERROR, "Missing 'export_directories.work'.")
		os.Exit(1)

	}

	if models.C.Export_Directories.Archive == "" {

		l.Logger(l.ERROR, "Missing 'export_directories.archive'.")
		os.Exit(1)

	}

	if models.C.HA_Proxy.Enabled == true {

		if models.C.HA_Proxy.Port == 0 {

			l.Logger(l.ERROR, "Missing 'haproxy.port'.")
			os.Exit(1)

		}

	}

}

func Monitor_Config(bearerToken string, config_json string) {

	var currentConfigJSON string

	for {

		/* Do sleep first, since we just got spawned */

		time.Sleep(models.Env.CONFIG_SLEEP)

		currentConfigJSON, bearerToken = GetConfigJSON(bearerToken)

		models.ConfigMu.RLock()
		changed := currentConfigJSON != models.C.JSON
		models.ConfigMu.RUnlock()

		if changed {

			l.Logger(l.NOTICE, "Configuration update detected.  Loading new configurations")

			Load_Config(currentConfigJSON)

		}

		/* DEBUG: option for "sleep" to show loop */
		//l.Logger(l.DEBUG, "Completed Loop")

	}

}
