package scrapers

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"text/template"

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
		fmt.Println("request url:", r.URL)
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

	c.OnHTML(`#mountNode > div:nth-child(2) > div.colored-background.inset.bg-gray-lightest.padding-top-4 > div > div.row.margin-horizontal-0.padding-horizontal-0 > div.row.display-flex > div.col-xs-12.col-md-9 > div:nth-child(6) > div.results-text-container.text-size-300.margin-right-4`, func(h *colly.HTMLElement) {
		resultText := strings.Split(h.Text, " ")
		numResult := resultText[2]
		fmt.Println("num results:", numResult)
	})

	testTemplate := template.New("testTemplate")
	testTemplate, err = testTemplate.Parse(AUTOTRADER_ZIP_URL)
	if err != nil {
		t.Error(err)
	}

	firstRecord := `975`

	var url strings.Builder
	if err = testTemplate.Execute(&url, struct {
		Zip         string
		FirstRecord string
	}{
		Zip:         strconv.Itoa(zip),
		FirstRecord: firstRecord,
	}); err != nil {
		t.Error(err)
	}

	if err := c.Visit(url.String()); err != nil {
		t.Error(err)
	}

	if len(links) == 0 {
		t.Error("err: failed to gather links")
	}

	for _, link := range links {
		fmt.Fprintln(file, link)
	}
}
