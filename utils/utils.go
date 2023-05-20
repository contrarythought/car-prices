package utils

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"

	"github.com/xuri/excelize/v2"
)

type brand = string
type model = string

type CarBrands struct {
	BrandModelMap map[brand][]model `json:"names"`
	mu            sync.RWMutex
}

func NewCarBrands() *CarBrands {
	return &CarBrands{
		BrandModelMap: make(map[brand][]model),
	}
}

const (
	URL = `https://en.wikipedia.org/wiki/List_of_car_brands`
)

func getUserAgent() string {
	userAgents := []string{
		`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36`,
		`Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36`,
		`Mozilla/5.0 (Macintosh; Intel Mac OS X 13_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Safari/605.1.15`,
		`Mozilla/5.0 (X11; CrOS x86_64 8172.45.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.64 Safari/537.36`,
		`Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.111 Safari/537.36`,
	}
	idx := rand.Intn(len(userAgents))
	return userAgents[idx]
}

func haveBrandFile() (bool, error) {
	entries, err := os.ReadDir(`C:\Users\athor\go\car_prices\utils`)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.Name() == `active_brands.json` {
			return true, nil
		}
	}
	return false, nil
}

func (b *CarBrands) Add(brand, model string) {
	b.BrandModelMap[brand] = append(b.BrandModelMap[brand], model)
}

// writes each model to its corresponding brand
func ScrapeBrandsFromSpreadsheet(brandMap *CarBrands) error {
	haveFile, err := haveBrandFile()
	if err != nil {
		return err
	}
	if haveFile {
		return fmt.Errorf("err: already have brand-model.json")
	}

	file, err := excelize.OpenFile(`Car Models List of Car Models.xlsx`)
	if err != nil {
		return err
	}
	defer file.Close()

	carListWksht := `Complete List of Car Brands`
	var brands []brand
	brand, err := file.GetCellValue(carListWksht, `A3`)
	if err != nil {
		return err
	}

	// read in each brand
	cellCnt := 4
	for brand != "" {
		brands = append(brands, brand)
		brand, err = file.GetCellValue(carListWksht, "A"+strconv.Itoa(cellCnt))
		if err != nil {
			return err
		}
		cellCnt++
	}

	var wg sync.WaitGroup

	// write to brand-model map
	for _, b := range brands {
		// have a worker for each worksheet
		wg.Add(1)
		go func(brand string) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("err recovered:", r)
				}
			}()

			cnt := 2
			model, err := file.GetCellValue(brand, "A"+strconv.Itoa(cnt))
			if err != nil {
				panic(err)
			}
			// find first cell that has value
			cnt++
			for model == "" {
				model, err = file.GetCellValue(brand, "A"+strconv.Itoa(cnt))
				if err != nil {
					panic(err)
				}
				cnt++
			}
			// write all values to map
			for model != "" {
				brandMap.mu.Lock()
				brandMap.Add(brand, model)
				brandMap.mu.Unlock()

				model, err = file.GetCellValue(brand, "A"+strconv.Itoa(cnt))
				if err != nil {
					panic(err)
				}
				cnt++
			}
		}(b)
	}
	wg.Wait()

	jsonFile, err := os.Create(`brand-model.json`)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	jsonData, err := json.MarshalIndent(brandMap.BrandModelMap, "", "   ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(jsonFile, string(jsonData))
	if err != nil {
		return err
	}

	return nil
}
