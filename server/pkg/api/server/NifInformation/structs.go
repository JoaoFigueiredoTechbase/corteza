package nifinformation

import (
	"encoding/json"
	"time"
)

type ClientInformation struct {
	ClientName string `json:"client_name"`
	ClientNif  string `json:"client_nif"`
	ApiKey     string `json:"api_key"`
}

type ApiResponse struct {
	Result  string                     `json:"result"`
	Records map[string]json.RawMessage `json:"records"`
}

type SearchResponse struct {
	Result  string                     `json:"result"`
	Records map[string]json.RawMessage `json:"records"`
}

type LightweightRecord struct {
	Nif    int    `json:"nif"`
	Title  string `json:"title"`
	City   string `json:"city"`
	Pc4    string `json:"pc4"`
	Pc3    string `json:"pc3"`
	Racius string `json:"racius"`
	Url    string `json:"seo_url"`
}

type NifApiResponse struct {
	Nif        int      `json:"nif"`
	Title      string   `json:"title"`
	Address    string   `json:"address"`
	Pc4        string   `json:"pc4"`
	Pc3        string   `json:"pc3"`
	City       string   `json:"city"`
	Activity   string   `json:"activity"`
	CaeList    []string `json:"cae"`
	Email      string   `json:"email"`
	Phone      string   `json:"phone"`
	Website    string   `json:"website"`
	Fax        string   `json:"fax"`
	Region     string   `json:"region"`
	County     string   `json:"county"`
	Parish     string   `json:"parish"`
	RaciusLink string   `json:"racius"`
}

type RateLimits struct {
	Month  int
	Day    int
	Hour   int
	Minute int
}

type Usage struct {
	Month      int
	Day        int
	Hour       int
	Minute     int
	LastUpdate time.Time
}
