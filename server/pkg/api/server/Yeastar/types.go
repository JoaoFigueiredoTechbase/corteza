package Yeastar

// Config represents the API configuration
type Config struct {
	ApiBaseUrl  string `json:"ApiBaseUrl"`
	ApiUserName string `json:"ApiUserName"`
	ApiSecret   string `json:"ApiSecret"`
}

// TokenResponse represents the token response from Yeastar API
type TokenResponse struct {
	AccessToken            string  `json:"AccessToken"`
	RefreshToken           string  `json:"RefreshToken"`
	AccessTokenExpireTime  float64 `json:"AccessTokenExpireTime"`
	RefreshTokenExpireTime float64 `json:"RefreshTokenExpireTime"`
	//ErrCode                int    `json:"errcode"`
}

type Agent struct {
	ID       int    `json:"id"`
	Presence string `json:"presence_status"`
	Number   string `json:"number"`
	Name     string `json:"caller_id_name"`
	Email    string `json:"email_addr"`
}

// Queue represents a call queue
type Queue struct {
	Name         string `json:"name"`
	Number       string `json:"number"`
	RingStrategy string `json:"ring_strategy"`
	SLATime      int    `json:"sla_time"`
}

// CDR represents a Call Detail Record
type CDR struct {
	ID                  int    `json:"id"`
	Time                string `json:"time"`
	CallFrom            string `json:"call_from"`
	CallTo              string `json:"call_to"`
	Timestamp           int64  `json:"timestamp"`
	UID                 string `json:"uid"`
	SrcAddr             string `json:"src_addr"`
	Duration            int    `json:"duration"`
	RingDuration        int    `json:"ring_duration"`
	TalkDuration        int    `json:"talk_duration"`
	Disposition         string `json:"disposition"`
	CallType            string `json:"call_type"`
	Reason              string `json:"reason"`
	CallFromNumber      string `json:"call_from_number"`
	CallFromName        string `json:"call_from_name"`
	CallToNumber        string `json:"call_to_number"`
	CallToName          string `json:"call_to_name"`
	CallID              string `json:"call_id"`
	CallNote            string `json:"call_note"`
	CallNoteType        string `json:"call_note_type"`
	CallNoteDescription string `json:"call_note_description"`
	CallNoteID          string `json:"call_note_id"`
	EnbCallNote         int    `json:"enb_call_note"`
	DID                 string `json:"did"`
	DIDName             string `json:"did_name"`
}

type DispositionCode struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CallNote struct {
	ID                  string            `json:"id"`
	GroupID             string            `json:"group_id"`
	DispositionCodeList []DispositionCode `json:"disposition_code_list"`
	Remark              string            `json:"remark"`
	AgentName           string            `json:"agent_name"`
	RegistrationTime    int64             `json:"registration_time"`
	UpdateEntry         string            `json:"update_entry"`
}

type YeastarAPIError struct {
	ErrCode int    `json:"errcode"`
	Message string `json:"errmsg"`
}

type APIResponse[T any] struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	TotalNumber int    `json:"total_number"`
	Data        []T    `json:"data"`
}

type AgentResponse = APIResponse[Agent]
type QueueResponse = APIResponse[Queue]
type CDRResponse = APIResponse[CDR]

type QueueMember struct {
	QueueID   int    `json:"queue_id"`
	QueueName string `json:"queue_name"`
	AgentID   int    `json:"agent_id"`
	AgentExt  string `json:"agent_ext"`
	Type      string `json:"type"`
}

type AgentEntry struct {
	Text  string `json:"text"`
	Text2 string `json:"text2"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type QueueRaw struct {
	ID            int          `json:"id"`
	Name          string       `json:"name"`
	Number        string       `json:"number"`
	RingStrategy  string       `json:"ring_strategy"`
	DynamicAgents []AgentEntry `json:"dynamic_agent_list"`
	StaticAgents  []AgentEntry `json:"static_agent_list"`
	Managers      []AgentEntry `json:"manager_list"`
}
