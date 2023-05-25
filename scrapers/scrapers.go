package scrapers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

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

type Vehicle struct {
	Name  string
	Price float64
}

func NewVehicle(name string, price float64) *Vehicle {
	return &Vehicle{
		Name:  name,
		Price: price,
	}
}

type VehicleMap struct {
	set map[int][]Vehicle
}

func NewVehicleMap() *VehicleMap {
	return &VehicleMap{
		set: make(map[int][]Vehicle),
	}
}

type DealerSet struct {
	set map[string]*Dealer
}

func NewDealerSet() *DealerSet {
	return &DealerSet{set: make(map[string]*Dealer)}
}

type DealerMap struct {
	ZipToDealers map[int]*DealerSet
	dealerSet    *DealerSet
	VehicleMap   *VehicleMap
	mu           sync.RWMutex
}

func NewDealerMap() *DealerMap {
	return &DealerMap{
		ZipToDealers: make(map[int]*DealerSet),
		dealerSet:    NewDealerSet(),
		VehicleMap:   NewVehicleMap(),
	}
}

func (dm *DealerMap) Add(zip int, dealer Dealer) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.ZipToDealers[zip].set[dealer.Name] = &dealer
	dm.dealerSet.set[dealer.Name] = &dealer
}

func (dm *DealerMap) GetDealerByName(dealerName string) (*Dealer, error) {
	dealer, have := dm.dealerSet.set[dealerName]
	if !have {
		return nil, fmt.Errorf("dealer not found")
	}
	return dealer, nil
}

func (dm *DealerMap) GetDealersbyZip(zip int) (*DealerSet, error) {
	set, have := dm.ZipToDealers[zip]
	if have && set != nil {
		return set, nil
	}
	return nil, fmt.Errorf("err: failed to find dealer set")
}

type ZipCodes struct {
	StateToZip map[string][]int
}

func NewZipCodes() *ZipCodes {
	return &ZipCodes{
		StateToZip: make(map[string][]int),
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
			zipCodes.StateToZip[state] = zipRangeArr
		}
	})

	if err := c.Visit(ZIPCODE_URL); err != nil {
		return nil, err
	}

	return zipCodes, nil
}

func setZipCodeHeaders(r *colly.Request, zipCode int) {
	r.Headers.Set(`authority`, `www.autotrader.com`)
	r.Headers.Set(`method`, r.Method)
	r.Headers.Set(`path`, `/cars-for-sale/all-cars?zip=`+strconv.Itoa(zipCode))
	r.Headers.Set(`scheme`, `https`)
	r.Headers.Set(`accept`, `text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7`)
	r.Headers.Set(`accept-Encoding`, `gzip, deflate, br`)
	r.Headers.Set(`accept-Language`, `en,en-US;q=0.9,zh-TW;q=0.8,zh;q=0.7`)
	r.Headers.Set(`sec-Ch-Ua`, `"Google Chrome";v="113", "Chromium";v="113", "Not-A.Brand";v="24"`)
	r.Headers.Set(`sec-Ch-Ua-Mobile`, `?0`)
	r.Headers.Set(`sec-Ch-Ua-Platform`, `"Windows"`)
	r.Headers.Set(`sec-Fetch-Dest`, `document`)
	r.Headers.Set(`sec-Fetch-Mode`, `navigate`)
	r.Headers.Set(`sec-Fetch-Site`, `same-origin`)
	r.Headers.Set(`sec-Fetch-User`, `?1`)
	r.Headers.Set(`upgrade-Insecure-Requests`, `1`)
}

const (
	NUM_WORKERS        = 10
	AUTOTRADER_ZIP_URL = `https://www.autotrader.com/cars-for-sale/all-cars?zip={{.Zip}}&firstRecord={{.FirstRecord}}`
)

// TODO: scrape all dealers within range of a zip code
func scrapeDealers(zipCode int, dealerMap *DealerMap) error {
	// 1. find num pages to scrape
	// 2. loop through all pages with start firstRecord = 0, and increment that all the way to max page - increment (25, 100, etc)
	// 3. visit each vehicledetail link
	// 4. if dealer is unique, add name and address to dealermap
	// 5. add car name and price to vehiclemap



	return nil
}

// TODO
func ScrapeDealers(zipCodes *ZipCodes) (*DealerMap, error) {
	if zipCodes == nil {
		return nil, fmt.Errorf("err: zipCodes not allocated")
	}

	dealerMap := NewDealerMap()
	zipCodeChan := make(chan int, NUM_WORKERS)
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	for i := 0; i < NUM_WORKERS; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					fmt.Println(r)
				}
			}()

			select {
			case code := <-zipCodeChan:
				if err := scrapeDealers(code, dealerMap); err != nil {
					panic(err)
				}
			case <-ctx.Done():
				if len(zipCodeChan) == 0 {
					return
				}
			}
		}()
	}

	// 1. loop through each state and state zip code
	for _, codes := range zipCodes.StateToZip {
		for i, code := range codes {
			if (i+1)%10 == 0 {
				time.Sleep(time.Duration(rand.Intn(4) + 3))
			}

			zipCodeChan <- code

		}
	}

	close(zipCodeChan)
	cancel()
	wg.Wait()

	// 2. send request for each zip code
	// 3. scrape the page for link to vehicle
	// 4. scrape vehicle page and extract dealer info
	// 5. place dealer info into DealerMap
	// 6. generate json file
	// 7. put json file info into db
	return dealerMap, nil
}
