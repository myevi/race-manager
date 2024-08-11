package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dslipak/pdf"
)

const (
	LF               = 10  // Symbol \n
	PitstopSymbol    = "P" // Pit-stop
	FirstSector      = 1
	SecondSector     = 2
	ThirdSector      = 3
	firstLapSkipRows = 7
	lapSkipRows      = 8
)

type LapInfo struct {
	Pit      bool
	Sectors  []SectorInfo
	FullTime time.Time
	NumOfLap int64
}

type SectorInfo struct {
	Time  time.Time
	Speed float64
	Num   int
}

func main() {
	filename := flag.String("f", "silverstone2024.pdf", "file name to parse")
	flag.Parse()
	result, err := parsePDF(*filename)
	if err != nil {
		panic(err)
	}

	res, err := json.MarshalIndent(result, "", "	")
	if err != nil {
		panic(err)
	}

	file, err := os.Create("result.json")
	if err != nil {
		panic(err)
	}

	_, err = file.WriteString(string(res))
	if err != nil {
		panic(err)
	}

	fmt.Println("file successed parsed")
}

func parsePDF(filename string) (*map[string][]LapInfo, error) {
	pdfReader, err := pdf.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open .pdf file: %w", err)
	}

	reader, err := pdfReader.GetPlainText()
	if err != nil {
		return nil, fmt.Errorf("failed to get reader from pdf: %w", err)
	}

	rows, err := getRows(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to getting rows from reader: %w", err)
	}

	raceData := make(map[string][]LapInfo)
	racerName := "unknown"

	for i := 0; i < len(rows); i++ {
		if isLapNumber(rows, i) {
			// todo: temp skipping laps with pitstop
			if rows[i+1] == "P" {
				continue
			}

			lapData, rowsToSkip, err := setLapData(rows, i)
			if err != nil {
				return nil, fmt.Errorf("failed to set lap data: %w", err)
			}

			raceData[racerName] = append(raceData[racerName], *lapData)
			i += rowsToSkip
			continue
		}

		if isRacerNumber(rows[i]) {
			racerName = rows[i]
			raceData[racerName] = raceData["unknown"]
			i++
			continue
		}

		// it is end of file
		if rows[i] == "Race Sector Analysis" {
			racerName = "unknown"
		}
	}

	return &raceData, nil
}

func getRows(reader io.Reader) ([]string, error) {
	// todo: can optimaze copacity by one more parameter length in func
	res := make([]string, 0)
	var buf bytes.Buffer
	_, err := io.Copy(&buf, reader)
	if err != nil {
		return res, fmt.Errorf("failed to copy readers data to buffer: %w", err)
	}

	for {
		row, err := buf.ReadBytes(LF)
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, fmt.Errorf("failed to read row of bytes: %w", err)
		}

		// ":len(row) - 1" is removing "new line" symbol
		res = append(res, string(row[:len(row)-1]))
	}

	return res, nil
}

func setLapData(rows []string, index int) (*LapInfo, int, error) {
	lap, _ := strconv.ParseInt(rows[index], 10, 64)
	result := &LapInfo{
		NumOfLap: lap,
	}
	var sectorData SectorInfo
	// result.WasPit = rows[index+1] == PitstopSymbol
	arrIndex := index
	// if result.WasPit {
	// 	pitTime, err := parseTime(rows[index+2])
	// 	if err != nil {
	// 		return nil, 0, fmt.Errorf("failed to parse time on pit lap. value: %s, index: %d, error: %w", rows[arrIndex+1], arrIndex+1, err)
	// 	}

	// 	sectorData = SectorInfo{
	// 		Time: pitTime,
	// 		Num:   FirstSector,
	// 	}

	// 	result.Sectors = append(result.Sectors, sectorData)
	// 	arrIndex += 3
	// 	for i := 0; i < 2; i++ {
	// 		arrIndex += i * 2
	// 		sectorData = SectorInfo{}
	// 		sectorTime, err := parseTime(rows[arrIndex])
	// 		if err != nil {
	// 			return nil, 0, fmt.Errorf("failed to parse sector time on pit lap. value: %s, index: %d, error: %w", rows[arrIndex], arrIndex, err)
	// 		}

	// 		speed, err := parseSpeed(rows[arrIndex+1])
	// 		if err != nil {
	// 			return nil, 0, fmt.Errorf("failed to parse speed on pit lap. value: %s, index: %d, error: %w", rows[arrIndex+1], arrIndex+1, err)
	// 		}

	// 		sectorData.Speed = speed
	// 		sectorData.Time = sectorTime
	// 		result.Sectors = append(result.Sectors, sectorData)
	// 	}

	// 	return result, firstLapSkipRows, nil
	// }

	if lap == 1 {
		speed, err := parseSpeed(rows[index+2])
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse speed on start grip. value: %s, index: %d, error: %w", rows[arrIndex+1], arrIndex+1, err)
		}

		sectorData = SectorInfo{
			Speed: speed,
			Num:   FirstSector,
		}

		result.Sectors = append(result.Sectors, sectorData)
		arrIndex += 3
		for i := 0; i < 2; i++ {
			arrIndex += i * 2
			sectorData = SectorInfo{}
			sectorTime, err := parseTime(rows[arrIndex])
			if err != nil {
				return nil, 0, fmt.Errorf("failed to parse sector time in start lap. value: %s, index: %d, error: %w", rows[arrIndex], arrIndex, err)
			}

			speed, err := parseSpeed(rows[arrIndex+1])
			if err != nil {
				return nil, 0, fmt.Errorf("failed to parse speed on first lap. value: %s, index: %d, error: %w", rows[arrIndex+1], arrIndex+1, err)
			}

			sectorData.Speed = speed
			sectorData.Time = sectorTime
			result.Sectors = append(result.Sectors, sectorData)
		}

		return result, firstLapSkipRows, nil
	}

	arrIndex += 2
	for i := 0; i < 3; i++ {
		arrIndex += i * 2
		// todo: refactor this
		sectorData = SectorInfo{
			Num: i - 1,
		}
		sectorTime, err := parseTime(rows[arrIndex])
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse sector time. value: %s, index: %d, error: %w", rows[arrIndex], arrIndex, err)
		}

		speed, err := parseSpeed(rows[arrIndex+1])
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse speed. value: %s, index: %d, error: %w", rows[arrIndex+1], arrIndex+1, err)
		}

		sectorData.Speed = speed
		sectorData.Time = sectorTime
		result.Sectors = append(result.Sectors, sectorData)
	}

	fullTime, err := parseTime(rows[arrIndex+1])
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse whole lap time: %w", err)
	}

	result.FullTime = fullTime
	return result, lapSkipRows, nil
}

func isLapNumber(rows []string, index int) bool {
	_, err := strconv.ParseInt(rows[index], 10, 32)
	if err != nil {
		return false
	}

	if index == 0 {
		return false
	}

	isTime, _ := regexp.MatchString(`^\d+\d{2}\d{3}$`, rows[index-1])
	return isTime || rows[index-1] == "TIME"
}

func isRacerNumber(value string) bool {
	arrName := strings.Split(value, " ")
	return len(arrName) == 2 && arrName[0] != "SECTOR"
}

func parseTime(value string) (time.Time, error) {
	var result time.Time
	result, err := time.Parse("4:05.999", value)
	if err != nil {
		result, err = time.Parse("05.999", value)
		if err != nil {
			return time.Time{}, err
		}

	}

	return result, nil
}

func parseSpeed(value string) (float64, error) {
	speed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse speed. value: %s, error: %w", value, err)
	}

	return speed, nil
}
