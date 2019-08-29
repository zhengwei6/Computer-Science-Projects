package nchc

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	// EnvVars environment variables
	EnvVars = map[string]string{
		"servicePort":     "NCHCservicePort",
		"configPath":      "NCHCconfigPath",
		"scriptPath":      "NCHCscriptPath",
		"dataSourcePath":  "NCHCdataSourcePath",
		"cacheImagePath":  "NCHCcacheImagePath",
		"cachePublicPath": "NCHCcachePublicPath",
		"webImagePath":    "NCHCwebImagePath",
		"dataFileName":    "NCHCdataFileName",
		"dataFileName2":   "NCHCdataFileName2",
		"dataFileName3":   "NCHCdataFileName3",
		"apiURL":          "NCHCapiURL",
		"apiUser":         "NCHCapiUser",
		"apiPassword":     "NCHCapiPassword",
		"minValidValue":   "NCHCminValidValue",
		"maxValidValue":   "NCHCmaxValidValue",
		"recordOmitNum":   "NCHCrecordOmitNum",
		"curingTCLayout":  "NCHCcuringTCLayout",
		"vibrationPath":   "NCHCvibrationPath",
	}
)

var (
	servicePort    = ""       // service port, default 3266, apiURL should also be updated if default port is changed
	configPath     = "config" // path of config directory
	scriptPath     = "script" // path of scripts directory
	dataSourcePath = "data"   // data source path, each data source should be one directory
	// cacheImagePath = "../public/images/"        // local image path for web service (dev)
	// cacheImagePath = "../build/images/"     // local image path for web service (release)
	cacheImagePath  = "web/images"           // local image path for web service (cloud)
	cachePublicPath = "web"                  // local web path (cloud)
	webImagePath    = "/images/"             // public image path for web service
	vibrationPath   = "vibrationData"        // vibration data path
	dataFileName    = "sync-output-plot.csv" // data file name for output (nchc sensor only)
	dataFileName2   = "sync-curing-plot.csv" // data file name for output (nchc sensor + aidc curing)
	dataFileName3   = "Select_Frequency.csv" // data file name for vibraiton frequency
	apiURL          = ""                     // api url, default url is http://localhost:3266
	apiUser         = ""                     // api user name
	apiPassword     = ""                     // api password
	minValidValue   = "0"                    // min valid temperature value
	maxValidValue   = "700"                  // max valid temperature value
	recordOmitNum   = "1"                    // omit some records to reduce result number
	curingTCLayout  = "thermalcouple.csv"    // layout config for thermal couple
	// config that will not be changed
	dataSubTmpDir  = "tmp"              // temp directory for data source
	rackConfigName = "rack-config.json" // rack config file in json format

	// config that will be updated automatically
	mCubeSizeX = 10.0
	mCubeSizeY = 10.0
	mCubeSizeZ = 10.0
)

func init() {
	LoadVariablesFromEnv()
	// handle service port and apiURL
	if servicePort != "" && apiURL == "" {
		apiURL = fmt.Sprintf("localhost:%s", servicePort)
	}

	opts := GetAllVariables()
	fmt.Printf("NCHC Variables:\n")
	for k, v := range opts {
		fmt.Printf(">> %s => %s\n", k, v)
	}

	GenEnvJS()
}

// GetVariable get variable
func GetVariable(s string) string {
	switch s {
	case "servicePort":
		return servicePort
	case "configPath":
		return configPath
	case "scriptPath":
		return scriptPath
	case "dataSourcePath":
		return dataSourcePath
	case "cacheImagePath":
		return cacheImagePath
	case "cachePublicPath":
		return cachePublicPath
	case "webImagePath":
		return webImagePath
	case "dataFileName":
		return dataFileName
	case "dataFileName2":
		return dataFileName2
	case "dataFileName3":
		return dataFileName3
	case "apiURL":
		return apiURL
	case "apiUser":
		return apiUser
	case "apiPassword":
		return apiPassword
	case "minValidValue":
		return minValidValue
	case "maxValidValue":
		return maxValidValue
	case "recordOmitNum":
		return recordOmitNum
	case "curingTCLayout":
		return curingTCLayout
	case "vibrationPath":
		return vibrationPath
	}
	return ""
}

