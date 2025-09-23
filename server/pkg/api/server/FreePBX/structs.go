package freepbx

type KV[T any] struct {
	Type  string `json:"@type"`
	Value T      `json:"@value"`
}

type CallValue struct {
	BillSec           string `json:"billsec"`
	CdrId             string `json:"cdr_id"`
	Dst               string `json:"dst"`
	Sequence          string `json:"sequence"`
	UniqueId          string `json:"unique_id"`
	TrunkClientRecord string `json:"trunk_cliente_record"`
}

type ClientValue struct {
	ClientRecord  string   `json:"client_record"`
	PlanCountries []string `json:"plan_countries"`
	ServiceTime   string   `json:"service_time"`
	RecordID      string   `json:"recordID"`
}

type PriceValue struct {
	CallRating  string `json:"call_rating"`
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
	Price       string `json:"price"`
	Type        string `json:"type"`
}

type HandleCalculatePriceBody struct {
	Calls   []string `json:"Calls"`
	Clients []string `json:"Clients"`
	Prices  []string `json:"Prices"`
}

type CalculatePriceResponse struct {
	Sequence    string  `json:"sequence"`
	CallType    string  `json:"call_type"`
	CallPrice   float64 `json:"call_price"`
	PriceRecord string  `json:"price_record"`
}

type ClientSummary struct {
	ClientRecord      string  `json:"client_record"`
	RecordID          string  `json:"record_id"`
	TotalCost         float64 `json:"total_cost"`
	NationalCost      float64 `json:"national_cost"`
	InternationalCost float64 `json:"international_cost"`
	TotalTime         int     `json:"total_time"`
	NationalTime      int     `json:"national_time"`
	InternationalTime int     `json:"international_time"`
}

type CallDetail struct {
	Sequence    string  `json:"sequence"`
	CdrId       string  `json:"cdr_id"`
	UniqueId    string  `json:"unique_id"`
	CallPrice   float64 `json:"call_price"`
	CountryName string  `json:"country_name"`
	CountryCode string  `json:"country_code"`
	CallType    string  `json:"call_type"`
}

type CalculatePriceFullResponse struct {
	Clients []ClientSummary `json:"clients"`
	Calls   []CallDetail    `json:"calls"`
}
