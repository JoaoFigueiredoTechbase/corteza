package freepbx

type KV[T any] struct {
	Type  string `json:"@type"`
	Value T      `json:"@value"`
}

type CallValue struct {
	BillSec  string `json:"billsec"`
	Dst      string `json:"dst"`
	Sequence string `json:"sequence"`
}

type PriceValue struct {
	CallRating  string `json:"call_rating"`
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
	Price       string `json:"price"`
	PriceRecord string `json:"price_record"`
	Type        string `json:"type"`
}

type HandleCalculatePriceBody struct {
	Calls  []KV[CallValue]  `json:"Calls"`
	Prices []KV[PriceValue] `json:"Prices"`
}

type CalculatePriceResponse struct {
	Sequence    string  `json:"sequence"`
	CallType    string  `json:"call_type"`
	CallPrice   float64 `json:"call_price"`
	PriceRecord string  `json:"price_record"`
}
