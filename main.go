package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dslipak/pdf"
)

const (
	PitstopLetter = "P"
	FirstSector   = 1
	SecondSector  = 2
	ThirdSector   = 3
)

type LapInfo struct {
	Pit      bool
	Sectors  []SectorInfo
	FullTime time.Time
	NumOfLap int64
}

type SectorInfo struct {
	Time  uint64
	Speed float64
	Num   int
}

func main() {
	err := parsePDF("silverstone2024.pdf")
	if err != nil {
		panic(err)
	}
}

type RaceData map[string][]LapInfo

func parsePDF(filename string) error {
	pdfReader, err := pdf.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open .pdf file: %w", err)
	}

	raceData := make(RaceData)
	for i := 1; i <= pdfReader.NumPage(); i++ {
		page := pdfReader.Page(i)
		sourceData, err := getSourceDataFromPage(page)
		if err != nil {
			return fmt.Errorf("failed to get data from page: %d. error: %w", i, err)
		}

		err = raceData.parseRacerData(sourceData)
		if err != nil {
			return fmt.Errorf("failed to parse racer data. page: %d, err: %w", i, err)
		}
	}
	err = raceData.writeToFile()
	if err != nil {
		return fmt.Errorf("failed to write data into file: %w", err)
	}

	return nil
}

func (r RaceData) writeToFile() error {
	result, err := json.MarshalIndent(r, "", "	")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	file, err := os.Create("result.json")
	if err != nil {
		return fmt.Errorf("failed to create result.json file: %w", err)
	}

	_, err = file.WriteString(string(result))
	if err != nil {
		return fmt.Errorf("failed to write data into file: %w", err)
	}

	return nil
}

func (r RaceData) parseRacerData(sourceData []string) error {
	racerName := "unknown"
	for i := 8; i < len(sourceData); i++ {
		copyIndex := i
		switch {
		case isLapNumber(sourceData, i):
			lapData, err := setLapData(sourceData, &i)
			if err != nil {
				return fmt.Errorf("failed to set lap data: %w", err)
			}

			r[racerName] = append(r[racerName], *lapData)
			continue
		case isRacerNumber(sourceData, &i):
			racerName = sourceData[copyIndex+1]
			r[racerName] = r["unknown"]
			delete(r, "unknown")
			continue
		}
		break
	}

	return nil
}

// data is not [i: i+9] cause we need check prev values
func setLapData(data []string, index *int) (*LapInfo, error) {
	k := *index
	lap, _ := strconv.ParseInt(data[k], 10, 64)
	lapInfo := &LapInfo{
		NumOfLap: lap,
	}

	var (
		startIndex int
		endIndex   int
	)
	switch {
	case data[*index+1] == PitstopLetter:
		*index += 7
		return &LapInfo{}, nil
	case isLapNumber(data, *index+8):
		speed, err := parseSpeed(data[k+2])
		if err != nil {
			return nil, fmt.Errorf("failed to parse speed on start grip. value: %s, index: %d, error: %w", data[k+2], k+2, err)
		}

		sectorData := SectorInfo{
			Speed: speed,
			Num:   FirstSector,
		}
		lapInfo.Sectors = append(lapInfo.Sectors, sectorData)

		startIndex = 3
		endIndex = 7
		*index += 7
	default:
		startIndex = 2
		endIndex = 7
		*index += 8
	}

	for i := startIndex; i < endIndex; i++ {
		sectorData := SectorInfo{}
		sectorTime, err := parseTime(data[i+k])
		if err != nil {
			return nil, fmt.Errorf("failed to parse sector time. value: %s, index: %d, error: %w", data[i+k], i+k, err)
		}

		speed, err := parseSpeed(data[i+k+1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse speed. value: %s, index: %d, error: %w", data[i+k+1], i+k+1, err)
		}

		sectorData.Speed = speed
		sectorData.Time = sectorTime
		lapInfo.Sectors = append(lapInfo.Sectors, sectorData)
		i++
	}

	return lapInfo, nil
}

func parseSpeed(value string) (float64, error) {
	speed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse speed. value: %s, error: %w", value, err)
	}

	return speed, nil
}

func isRacerNumber(value []string, index *int) bool {
	arrName := strings.Split(value[*index+1], " ")
	if len(arrName) == 2 && arrName[0] != "SECTOR" {
		*index += 11
		return true
	}

	return false
}

func isLapNumber(data []string, index int) bool {
	_, err := strconv.ParseInt(data[index], 10, 32)
	if err != nil {
		return false
	}

	// check racer name
	arrName := strings.Split(data[index+1], " ")
	if len(arrName) == 2 && arrName[0] != "SECTOR" {
		return false
	}

	isTime, _ := regexp.MatchString(`^(?:\d{1,3}:)?\d{2}[.]\d{3}$`, data[index-1])
	isStartTime, _ := regexp.MatchString(`^\d{2}[:]\d{2}[:]\d{2}$`, data[index-1])
	switch {
	case isTime, isStartTime, data[index-1] == "TIME":
		return true
	}

	return false
}

func getSourceDataFromPage(page pdf.Page) ([]string, error) {
	columns, err := page.GetTextByColumn()
	if err != nil {
		return nil, err
	}

	result := make([]string, 0)
	for _, col := range columns {
		for _, content := range col.Content {

			if len(content.S) == 0 {
				continue
			}

			result = append(result, content.S)
		}

	}

	return result, nil
}

func parseTime(value string) (result uint64, err error) {
	timeSlice := strings.Split(value, ":")

	var timeValue time.Time
	switch {
	// time = xx.xxx
	case len(timeSlice) == 1:
		timeValue, err = time.Parse("05.999", value)
		if err != nil {
			return 0, fmt.Errorf("failed to parse time by layout 05.999: %w", err)
		}
	// time = xx:xx.xxx
	case len(timeSlice) == 2:
		timeValue, err = time.Parse("4:05.999", value)
		if err != nil {
			return 0, fmt.Errorf("failed to parse time by layout 4:05.999: %w", err)
		}
	default:
		return 0, errors.New("unknown layout of time")
	}

	result += uint64(timeValue.Nanosecond() / 1e6)
	result += uint64(timeValue.Second() * 1e1)
	result += uint64(timeValue.Minute() * 1e1 * 60)

	return
}
