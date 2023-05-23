package main

import (
	"car_prices/scrapers"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

type Config struct {
	ScrapeZip     bool `json:"scrapeZip"`
	ScrapeDealers bool `json:"scrapeDealers"`
}

func readConfig() (*Config, error) {
	file, err := os.Open(`config.json`)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	jsonData, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config Config
	if err = json.Unmarshal(jsonData, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func main() {
	config, err := readConfig()
	if err != nil {
		log.Fatal(err)
	}

	if config.ScrapeZip {
		zipCodes, err := scrapers.ScrapeZipCodes()
		if err != nil {
			log.Fatal(err)
		}

		file, err := os.Create(`scrapers\zip_codes.json`)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		jsonData, err := json.MarshalIndent(zipCodes.StateToZip, "", "   ")
		if err != nil {
			log.Fatal(err)
		}

		if _, err = fmt.Fprintln(file, string(jsonData)); err != nil {
			log.Fatal(err)
		}
	}

	// TODO
	if config.ScrapeDealers {

	}
}
