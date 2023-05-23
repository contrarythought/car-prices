package scrapers

import (
	"fmt"
	"os"
	"testing"
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
