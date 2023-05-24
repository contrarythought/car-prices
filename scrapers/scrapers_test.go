package scrapers

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gocolly/colly/v2"
)

func TestScrapeSpread(t *testing.T) {
	b := NewCarBrands()
	if err := ScrapeBrandsFromSpreadsheet(b); err != nil {
		t.Error(err)
	}
}

func TestScrapeZip(t *testing.T) {
	file, err := os.Create("zipcodes.txt")
	if err != nil {
		t.Error(err)
	}
	defer file.Close()

	zipcodes, err := ScrapeZipCodes()
	if err != nil {
		t.Error(err)
	}

	fmt.Fprintln(file, zipcodes)
}

func TestGetAutoTrader(t *testing.T) {
	file, err := os.Create("vehicle_detail_links.txt")
	if err != nil {
		t.Error(err)
	}
	defer file.Close()

	var links []string
	zip := 90210
	c := colly.NewCollector(colly.UserAgent(getUserAgent()))

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("error:", r.StatusCode, ":", err)
	})

	c.OnRequest(func(r *colly.Request) {
		setZipCodeHeaders(r, zip)
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("response:", r.StatusCode)
	})

	c.OnHTML(`a[href]`, func(h *colly.HTMLElement) {
		link := h.Attr("href")
		if strings.Index(link, `/cars-for-sale/vehicledetails`) > -1 {
			links = append(links, link)
		}
	})

	if err := c.Visit(AUTOTRADER_ZIP_URL + strconv.Itoa(zip)); err != nil {
		t.Error(err)
	}

	if len(links) == 0 {
		t.Error("err: failed to gather links")
	}

	for _, link := range links {
		fmt.Println("writing:", link)
		fmt.Fprintln(file, link)
	}
}
