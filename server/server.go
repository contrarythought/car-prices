package server

import (
	"net/http"
	"strings"
)

type HandlerMap struct {
	pathToHandler map[string]http.HandlerFunc
}

func NewHandlerMap() *HandlerMap {
	return &HandlerMap{
		pathToHandler: make(map[string]http.HandlerFunc),
	}
}

func (hm *HandlerMap) AddPath(method, resPath string, handlerFunc http.HandlerFunc) {
	hm.pathToHandler[strings.Join([]string{method, resPath}, " ")] = handlerFunc
}

// TODO
func Login(w http.ResponseWriter, req *http.Request) {

}

type DataSubmission struct {
	Brand   string  `json:"brand"`
	Model   string  `json:"model"`
	State   string  `json:"state"`
	City    string  `json:"city,omitempty"`
	Seller  string  `json:"dealership,omitempty"`
	Price   float64 `json:"price"`
	Receipt []byte  `json:"receipt"`
	Review  string  `json:"review,omitempty"`
}

func NewDataSubmission(brand, model, state, city, seller, review string, price float64, receipt []byte) *DataSubmission {
	return &DataSubmission{
		Brand:   brand,
		Model:   model,
		State:   state,
		City:    city,
		Seller:  seller,
		Price:   price,
		Receipt: receipt,
		Review:  review,
	}
}

