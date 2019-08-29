package nchc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo"
)

// LoadVibrationImagesRequest load vibration images
type LoadVibrationImagesRequest struct {
	SourceName string  `json:"sourceName"`
	AxisName   string  `json:"axisName"`
	MethodName string  `json:"methodName"`
	TypeName   *string `json:"typeName"`
}

// LoadVibrationImagesResponse response
type LoadVibrationImagesResponse struct {
	ImgSFFT      string `json:"imgSFFT"`
	AnomalyScore string `json:"anomalyScore"`
}

// LoadOvenImagesRequest load oven images
type LoadOvenImagesRequest struct {
	SourceName      string  `json:"sourceName"`
	Timestamp       int64   `json:"timestamp"`
	HeatColorMethod string  `json:"heatColorMethod"`
	CenterX         float64 `json:"centerX"`
	CenterY         float64 `json:"centerY"`
	CenterZ         float64 `json:"centerZ"`
	ReferenceLine   string  `json:"refLine"`
	ImgCompareRange int64   `json:"imgCompareRange"`
	MinValue        float64 `json:"minValue"` // min temperature for display
	MaxValue        float64 `json:"maxValue"` // max temperature for display
}

// LoadOvenImagesResponse response
type LoadOvenImagesResponse struct {
	MinValue    float64       `json:"minValue"`
	MaxValue    float64       `json:"maxValue"`
	Timestrings []string      `json:"timestrings"`
	Timestamps  []int64       `json:"timestamps"` // refer to ImgYZ, ImgXZ, ImgXY, ImgTop, ImgMiddle, ImgBottom
	ImgYZ       []string      `json:"imgYZ"`
	ImgXZ       []string      `json:"imgXZ"`
	ImgXY       []string      `json:"imgXY"`
	ImgTop      []string      `json:"imgTop"`
	ImgMiddle   []string      `json:"imgMiddle"`
	ImgBottom   []string      `json:"imgBottom"`
	Sensors     []*SensorData `json:"sensors"`
	Levels      []float64     `json:"levels"`
}

func randomString() string {
	length := 8
	const charset = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}

	return string(b)
}

func genOvenImages(s string, sensors []*SensorData, imgReq *LoadOvenImagesRequest, imgRes *LoadOvenImagesResponse) error {
	imgPath := cacheImagePath
	// clean old images
	removeOvenImages(imgPath)

	imgName := filepath.Join(imgPath, s+"-yz.png")
	webImgName := filepath.Join(webImagePath, s+"-yz.png")
	createImageYZ(imgName, sensors, imgReq)
	imgRes.ImgYZ = append(imgRes.ImgYZ, webImgName)

	imgName = filepath.Join(imgPath, s+"-xz.png")
	webImgName = filepath.Join(webImagePath, s+"-xz.png")
	createImageXZ(imgName, sensors, imgReq)
	imgRes.ImgXZ = append(imgRes.ImgXZ, webImgName)

	imgName = filepath.Join(imgPath, s+"-xy.png")
	webImgName = filepath.Join(webImagePath, s+"-xy.png")
	createImageXY(imgName, sensors, imgReq)
	imgRes.ImgXY = append(imgRes.ImgXY, webImgName)

	rackData, err := readRackConfig()
	if err != nil {
		return nil
	}

	imgName = filepath.Join(imgPath, s+"-top")
	webImgName = filepath.Join(webImagePath, s+"-top.png")
	// don't draw reference line for rack images
	imgReq.ReferenceLine = ""
	imgReq.CenterZ = rackData.LevelVals[2]
	//createImageXY(imgName, sensors, imgReq)
	createImageXYFromType(imgName+"-nchc", sensors, imgReq, "nchc")
	createImageXYFromType(imgName+"-aidc", sensors, imgReq, "aidc")
	imgRes.ImgTop = append(imgRes.ImgTop, webImgName)

	imgName = filepath.Join(imgPath, s+"-middle")
	webImgName = filepath.Join(webImagePath, s+"-middle.png")
	imgReq.CenterZ = rackData.LevelVals[1]
	//createImageXY(imgName, sensors, imgReq)
	createImageXYFromType(imgName+"-nchc", sensors, imgReq, "nchc")
	createImageXYFromType(imgName+"-aidc", sensors, imgReq, "aidc")
	imgRes.ImgMiddle = append(imgRes.ImgMiddle, webImgName)

	imgName = filepath.Join(imgPath, s+"-bottom")
	webImgName = filepath.Join(webImagePath, s+"-bottom.png")
	imgReq.CenterZ = rackData.LevelVals[0]
	//createImageXY(imgName, sensors, imgReq)
	createImageXYFromType(imgName+"-nchc", sensors, imgReq, "nchc")
	createImageXYFromType(imgName+"-aidc", sensors, imgReq, "aidc")
	imgRes.ImgBottom = append(imgRes.ImgBottom, webImgName)

	return nil
}

