package nchc

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	ovenCenterY = 0.0
	ovenCenterX = 484.0
	ovenCenterZ = 183.0
)

// DataMap data map
type DataMap struct {
	Title     string    `json:"title"`
	Timestamp int64     `json:"timestamp"`
	MaxValue  float64   `json:"maxValue"`
	MinValue  float64   `json:"minValue"`
	XNum      int       `json:"xNum"`
	YNum      int       `json:"yNum"`
	Data      []float64 `json:"data"`
}

// NewDataMap new DataMap
func NewDataMap() *DataMap {
	return &DataMap{
		Data: make([]float64, 0),
	}
}

// SensorData sensor data
type SensorData struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Z     float64 `json:"z"`
	Value float64 `json:"value"`
	Valid bool    `json:"valid"`
	Type  string  `json:"type"` // sensor type could be nchc or aidc
}

// NewSensorData new SensorData
func NewSensorData() *SensorData {
	return &SensorData{
		ID:    "",
		Name:  "",
		X:     0,
		Y:     0,
		Z:     0,
		Value: 0,
		Valid: false,
		Type:  "",
	}
}

func (s *SensorData) coordinate() {
	// adjust coordinate  based on the oven center
	s.X = s.X - ovenCenterX
	s.Y = s.Y - ovenCenterY
	s.Z = s.Z - ovenCenterZ
}

func getValueRange(s []*SensorData) (float64, float64) {
	if len(s) == 0 {
		return 0, 0
	}

	vMin := s[0].Value
	vMax := s[0].Value

	for _, v := range s {
		if v.Value > vMax {
			vMax = v.Value
		}
		if v.Value < vMin {
			vMin = v.Value
		}
	}

	return vMin, vMax
}

func takeLayoutFile(name string) string {
	defaultFile := "default.sensor.csv"

	s := ""
	dirName := strings.ToLower(name)
	if len(dirName) >= 2 {
		s = dirName[0:2]
	}

	s = s + ".sensor.csv"
	// check if layout exists
	if _, err := os.Stat(filepath.Join(configPath, s)); os.IsNotExist(err) {
		log.Printf("sensor layout config [%s] doesn't exist", s)
		s = defaultFile
	}

	return s
}

// use second timestamp in string to read sensing data with layout
func mapSensingData(name, timestamp string) ([]*SensorData, error) {
	// t := fmt.Sprintf("%v", timestamp)
	data, err := readDataFromFile(name, timestamp)
	if err != nil {
		log.Printf("failed to read sensing data: %s", err)
		return nil, err
	}

	// read sensor layout config
	// layout := "config/a.layout.csv"
	layoutFile := takeLayoutFile(name)
	layout := filepath.Join(configPath, layoutFile)
	sensors, err := readSensorLayout(layout)
	if err != nil {
		log.Printf("failed to read sensor layout: %s", err)
		return nil, err
	}

	vv0, vv1 := GetValidValueRange()

	// Integrate sensing data and sensor layuout
	for i, s := range sensors {
		key := s.ID
		value, err := strconv.ParseFloat(data[key], 64)
		if err != nil {
			continue
		}

		// assign sensing value
		sensors[i].Value = value
		sensors[i].Valid = true

		if value < vv0 || value > vv1 {
			sensors[i].Valid = false
		}
	}

	// remove invalid sensors
	results := make([]*SensorData, 0)
	for _, s := range sensors {
		if s.Valid {
			results = append(results, s)
		}
	}

	// handle aidc sensor
	hasCuringData := true
	curingFile := filepath.Join(dataSourcePath, name, dataSubTmpDir, dataFileName2)
	if _, err := os.Stat(curingFile); os.IsNotExist(err) {
		hasCuringData = false
	}

	if hasCuringData {
		curingData, err := readCuringSensorData(name, data)
		if err != nil {
			log.Printf("failed to load curing data in %s: %s", name, err)
		}
		for _, s := range curingData {
			// fmt.Printf("[%s] (%s): %v (%v) <%f,%f,%f>\n", s.ID, s.Name, s.Value, s.Valid, s.X, s.Y, s.Z)
			results = append(results, s)
		}
	}

	// test result
	// for _, s := range results {
	// 	fmt.Printf("<%s> [%s] (%s): %.2f (%v) <%.1f,%.1f,%.1f>\n", s.Type, s.ID, s.Name, s.Value, s.Valid, s.X, s.Y, s.Z)
	// }

	return results, nil
}

// id should be timestamp in string
// if id is empty, take the last record
func readDataFromFile(name, id string) (map[string]string, error) {
	data := make(map[string]string)

	sourceFile := filepath.Join(dataSourcePath, name, dataSubTmpDir, dataFileName)
	// check if new curing data exists
	sourceFile2 := filepath.Join(dataSourcePath, name, dataSubTmpDir, dataFileName2)
	if _, err := os.Stat(sourceFile2); !os.IsNotExist(err) {
		// the file exists, use it as new source file
		sourceFile = sourceFile2
	}

	inFile, err := os.Open(sourceFile)
	if err != nil {
		return nil, err
	}

	r := csv.NewReader(inFile)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	var headers []string
	lastIndex := len(records) - 1
	timestampIndex := -1 // timestamp index in headers
	for i, v := range records {
		if i == 0 { // header
			headers = v
			timestampIndex = findArrayIndex(v, "timestamp")
			continue
		}
		if timestampIndex < 0 { // invalid header
			continue
		}

		// by default, timestamp is in index 0
		if id == "" {
			if i < lastIndex { // try to find the last record
				continue
			}
		} else if id != v[timestampIndex] {
			continue
		}

		// data found
		for j, k := range headers {
			// data[k] = v[j]
			data[k] = strings.TrimSpace(v[j])
		}
		return data, nil
	}

	// not found
	// return data, nil
	return nil, fmt.Errorf("no available data")
}

func readSensorLayout(sourceFile string) ([]*SensorData, error) {
	data := make([]*SensorData, 0)

	inFile, err := os.Open(sourceFile)
	if err != nil {
		return nil, err
	}

	r := csv.NewReader(inFile)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	for i, v := range records {
		if i == 0 { // header
			continue
		}

		if v[0] == "" {
			continue
		}

		e := NewSensorData()
		e.ID = v[0]
		e.Name = v[1]

		tValue, err := strconv.ParseFloat(v[2], 64)
		if err != nil {
			return nil, err
		}
		e.X = tValue

		tValue, err = strconv.ParseFloat(v[3], 64)
		if err != nil {
			return nil, err
		}
		e.Y = tValue

		tValue, err = strconv.ParseFloat(v[4], 64)
		if err != nil {
			return nil, err
		}
		e.Z = tValue
		e.Type = "nchc"

		e.coordinate()
		data = append(data, e)
	}

	return data, nil
}