// SetVariables set variables
func SetVariables(opts map[string]string) {
	if k, ok := opts["servicePort"]; ok {
		if k != "" {
			servicePort = k
		}
	}

	if k, ok := opts["configPath"]; ok {
		if k != "" {
			configPath = k
		}
	}

	if k, ok := opts["scriptPath"]; ok {
		if k != "" {
			scriptPath = k
		}
	}

	if k, ok := opts["dataSourcePath"]; ok {
		if k != "" {
			dataSourcePath = k
		}
	}

	if k, ok := opts["cacheImagePath"]; ok {
		if k != "" {
			cacheImagePath = k
		}
	}

	if k, ok := opts["cachePublicPath"]; ok {
		if k != "" {
			cachePublicPath = k
		}
	}

	if k, ok := opts["webImagePath"]; ok {
		if k != "" {
			webImagePath = k
		}
	}

	if k, ok := opts["dataFileName"]; ok {
		if k != "" {
			dataFileName = k
		}
	}

	if k, ok := opts["dataFileName2"]; ok {
		if k != "" {
			dataFileName2 = k
		}
	}

	if k, ok := opts["dataFileName3"]; ok {
		if k != "" {
			dataFileName3 = k
		}
	}

	if k, ok := opts["apiURL"]; ok {
		if k != "" {
			apiURL = k
		}
	}

	if k, ok := opts["apiUser"]; ok {
		if k != "" {
			apiUser = k
		}
	}

	if k, ok := opts["apiPassword"]; ok {
		if k != "" {
			apiPassword = k
		}
	}

	if k, ok := opts["minValidValue"]; ok {
		if k != "" {
			minValidValue = k
		}
	}

	if k, ok := opts["maxValidValue"]; ok {
		if k != "" {
			maxValidValue = k
		}
	}

	if k, ok := opts["recordOmitNum"]; ok {
		if k != "" {
			recordOmitNum = k
		}
	}

	if k, ok := opts["curingTCLayout"]; ok {
		if k != "" {
			curingTCLayout = k
		}
	}

	if k, ok := opts["vibrationPath"]; ok {
		if k != "" {
			vibrationPath = k
		}
	}
}

// LoadVariablesFromEnv load variable from env
func LoadVariablesFromEnv() {
	opts := make(map[string]string)
	for k, v := range EnvVars {
		tmp := os.Getenv(v)
		if tmp != "" {
			opts[k] = tmp
		}
	}
	SetVariables(opts)
}

// GetAllVariables get all variables
func GetAllVariables() map[string]string {
	opts := make(map[string]string)
	opts["servicePort"] = servicePort
	opts["configPath"] = configPath
	opts["scriptPath"] = scriptPath
	opts["dataSourcePath"] = dataSourcePath
	opts["cacheImagePath"] = cacheImagePath
	opts["cachePublicPath"] = cachePublicPath
	opts["webImagePath"] = webImagePath
	opts["dataFileName"] = dataFileName
	opts["dataFileName2"] = dataFileName2
	opts["dataFileName3"] = dataFileName3
	opts["apiURL"] = apiURL
	opts["apiUser"] = apiUser
	opts["apiPassword"] = apiPassword
	opts["minValidValue"] = minValidValue
	opts["maxValidValue"] = maxValidValue
	opts["recordOmitNum"] = recordOmitNum
	opts["curingTCLayout"] = curingTCLayout
	opts["vibrationPath"] = vibrationPath
	return opts
}

// GetValidValueRange get valid value range
func GetValidValueRange() (float64, float64) {
	v0 := 0.0
	v1 := 700.0
	v, err := strconv.ParseFloat(minValidValue, 64)
	if err == nil {
		v0 = v
	}
	v, err = strconv.ParseFloat(maxValidValue, 64)
	if err == nil {
		v1 = v
	}
	if v1 < v0 {
		v0 = 0.0
		v1 = 700.0
	}

	return v0, v1
}

// GetRecordOmitNum get record omit number
func GetRecordOmitNum() int {
	v, err := strconv.ParseInt(recordOmitNum, 10, 64)
	if err != nil {
		return 10
	}

	if v < 1 {
		return 1
	}

	return int(v)
}

// GenEnvJS generate env.js for react web
func GenEnvJS() error {
	// generate env.js on public path
	// API_URL: http://localhost:3266
	// API_USER:
	// API_PASSWORD:
	fName := "env.js"
	fPath := filepath.Join(cachePublicPath, fName)
	envContent := make([]string, 0)
	tmpURL := apiURL
	if tmpURL != "" {
		if !strings.HasPrefix(tmpURL, "http://") {
			tmpURL = "http://" + tmpURL
		}
		if strings.Contains(tmpURL, "localhost") {
			// Because web page is static and be mounted by same domain
			tmpURL = ""
		}
		envContent = append(envContent, fmt.Sprintf("API_URL: '%s'", tmpURL))
	}

	if apiUser != "" && apiPassword != "" {
		envContent = append(envContent, fmt.Sprintf("API_USER: '%s'", apiUser))
		envContent = append(envContent, fmt.Sprintf("API_PASSWORD: '%s'", apiPassword))
	}

	for i, v := range envContent {
		envContent[i] = "  " + v
	}

	data := "window.env = {\n"
	data += strings.Join(envContent, ",\n")
	data += "\n}\n"

	err := ioutil.WriteFile(fPath, []byte(data), 0600)
	if err != nil {
		log.Printf("failed to generate %s: %s", fPath, err)
	}

	return err
}
