package freepbx

type KV[T any] struct {
	Type  string `json:"@type"`
	Value T      `json:"@value"`
}

type CallValue struct {
	BillSec           string `json:"billsec"`
	CdrId             string `json:"cdr_id"`
	Dst               string `json:"dst"`
	Src               string `json:"src"` // Caller number - to be added to request
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

// Call classification type
type CallClassification struct {
	Type        string `json:"type"`        // "landline", "mobile", "non_geographic", "international", "short", "value_added", "other"
	Description string `json:"description"` // Human readable description
}

// Caller type classification
type CallerType struct {
	Type           string `json:"type"`           // "geographic" (landline), "nomad", "other"
	Number         string `json:"number"`         // The caller number
	Classification string `json:"classification"` // Detailed classification
}

// Statistics for geographic (landline) callers
type GeographicCallerStats struct {
	CallerNumber string `json:"caller_number"`

	// Call counts by destination type
	LandlineCalls      int `json:"landline_calls"`
	MobileCalls        int `json:"mobile_calls"`
	InternationalCalls int `json:"international_calls"`
	NonGeographicCalls int `json:"non_geographic_calls"`
	ShortCalls         int `json:"short_calls"`
	NomadCalls         int `json:"nomad_calls"`
	ValueAddedCalls    int `json:"value_added_calls"` // 760/761 numbers
	Value760Calls      int `json:"value_760_calls"`   // Specifically 760 numbers

	// Minutes by destination type
	LandlineMinutes      int `json:"landline_minutes"`
	MobileMinutes        int `json:"mobile_minutes"`
	InternationalMinutes int `json:"international_minutes"`
	NonGeographicMinutes int `json:"non_geographic_minutes"`
	ShortMinutes         int `json:"short_minutes"`
	NomadMinutes         int `json:"nomad_minutes"`
	ValueAddedMinutes    int `json:"value_added_minutes"`
	Value760Minutes      int `json:"value_760_minutes"`

	TotalCalls   int `json:"total_calls"`
	TotalMinutes int `json:"total_minutes"`
}

// Statistics for nomad callers
type NomadCallerStats struct {
	CallerNumber         string `json:"caller_number"`
	TotalCalls           int    `json:"total_calls"`
	TotalMinutes         int    `json:"total_minutes"`
	InternationalCalls   int    `json:"international_calls"`
	InternationalMinutes int    `json:"international_minutes"`
}

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

	// Portuguese call counts
	MobileCalls         int     `json:"mobile_calls"`
	LandlineCalls       int     `json:"landline_calls"`
	PremiumCalls        int     `json:"premium_calls"`
	FreeCalls           int     `json:"free_calls"`
	SharedCostCalls     int     `json:"shared_cost_calls"`
	InternetCalls       int     `json:"internet_calls"`
	AudiotextCalls      int     `json:"audiotext_calls"`
	SpecialServiceCalls int     `json:"special_service_calls"`
	MobileCost          float64 `json:"mobile_cost"`
	LandlineCost        float64 `json:"landline_cost"`
	PremiumCost         float64 `json:"premium_cost"`
	FreeCost            float64 `json:"free_cost"`
	SharedCostCost      float64 `json:"shared_cost_cost"`
	InternetCost        float64 `json:"internet_cost"`
	AudiotextCost       float64 `json:"audiotext_cost"`
	SpecialServiceCost  float64 `json:"special_service_cost"`
}

type ClientSummary struct {
	// Basic client info
	ClientRecord string `json:"client_record"`
	RecordID     string `json:"record_id"`

	// Monthly totals
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

	// Portuguese call counts
	MobileCalls         int     `json:"mobile_calls"`
	LandlineCalls       int     `json:"landline_calls"`
	PremiumCalls        int     `json:"premium_calls"`
	FreeCalls           int     `json:"free_calls"`
	SharedCostCalls     int     `json:"shared_cost_calls"`
	InternetCalls       int     `json:"internet_calls"`
	AudiotextCalls      int     `json:"audiotext_calls"`
	SpecialServiceCalls int     `json:"special_service_calls"`
	MobileCost          float64 `json:"mobile_cost"`
	LandlineCost        float64 `json:"landline_cost"`
	PremiumCost         float64 `json:"premium_cost"`
	FreeCost            float64 `json:"free_cost"`
	SharedCostCost      float64 `json:"shared_cost_cost"`
	InternetCost        float64 `json:"internet_cost"`
	AudiotextCost       float64 `json:"audiotext_cost"`
	SpecialServiceCost  float64 `json:"special_service_cost"`

	// Daily breakdown
	DailyStats []DailySummary `json:"daily_stats"` // Statistics broken down by day

	// Caller statistics
	GeographicCallers []*GeographicCallerStats `json:"geographic_callers,omitempty"` // Stats by geographic caller
	NomadCallers      []*NomadCallerStats      `json:"nomad_callers,omitempty"`      // Stats by nomad caller
}

type CallDetail struct {
	Sequence           string              `json:"sequence"`
	CdrId              string              `json:"cdr_id"`
	UniqueId           string              `json:"unique_id"`
	CallPrice          float64             `json:"call_price"`
	CountryName        string              `json:"country_name"`
	CountryCode        string              `json:"country_code"`
	CallType           string              `json:"call_type"`
	InPlan             bool                `json:"in_plan"`
	IsNational         bool                `json:"is_national"`
	PortugueseCallType *PortugueseCallType `json:"portuguese_call_type"`
	CallerType         *CallerType         `json:"caller_type"`      // Classification of caller
	DestinationType    *CallClassification `json:"destination_type"` // Classification of destination
}

type CalculatePriceFullResponse struct {
	Clients []ClientSummary `json:"clients"`
	Calls   []CallDetail    `json:"calls"`
}

type PortugueseCallType struct {
	Type        string `json:"type"`        // "mobile", "landline", "premium", "free", "internet", "audiotext", "shared_cost", "unknown"
	Description string `json:"description"` // Human readable description
	Prefix      string `json:"prefix"`      // The identifying prefix (3, 6, 7, 8, 9, etc.)
	Category    string `json:"category"`    // "standard", "special_service", "premium_rate"
}
