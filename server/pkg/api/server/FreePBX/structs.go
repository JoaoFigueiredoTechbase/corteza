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
	Calldate          string `json:"calldate"`
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

// DailySummary represents statistics for a single day
type DailySummary struct {
	Date string `json:"date"` // Format: YYYY-MM-DD

	// Time tracking
	TotalTime         int `json:"total_time"`          // Total time of all calls for this day
	UsedPlanTime      int `json:"used_plan_time"`      // Time used from plan (covered time)
	ExceededPlanTime  int `json:"exceeded_plan_time"`  // Time that exceeded the plan
	PlanTotalTime     int `json:"plan_total_time"`     // Total time of plan calls
	NonPlanTotalTime  int `json:"non_plan_total_time"` // Total time of non-plan calls
	NationalTime      int `json:"national_time"`       // Total national call time
	InternationalTime int `json:"international_time"`  // Total international call time

	// Cost tracking
	TotalCost         float64 `json:"total_cost"`         // Total cost of all calls
	NationalCost      float64 `json:"national_cost"`      // Cost of national calls
	InternationalCost float64 `json:"international_cost"` // Cost of international calls
	ExceededPlanCost  float64 `json:"exceeded_plan_cost"` // Cost of time that exceeded plan

	// Call counting
	PlanCalls          int `json:"plan_calls"`          // Number of calls within plan countries
	NonPlanCalls       int `json:"non_plan_calls"`      // Number of calls outside plan countries
	NationalCalls      int `json:"national_calls"`      // Number of national calls (PT)
	InternationalCalls int `json:"international_calls"` // Number of international calls (non-PT)

	// Plan status for this day
	RemainingTimeAtStart int  `json:"remaining_time_at_start"` // Remaining plan time at start of day
	RemainingTimeAtEnd   int  `json:"remaining_time_at_end"`   // Remaining plan time at end of day
	PlanEndedThisDay     bool `json:"plan_ended_this_day"`     // Whether the plan ended on this day
}

// ClientSummary represents the monthly totals and daily breakdown for a client
type ClientSummary struct {
	// Basic client info
	ClientRecord string `json:"client_record"`
	RecordID     string `json:"record_id"`

	// Monthly totals (same as before)
	TotalServiceTime int    `json:"total_service_time"` // Original plan time in seconds
	TotalTime        int    `json:"total_time"`         // Total time of all calls
	UsedPlanTime     int    `json:"used_plan_time"`     // Time used from plan (covered time)
	RemainingTime    int    `json:"remaining_time"`     // Time remaining in plan
	ExceededPlanTime int    `json:"exceeded_plan_time"` // Time that exceeded the plan
	PlanEndDate      string `json:"plan_end_date"`      // Date when plan ended

	// Time breakdown by plan/non-plan (monthly totals)
	PlanTotalTime    int `json:"plan_total_time"`     // Total time of plan calls
	NonPlanTotalTime int `json:"non_plan_total_time"` // Total time of non-plan calls

	// Time breakdown by location (monthly totals)
	NationalTime      int `json:"national_time"`      // Total national call time
	InternationalTime int `json:"international_time"` // Total international call time

	// Cost tracking (monthly totals)
	TotalCost         float64 `json:"total_cost"`         // Total cost of all calls
	NationalCost      float64 `json:"national_cost"`      // Cost of national calls
	InternationalCost float64 `json:"international_cost"` // Cost of international calls
	ExceededPlanCost  float64 `json:"exceeded_plan_cost"` // Cost of time that exceeded plan

	// Call counting (monthly totals)
	PlanCalls          int `json:"plan_calls"`          // Number of calls within plan countries
	NonPlanCalls       int `json:"non_plan_calls"`      // Number of calls outside plan countries
	NationalCalls      int `json:"national_calls"`      // Number of national calls (PT)
	InternationalCalls int `json:"international_calls"` // Number of international calls (non-PT)

	// NEW: Daily breakdown
	DailyStats []DailySummary `json:"daily_stats"` // Statistics broken down by day
}

type CallDetail struct {
	Sequence    string  `json:"sequence"`
	CdrId       string  `json:"cdr_id"`
	UniqueId    string  `json:"unique_id"`
	CallPrice   float64 `json:"call_price"`
	CountryName string  `json:"country_name"`
	CountryCode string  `json:"country_code"`
	CallType    string  `json:"call_type"`
	InPlan      bool    `json:"in_plan"`
	IsNational  bool    `json:"is_national"`
}

type CalculatePriceFullResponse struct {
	Clients []ClientSummary `json:"clients"`
	Calls   []CallDetail    `json:"calls"`
}