func removeOvenImages(imgPath string) error {
	files, err := ioutil.ReadDir(imgPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		fName := file.Name()
		// fmt.Println(file.Name())
		if strings.HasSuffix(fName, "-yz.png") || strings.HasSuffix(fName, "-xz.png") || strings.HasSuffix(fName, "-xy.png") ||
			strings.HasSuffix(fName, "-top.png") || strings.HasSuffix(fName, "-middle.png") || strings.HasSuffix(fName, "-bottom.png") {
			fName = filepath.Join(imgPath, fName)
			// take file time
			info, err := os.Stat(fName)
			if err != nil {
				log.Printf("failed to get image stat: %s", err)
			}
			th := time.Minute * -10 // threshold to remove images
			t0 := time.Now().Add(th)
			if t0.After(info.ModTime()) { // the image is out of life time
				// fmt.Printf("X %s time: %s\n", fName, info.ModTime())
				err = os.Remove(fName)
				if err != nil {
					log.Printf("failed to remove image: %s", err)
				}
			} else { // the image is in life time
				// fmt.Printf("O %s time: %s\n", fName, info.ModTime())
			}
		}
	}

	return nil
}

// FunLoadOvenImages load oven images
func FunLoadOvenImages(imgReq *LoadOvenImagesRequest) (*LoadOvenImagesResponse, error) {
	s := fmt.Sprintf("%v", imgReq.Timestamp)
	name := imgReq.SourceName

	// take full event list
	eventList := NewEventTimeList()
	_, err := ReadEventFromFile(name, eventList)
	if err != nil {
		return nil, err
	}

	index := 0
	for index = 0; index < len(eventList.Data); index++ {
		if fmt.Sprintf("%v", eventList.Data[index].Timestamp) == s {
			break
		}
	}
	if index >= len(eventList.Data) {
		return nil, fmt.Errorf("invalid timestamp")
	}

	timeList := make([]int64, 0)
	timeStringList := make([]string, 0)
	// append previous event
	if imgReq.ImgCompareRange > 0 && index-int(imgReq.ImgCompareRange) >= 0 {
		t := eventList.Data[index-int(imgReq.ImgCompareRange)].Timestamp
		timeList = append(timeList, t)
		timeStringList = append(timeStringList, time.Unix(t, 0).String())
	}
	timeList = append(timeList, imgReq.Timestamp)
	timeStringList = append(timeStringList, time.Unix(imgReq.Timestamp, 0).String())
	// append next event
	if imgReq.ImgCompareRange > 0 && index+int(imgReq.ImgCompareRange) < len(eventList.Data) {
		t := eventList.Data[index+int(imgReq.ImgCompareRange)].Timestamp
		timeList = append(timeList, t)
		timeStringList = append(timeStringList, time.Unix(t, 0).String())
	}

	sensors, err := mapSensingData(name, s)
	if err != nil {
		return nil, err
	}
	vMin, vMax := getValueRange(sensors)

	// check temperature range
	for _, t := range timeList {
		tmpSensors, err := mapSensingData(name, fmt.Sprintf("%v", t))
		if err != nil {
			return nil, err
		}
		tmpMin, tmpMax := getValueRange(tmpSensors)
		if tmpMin < vMin {
			vMin = tmpMin
		}
		if tmpMax > vMax {
			vMax = tmpMax
		}
	}
	if imgReq.MaxValue == 0 && imgReq.MinValue == 0 {
		imgReq.MaxValue = vMax
		imgReq.MinValue = vMin
	}

	// generate images
	imgRes := &LoadOvenImagesResponse{
		MinValue:    vMin,
		MaxValue:    vMax,
		ImgYZ:       make([]string, 0),
		ImgXZ:       make([]string, 0),
		ImgXY:       make([]string, 0),
		ImgTop:      make([]string, 0),
		ImgMiddle:   make([]string, 0),
		ImgBottom:   make([]string, 0),
		Sensors:     sensors,
		Timestamps:  timeList,
		Timestrings: timeStringList,
	}
	for _, t := range timeList {
		ts := fmt.Sprintf("%v", t)
		tmpSensors, err := mapSensingData(name, ts)
		if err != nil {
			return nil, err
		}
		tmpImgReq := *imgReq
		// add a random id to file name to handle cache refresh
		err = genOvenImages(s+"-"+randomString(), tmpSensors, &tmpImgReq, imgRes)
		if err != nil {
			return nil, err
		}
	}

	rackData, err := readRackConfig()
	if err != nil {
		return nil, err
	}
	imgRes.Levels = rackData.LevelVals

	return imgRes, nil
}

