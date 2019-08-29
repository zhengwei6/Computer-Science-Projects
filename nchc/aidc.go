package nchc

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// RackConfig rack config
type RackConfig struct {
	CubeSize  []float64 `json:"cube-size"` // cube size: x, y, z
	LevelVals []float64 `json:"level"`     // Z
	RowVals   []float64 `json:"row"`       // Y
	ColVals   []float64 `json:"col"`       // X
}

func readCuringSensorData(name string, data map[string]string) ([]*SensorData, error) {
	// read rack config
	rackData, err := readRackConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read rack config: %s", err)
	}

	// read curing sensor layout
	sourceFile := filepath.Join(dataSourcePath, name, curingTCLayout)
	// fmt.Printf("layout: %s\n", sourceFile)
	inFile, err := os.Open(sourceFile)
	if err != nil {
		// fmt.Printf("failed to open %s: %s\n", sourceFile, err)
		return nil, err
	}

	r := csv.NewReader(inFile)
	records, err := r.ReadAll()
	if err != nil {
		// fmt.Printf("failed to parse %s: %s\n", sourceFile, err)
		return nil, err
	}

	headerIndex := []int{-1, -1, -1, -1} // headers should be TC_ID, LEVEL, Row, Col
	vv0, vv1 := GetValidValueRange()
	curingData := make([]*SensorData, 0)
	for i, g := range records {
		if i == 0 { // header
			for j, k := range g {
				if strings.EqualFold(k, "TC_ID") {
					headerIndex[0] = j
				}
				if strings.EqualFold(k, "LEVEL") {
					headerIndex[1] = j
				}
				if strings.EqualFold(k, "Row") {
					headerIndex[2] = j
				}
				if strings.EqualFold(k, "Col") {
					headerIndex[3] = j
				}
			}

			for _, k := range headerIndex {
				if k < 0 {
					return nil, fmt.Errorf("incorrect header in %s", sourceFile)
				}
			}
			continue
		}

		// handle data
		vLevel, err := strconv.ParseInt(g[headerIndex[1]], 10, 64)
		if err != nil {
			return nil, err
		}

		vRow, err := strconv.ParseInt(g[headerIndex[2]], 10, 64)
		if err != nil {
			return nil, err
		}

		vCol, err := strconv.ParseInt(g[headerIndex[3]], 10, 64)
		if err != nil {
			return nil, err
		}

		tcids := strings.Split(g[headerIndex[0]], ",")
		if len(tcids) <= 0 {
			return nil, fmt.Errorf("invalid thermal couple id")
		}

		cValues := make([]float64, 0)
		for _, k := range tcids {
			if data[k] == "" {
				continue
			}

			v, err := strconv.ParseFloat(data[k], 64)
			if err != nil {
				log.Printf("failed to parse %s value : %s\n", k, err)
				continue
			}

			// omit invalid value
			if v < vv0 || v > vv1 {
				continue
			}

			cValues = append(cValues, v)
		}

		if len(cValues) == 0 { // no valid value
			continue
		}

		// take average value
		tValue := 0.0
		for _, v := range cValues {
			tValue += v
		}
		tValue /= float64(len(cValues))

		sData := NewSensorData()
		sData.ID = fmt.Sprintf("%s-TCG%02d", name, i)
		sData.Name = g[headerIndex[0]]
		sData.Type = "aidc"
		sData.Value = tValue
		// sData.X = float64(vCol)
		// sData.Y = float64(vRow)
		// sData.Z = float64(vLevel)
		// vLevel is from 1 to 3
		if vLevel > 0 && vLevel <= int64(len(rackData.LevelVals)) {
			sData.Z = rackData.LevelVals[vLevel-1]
		} else {
			log.Printf("invalid rack data: incorrect level value")
			continue
		}
		// vRow is from 1 to 3
		if vRow > 0 && vRow <= int64(len(rackData.RowVals)) {
			sData.Y = rackData.RowVals[vRow-1]
		} else {
			log.Printf("invalid rack data: incorrect row value")
			continue
		}
		// vCol is from 1 to 24
		if vCol > 0 && vCol <= int64(len(rackData.ColVals)) {
			sData.X = rackData.ColVals[vCol-1]
		} else {
			log.Printf("invalid rack data: incorrect col value")
			continue
		}
		sData.Valid = true
		if tValue < vv0 || tValue > vv1 {
			sData.Valid = false
		}
		curingData = append(curingData, sData)
	}

	return curingData, nil
}

func readRackConfig() (*RackConfig, error) {
	r := &RackConfig{}

	sourceFile := filepath.Join(configPath, rackConfigName)
	data, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, r)
	if err != nil {
		return nil, err
	}

	// shift X
	for i, x := range r.ColVals {
		r.ColVals[i] = x - ovenCenterX
	}

	// check cube size
	if len(r.CubeSize) != 3 {
		return nil, fmt.Errorf("invalid cube size in rack config")
	}

	mCubeSizeX = r.CubeSize[0]
	mCubeSizeY = r.CubeSize[1]
	mCubeSizeZ = r.CubeSize[2]

	return r, nil
}
