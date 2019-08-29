package nchc

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo"
)

const (
	// JSONIndent indent string for JSON pretty
	JSONIndent = "  "
)

/*
	Reading Data Source
*/

// ReadDataSources read data source list
func ReadDataSources(c echo.Context) error {
	res := NewDataSourceList()
	ReadDataSourceFromDir(res)
	return c.JSONPretty(http.StatusOK, res, JSONIndent)
}

/*
	Reading Records
*/

func genSyncFile(name string) error {
	// buf := new(bytes.Buffer)
	sDir := filepath.Join(GetVariable("dataSourcePath"), name)
	tDir := filepath.Join(GetVariable("dataSourcePath"), name, dataSubTmpDir)
	// args := []string{sDir, tDir}
	// sName := "parseRawTemperature"

	// cmd1 := "parse-temperature-data.rb"
	// args1 := []string{sDir, tDir}
	cmd1 := "ruby"
	args1 := []string{filepath.Join(scriptPath, "parse-temperature-data.rb"), sDir, tDir}
	out1, err1 := runScript(cmd1, args1)

	if err1 != nil {
		log.Printf("failed to run script[%s]: %s", cmd1, err1)
		fmt.Printf("%s outputs:\n%s\n", cmd1, out1)
		return err1
	}

	// cmd2 := "sync-temperature-data.rb"
	// args2 := []string{tDir, tDir}
	cmd2 := "ruby"
	args2 := []string{filepath.Join(scriptPath, "sync-temperature-data.rb"), tDir, tDir}
	out2, err2 := runScript(cmd2, args2)
	if err2 != nil {
		log.Printf("failed to run script[%s]: %s", cmd2, err2)
		fmt.Printf("%s outputs:\n%s\n", cmd2, out2)
		return err1
	}

	return nil
}

func genSyncFile2(name string) error {
	sDir := filepath.Join(GetVariable("dataSourcePath"), name)
	tDir := filepath.Join(GetVariable("dataSourcePath"), name, dataSubTmpDir)

	// this script will parse raw curing data and generate curing-data.csv
	cmd1 := "ruby"
	args1 := []string{filepath.Join(scriptPath, "parse-curing-data.rb"), sDir, tDir}
	out1, err1 := runScript(cmd1, args1)

	if err1 != nil {
		log.Printf("failed to run script[%s]: %s", cmd1, err1)
		fmt.Printf("%s outputs:\n%s\n", cmd1, out1)
		return err1
	}

	// this script will combine curing-data.csv and sync-output-plot.csv
	cmd2 := "ruby"
	args2 := []string{filepath.Join(scriptPath, "combine-curing.rb"), tDir, tDir}
	out2, err2 := runScript(cmd2, args2)
	if err2 != nil {
		log.Printf("failed to run script[%s]: %s", cmd2, err2)
		fmt.Printf("%s outputs:\n%s\n", cmd2, out2)
		return err1
	}

	return nil
}

func genFanCurrentFile(name string, dataFileName string) ([]byte, error) {
	sDir := filepath.Join(GetVariable("dataSourcePath"), name)
	tDir := filepath.Join(GetVariable("dataSourcePath"), name, dataSubTmpDir)
	cmd := "python3"
	args := []string{filepath.Join(scriptPath, "analysisFanCurrent.py"), name, dataFileName, sDir, tDir, "./model"}
	out, err := runScript(cmd, args)
	if err != nil {
		log.Printf("failed to run script[%s]: %s", cmd, err)
		fmt.Printf("%s outputs:\n%s\n", cmd, out)
		return out, err
	}
	return out, nil
}

func genHeaterCurrentFile(name string, dataFileName string) ([]byte, error) {
	sDir := filepath.Join(GetVariable("dataSourcePath"), name)
	tDir := filepath.Join(GetVariable("dataSourcePath"), name, dataSubTmpDir)
	cmd := "python3"
	args := []string{filepath.Join(scriptPath, "analysisHeaterCurrent.py"), name, dataFileName, sDir, tDir, "./model"}
	out, err := runScript(cmd, args)
	if err != nil {
		log.Printf("failed to run script[%s]: %s", cmd, err)
		fmt.Printf("%s outputs:\n%s\n", cmd, out)
		return out, err
	}
	return out, nil
}

