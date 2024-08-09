package main

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/dslipak/pdf"
)

type LapInfo struct {
	Pit      bool
	Sectors  []SectorInfo
	FullTime time.Time
	NumOfLap int
}

type SectorInfo struct {
	Time  time.Time
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
	for i := range pdfReader.NumPage() {
		page := pdfReader.Page(i)
		sourceData, err := getSourceDataFromPage(page)
		if err != nil {
			return fmt.Errorf("failed to get data from page: %d. error: %w", i, err)
		}

		err := raceData.parseRacerData(sourceData)
		if err != nil {
			return fmt.Errorf("failed to parse racer data: %w", err)
		}
	}
	return nil
}

func (r *RaceData) parseRacerData(sourceData []string) error {
	racerName := "unknown"


	// for index, value := range sourceData {
	for i := 0; i < len(sourceData); i++ {

		if isLapNumber(sourceData, index) {

			lapData, rowsToSkip, err := setLapData(rows, i)
			if err != nil {
				return fmt.Errorf("failed to set lap data: %w", err)
			}

			r[racerName] = append(r[racerName], *lapData)
			key += rowsToSkip
			continue
		}

	}
}

func isLapNumber(rows []string, index int) bool {
	_, err := strconv.ParseInt(rows[index], 10, 32)
	if err != nil {
		return false
	}

	if index == 0 {
		return false
	}

	isTime, _ := regexp.MatchString(`^(?:\d{1,3}:)?\d{2}[.]\d{3}$`, rows[index-1])
	return isTime || rows[index-1] == "TIME"
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