package nchc

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// EventTime event time
type EventTime struct {
	Timestamp int64   `json:"timestamp"`
	Time      string  `json:"time"`
	Value     float64 `json:"value"`
	MaxValue  float64 `json:"maxValue"`
	MinValue  float64 `json:"minValue"`
}

// NewEventTime new EventTime
func NewEventTime() *EventTime {
	return &EventTime{}
}

// EventTimeList event time list
type EventTimeList struct {
	Data []*EventTime `json:"records"`
}

// NewEventTimeList new EventTimeList
func NewEventTimeList() *EventTimeList {
	return &EventTimeList{
		Data: make([]*EventTime, 0),
	}
}

func findArrayIndex(s []string, k string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == k {
			return i
		}
	}
	return -1
}

func findTemperatureIndices(ss []string) []int {
	a := make([]int, 0)

	r1, _ := regexp.Compile("^[0-9]-[0-9]$")
	r2, _ := regexp.Compile("^T[0-9]+$")

	for i, s := range ss {
		if r1.MatchString(s) || r2.MatchString(s) {
			a = append(a, i)
		}
	}

	return a
}

// ReadEventFromFile read event from file
func ReadEventFromFile(name string, res *EventTimeList) ([]string, error) {
	var headers []string

	sourceFile := filepath.Join(dataSourcePath, name, dataSubTmpDir, dataFileName)

	// check if new curing data exists
	sourceFile2 := filepath.Join(dataSourcePath, name, dataSubTmpDir, dataFileName2)
	if _, err := os.Stat(sourceFile2); !os.IsNotExist(err) {
		// the file exists, use it as new source file
		// fmt.Printf("using %s instaed of %s ...\n", sourceFile2, sourceFile)
		sourceFile = sourceFile2
	}

	inFile, err := os.Open(sourceFile)
	if err != nil {
		return nil, err
	}
	log.Printf("Reading %s", sourceFile)

	r := csv.NewReader(inFile)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	vv0, vv1 := GetValidValueRange()

	tempIndices := make([]int, 0) // indixes of temperature fields
	timestampIndex := -1          // timestamp index in headers
	valueIndex := -1              // value index in headres
	for i, v := range records {
		if i == 0 { // header
			headers = v
			timestampIndex = findArrayIndex(v, "timestamp")
			tempIndices = findTemperatureIndices(headers)
			valueIndex = findArrayIndex(v, "AMV")
			if valueIndex < 0 {
				valueIndex = findArrayIndex(v, "0-5")
			}
			if valueIndex < 0 {
				valueIndex = findArrayIndex(v, "1-1")
			}
			if valueIndex < 0 {
				valueIndex = findArrayIndex(v, "0-1")
			}
			if valueIndex < 0 && len(tempIndices) > 0 {
				valueIndex = tempIndices[0]
			}
			continue
		}

		if timestampIndex < 0 { // invalid header
			continue
		}
		newData := NewEventTime()
		newData.Value = vv0 - 1.0 // set an invalid value
		newData.MaxValue = vv0
		newData.MinValue = vv1

		// append timestamp
		timestamp, err := strconv.ParseInt(strings.TrimSpace(v[timestampIndex]), 10, 64)
		if err != nil {
			fmt.Printf("[WARN] invalid timestamp: %s\n", err)
			continue
		}
		newData.Timestamp = timestamp
		newData.Time = time.Unix(timestamp, 0).String()

		// append reference value
		if valueIndex >= 0 {
			tValue, err := strconv.ParseFloat(strings.TrimSpace(v[valueIndex]), 64)
			if err != nil {
				fmt.Printf("[WARN] invalid valueIndex: %s\n", err)
				continue
			}
			// check if the value is valid
			if tValue < vv0 || tValue > vv1 {
				fmt.Printf("[WARN] invalid value: %v\n", tValue)
				continue
			}
			newData.Value = tValue
			newData.MaxValue = tValue
			newData.MinValue = tValue
		}

		// take max value and min value
		for _, j := range tempIndices {
			tValue, err := strconv.ParseFloat(strings.TrimSpace(v[j]), 64)
			if err != nil {
				continue
			}
			// check if the value is valid
			if tValue < vv0 || tValue > vv1 {
				continue
			}
			if newData.MaxValue < tValue {
				newData.MaxValue = tValue
			}
			if newData.MinValue > tValue {
				newData.MinValue = tValue
			}
		}

		// confirm data
		if newData.MaxValue < newData.MinValue {
			newData.MaxValue = 0
			newData.MinValue = 0
		}
		if newData.Value < vv0 || newData.Value > vv1 {
			newData.Value = 0
		}

		res.Data = append(res.Data, newData)
	}
	return headers, nil
}
