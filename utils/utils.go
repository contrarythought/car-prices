package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/xuri/excelize/v2"
	"golang.org/x/net/html"
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

func ScrapeBrands() error {
	var brand string
	haveFile, err := haveBrandFile()
	if err != nil {
		return err
	}
	if haveFile {
		return nil
	}

	file, err := os.Create(`active_brands.json`)
	if err != nil {
		return err
	}
	defer file.Close()

	c := colly.NewCollector(
		colly.UserAgent(getUserAgent()),
	)

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("err:", err, "status:", r.StatusCode)
	})

	c.OnHTML(`h3 + ul`, func(e *colly.HTMLElement) {
		brand = e.ChildText(`li > a[href^="/wiki/"]`)
		fmt.Println(brand)
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("requesting:", r.URL)
	})

	if err = c.Visit(URL); err != nil {
		return err
	}

	return nil
}

func ScrapeBrands2() error {
	haveFile, err := haveBrandFile()
	if err != nil {
		return err
	}

	if haveFile {
		return nil
	}

	file, err := os.Create(`active_brands.json`)
	if err != nil {
		return err
	}
	defer file.Close()

	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return err
	}
	req.Header.Add("user_agent", getUserAgent())

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return err
	}

	var brands []string
	var scraper func(n *html.Node, add bool)

	scraper = func(n *html.Node, add bool) {
		if add {
			if n.Type == html.ElementNode && n.Data == `a` {
				for i := 0; i < len(n.Attr)-1; i++ {
					if n.Attr[i].Key == `href` && strings.Contains(n.Attr[i].Val, `/wiki/`) {
						if n.Attr[i+1].Key == `title` && n.FirstChild != nil {
							if strings.Contains(strings.ToLower(n.Attr[i+1].Val), strings.ToLower(n.FirstChild.Data)) {
								if strings.Contains(n.FirstChild.Data, "Timeline") {
									fmt.Println("FALSE")
									time.Sleep(3 * time.Second)
									add = false
								}
								if add {
									brands = append(brands, n.FirstChild.Data)
								}
							}
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			scraper(c, add)
		}
	}

	scraper(doc, true)

	for _, brand := range brands {
		fmt.Println(brand)
	}
	return nil
}

// TODO: test
// writes each model to its corresponding brand
func (brandMap *CarBrands) ScrapeBrandsFromSpreadsheet() error {
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
		wg.Add(1)
		go func(brand string) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("err recovered:", r)
				}
			}()

			cnt := 2
			file, err := excelize.OpenFile(brand)
			if err != nil {
				panic(err)
			}
			defer file.Close()

			model, err := file.GetCellValue(brand, "A"+strconv.Itoa(cnt))
			if err != nil {
				panic(err)
			}
			// find first cell that has value
			cnt++
			for model == "" {
				brand, err = file.GetCellValue(brand, "A"+strconv.Itoa(cnt))
				if err != nil {
					panic(err)
				}
				cnt++
			}
			// write all values to map
			for model != "" {
				brandMap.mu.Lock()
				brandMap.BrandModelMap[brand] = append(brandMap.BrandModelMap[brand], model)
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

	_, err = fmt.Fprintln(jsonFile, jsonData)
	if err != nil {
		return err
	}

	return nil
}
