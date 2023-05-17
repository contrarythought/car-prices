package utils

import (
	"os"
	"testing"
)

func TestScrapeActiveBrands(t *testing.T) {
	ScrapeBrands()
	if err := os.Remove(`active_brands.json`); err != nil {
		t.Error(err)
	}
}

func TestScrapeBrand2(t *testing.T) {
	ScrapeBrands2()
	if err := os.Remove(`active_brands.json`); err != nil {
		t.Error(err)
	}
}