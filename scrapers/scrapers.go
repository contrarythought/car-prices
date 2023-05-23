package scrapers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/xuri/excelize/v2"
)

type brand = string
type model = string

const (
	WIKI_CAR_BRANDS_URL = `https://en.wikipedia.org/wiki/List_of_car_brands`
	AUTOTRADER_URL      = `https://www.autotrader.com/`
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

type CarBrands struct {
	BrandModelMap map[brand][]model `json:"names"`
	mu            sync.RWMutex
}

func NewCarBrands() *CarBrands {
	return &CarBrands{
		BrandModelMap: make(map[brand][]model),
	}
}

func (b *CarBrands) Add(brand, model string) {
	b.mu.Lock()
	defer b.mu.Unlock()
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
				brandMap.Add(brand, model)
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

type Dealer struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
}

type DealerSet struct {
	set map[string]*Dealer
}

func NewDealerSet() *DealerSet {
	return &DealerSet{set: make(map[string]*Dealer)}
}

type DealerMap struct {
	ZipToDealers map[uint]*DealerSet
	dealerSet    DealerSet
	mu           sync.RWMutex
}

func NewDealerMap() *DealerMap {
	return &DealerMap{
		ZipToDealers: make(map[uint]*DealerSet),
		dealerSet:    *NewDealerSet(),
	}
}

func (dm *DealerMap) Add(key uint, val Dealer) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.ZipToDealers[key].set[val.Name] = &val
	dm.dealerSet.set[val.Name] = &val
}

func (dm *DealerMap) Get(dealerName string) (*Dealer, error) {
	dealer, have := dm.dealerSet.set[dealerName]
	if !have {
		return nil, fmt.Errorf("dealer not found")
	}
	return dealer, nil
}

type ZipCodes struct {
	stateToZip map[string][]int
}

func NewZipCodes() *ZipCodes {
	return &ZipCodes{
		stateToZip: make(map[string][]int),
	}
}

const (
	ZIPCODE_URL = `https://www.zipcode.com.ng/2022/06/list-of-5-digit-zip-codes-united-states.html`
)

func ScrapeZipCodes() (*ZipCodes, error) {
	zipCodes := NewZipCodes()

	c := colly.NewCollector(colly.UserAgent(getUserAgent()))

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println(r.StatusCode, ":", err)
	})

	c.OnHTML(`tr`, func(h *colly.HTMLElement) {
		// get the state
		state := h.ChildText(`a[href^="/2022/06"]`)

		// get the zip code range
		zipRangeStr := h.ChildText(`#content > div:nth-child(5) > table > tbody > tr > td:nth-child(3)`)
		zipRange := strings.Split(zipRangeStr, " to ")

		if len(state) > 1 && len(zipRange) > 1 {
			startZip, err := strconv.Atoi(zipRange[0])
			if err != nil {
				fmt.Println(err)
			}

			endZip, err := strconv.Atoi(zipRange[1])
			if err != nil {
				fmt.Println(err)
			}

			zipRangeArr := make([]int, endZip-startZip+1)

			for i, j := startZip, 0; i <= endZip && j < endZip-startZip+1; i, j = i+1, j+1 {
				zipRangeArr[j] = i
			}

			// add state/zip into map
			zipCodes.stateToZip[state] = zipRangeArr
		}
	})

	if err := c.Visit(ZIPCODE_URL); err != nil {
		return nil, err
	}

	return zipCodes, nil
}

// TODO
func ScrapeDealers() error {

	return nil
}