// ReadRecords read records from a data source
func ReadRecords(c echo.Context) error {
	name := c.Param("name")
	datatype := c.QueryParam("type")
	log.Printf("Parsing data source: %s", name)
	if datatype != "" {
		log.Printf("Handling data type: %s", datatype)
	}

	switch datatype {
	case "fancurrentmodel":
		fallthrough
	case "fancurrent":
		return ReadRecordsForCurrent(c)
	case "heatercurrentmodel":
		fallthrough
	case "heatercurrent":
		fallthrough
	case "heatercurrentmodel-1":
		fallthrough
	case "heatercurrent-1":
		fallthrough
	case "heatercurrentmodel-2":
		fallthrough
	case "heatercurrent-2":
		return ReadRecordsForCurrent(c)
	default:
		return ReadRecordsForTemp(c)
	}
}

// ReadRecordsForTemp read records from a data source for temperature
func ReadRecordsForTemp(c echo.Context) error {
	name := c.Param("name")
	// check if the file exists
	dataSourcePath := GetVariable("dataSourcePath")
	dataFileName := GetVariable("dataFileName")
	sourceFile := filepath.Join(dataSourcePath, name, dataSubTmpDir, dataFileName)
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		// the file does not exist
		// call scripts to generate the file
		err = genSyncFile(name)
		if err != nil {
			msg := fmt.Sprintf("failed to generate %s : %s", dataFileName, err)
			return echo.NewHTTPError(http.StatusBadRequest, msg)
		}
	}

	// check if the new plot file exists
	dataFileName2 := GetVariable("dataFileName2")
	sourceFile2 := filepath.Join(dataSourcePath, name, dataSubTmpDir, dataFileName2)
	if _, err := os.Stat(sourceFile2); os.IsNotExist(err) {
		// the file does not exist
		// call scripts to generate the file
		err = genSyncFile2(name)
		if err != nil {
			msg := fmt.Sprintf("failed to generate %s : %s", dataFileName2, err)
			return echo.NewHTTPError(http.StatusBadRequest, msg)
		}
	}

	res := NewEventTimeList()
	_, err := ReadEventFromFile(name, res)
	if err != nil {
		msg := fmt.Sprintf("failed to fetch the event list: %s", err)
		return echo.NewHTTPError(http.StatusBadRequest, msg)
	}

	// filter some records
	nRes := NewEventTimeList()
	n := GetRecordOmitNum()
	for i, v := range res.Data {
		if i%n == 0 {
			nRes.Data = append(nRes.Data, v)
		}
	}

	return c.JSONPretty(http.StatusOK, nRes, JSONIndent)
}

// ReadRecordsForCurrent read records from a data source for temperature
func ReadRecordsForCurrent(c echo.Context) error {
	name := c.Param("name")
	datatype := c.QueryParam("type")
	// check if the file exists
	dataSourcePath := GetVariable("dataSourcePath")
	dataFileName := "fancurrent-output"
	fileType := ".csv"
	if datatype == "fancurrentmodel" || datatype == "heatercurrentmodel" || datatype == "heatercurrentmodel-1" || datatype == "heatercurrentmodel-2" {
		fileType = ".json"
	}

	// call scripts to generate the file
	if datatype == "fancurrentmodel" || datatype == "fancurrent" {
		_, err := genFanCurrentFile(name, dataFileName)
		if err != nil {
			msg := fmt.Sprintf("failed to generate %s : %s", dataFileName, err)
			return echo.NewHTTPError(http.StatusBadRequest, msg)
		}
	} else {
		dataFileName = "heatercurrent-output"
		strs := strings.Split(datatype, "-")
		if len(strs) > 1 {
			dataFileName += "-" + strs[1]
		}
		_, err := genHeaterCurrentFile(name, dataFileName)
		if err != nil {
			msg := fmt.Sprintf("failed to generate %s : %s", dataFileName, err)
			return echo.NewHTTPError(http.StatusBadRequest, msg)
		}
	}

	sourceFile := filepath.Join(dataSourcePath, name, dataSubTmpDir, dataFileName+fileType)
	return c.File(sourceFile)
}

/*
	Reading Images
*/

// LoadOvenImages load oven images
func LoadOvenImages(c echo.Context) error {
	imgReq := new(LoadOvenImagesRequest)
	if err := c.Bind(&imgReq); err != nil {
		return err
	}

	r, err := FunLoadOvenImages(imgReq)
	if err != nil {
		msg := fmt.Sprintf("failed to generate images: %s", err)
		return echo.NewHTTPError(http.StatusBadRequest, msg)
	}
	// fmt.Printf("loadImageResponse: %v\n", r)

	return c.JSONPretty(http.StatusOK, r, JSONIndent)
	// return c.JSONPretty(http.StatusOK, u, JSONIndent)
}