func genVibrationTime(name string) ([]byte, error) {
	sDir := filepath.Join(GetVariable("dataSourcePath"), name)
	cmd1 := "python3"
	args1 := []string{filepath.Join(scriptPath, "gen-vibration-time.py"), sDir}
	out1, err1 := runScript(cmd1, args1)
	if err1 != nil {
		log.Printf("failed to run script[%s]: %s", cmd1, err1)
		fmt.Printf("%s outputs:\n%s\n", cmd1, out1)
		return nil, err1
	}
	return out1, nil
}

func genSFFTImages(curringName, axis, startTime, endTime, receta, datatype string) error {
	sDir := filepath.Join(GetVariable("vibrationPath"))
	cmd1 := "python3"
	args1 := []string{filepath.Join(scriptPath, "gen-SFFT-image.py"), sDir, curringName, axis, startTime, endTime, receta, datatype}
	out, err := runScript(cmd1, args1)
	if err != nil {
		log.Printf("failed to run script[%s]: %s", cmd1, err)
		fmt.Printf("%s outputs:\n%s\n", cmd1, out)
		return err
	}
	return nil
}

func genVibrationScore(curringName, axis, datatype string) (string, error) {
	sfftsourceFile := filepath.Join(GetVariable("vibrationPath"), GetVariable("dataFileName3"))
	cmd1 := "python3"
	args1 := []string{filepath.Join(scriptPath, "vibration-anomaly-score.py"), curringName, axis, sfftsourceFile, datatype}
	out, err := runScript(cmd1, args1)
	if err != nil {
		log.Printf("failed to run script[%s]: %s", cmd1, err)
		fmt.Printf("%s outputs:\n%s\n", cmd1, out)
		// if recipe model doesn't exist
		return "N", err
	}
	score := fmt.Sprintf("%s", out)
	return score, nil
}

// LoadVibrationImages compute SFFT and load images
func LoadVibrationImages(c echo.Context) error {
	imgReq := new(LoadVibrationImagesRequest)
	if err1 := c.Bind(&imgReq); err1 != nil {
		return err1
	}
	fmt.Printf("get LoadVibrationImages request : %+v\n", imgReq)
	if imgReq.TypeName != nil {
		fmt.Printf("Type: %s\n", *imgReq.TypeName)
	}
	//check images exist
	fileName := imgReq.SourceName + "-SFFT-" + imgReq.AxisName
	if imgReq.TypeName != nil {
		fileName = fileName + "-" + *imgReq.TypeName
	} else {
		tmp := "fan"
		imgReq.TypeName = &tmp
	}
	fileName = fileName + ".png"
	imgsourceFile := filepath.Join(GetVariable("cacheImagePath"), fileName)
	websourceFile := filepath.Join(GetVariable("webImagePath"), fileName)
	sfftsourceFile := filepath.Join(GetVariable("vibrationPath"), GetVariable("dataFileName3"))
	score := "0"

	if _, err := os.Stat(imgsourceFile); os.IsNotExist(err) {
		//fmt.Printf("%s not exist\n", sourceFile)
		out1, err2 := genVibrationTime(imgReq.SourceName)
		if err2 != nil {
			msg := fmt.Sprintf("failed to generate VibrationTime: %s", err2)
			return echo.NewHTTPError(http.StatusBadRequest, msg)
		}
		var cutTime map[string]interface{}
		json.Unmarshal(out1, &cutTime)

		receta := fmt.Sprintf("%v", cutTime["receta"])
		startTime := fmt.Sprintf("%v", cutTime["start_time"])
		endTime := fmt.Sprintf("%v", cutTime["end_time"])

		// generate vibration SFFT images and save signal to file
		err3 := genSFFTImages(imgReq.SourceName, imgReq.AxisName, startTime, endTime, receta, *imgReq.TypeName)
		if err3 != nil {
			msg := fmt.Sprintf("failed to generate images: %s", err3)
			return echo.NewHTTPError(http.StatusBadRequest, msg)
		}
	}
	if _, err := os.Stat(sfftsourceFile); !os.IsNotExist(err) {
		anomalyScore, err2 := genVibrationScore(imgReq.SourceName, imgReq.AxisName, *imgReq.TypeName)
		if err2 != nil {
			//msg := fmt.Sprintf("failed to generate VibrationScore: %s", err2)
			//return echo.NewHTTPError(http.StatusBadRequest, msg)
		}
		score = anomalyScore
	}

	//images exist
	r := &LoadVibrationImagesResponse{
		ImgSFFT:      websourceFile,
		AnomalyScore: score,
	}

	return c.JSONPretty(http.StatusOK, r, JSONIndent)
}
