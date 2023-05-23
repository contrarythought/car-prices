package scrapers

import (
	"testing"
)

func TestScrapeSpread(t *testing.T) {
	b := NewCarBrands()
	if err := ScrapeBrandsFromSpreadsheet(b); err != nil {
		t.Error(err)
	}
}

func TestScrapeZip(t *testing.T) {
	if err := ScrapeZipCodes(); err != nil {
		t.Error(err)
	}
}
