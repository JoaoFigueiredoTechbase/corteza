package Yeastar

type ExtensionRegistrationEvent struct {
	TypeName     string  `json:"type_name"`
	EventType    *string `json:"event_type,omitempty"`
	Extension    *string `json:"extension,omitempty"`
	Kind         *string `json:"kind,omitempty"`
	Status       *string `json:"status,omitempty"`
	RegisteredIP *string `json:"registered_ip,omitempty"`
}

type ExtensionCallStatusEvent struct {
	TypeName  string  `json:"type_name"`
	EventType *string `json:"event_type,omitempty"`
	SN        *string `json:"sn,omitempty"`
	Extension *string `json:"extension,omitempty"`
	Status    *string `json:"status,omitempty"`
}

type ExtensionPresenceStatusEvent struct {
	TypeName  string  `json:"type_name"`
	EventType *string `json:"event_type,omitempty"`
	Extension *string `json:"extension,omitempty"`
	Status    *string `json:"status,omitempty"`
}

type CallStatusChangedEvent struct {
	TypeName  string       `json:"type_name"`
	EventType *string      `json:"event_type,omitempty"`
	CallID    *string      `json:"call_id,omitempty"`
	Reason    *string      `json:"reason,omitempty"`
	Members   []CallMember `json:"members,omitempty"`
}

type CallEvent struct {
	TypeName  string       `json:"type_name"`
	EventType *string      `json:"event_type,omitempty"`
	SN        *string      `json:"sn,omitempty"` // Only used by CallTransfer
	CallID    *string      `json:"call_id,omitempty"`
	Reason    *string      `json:"reason,omitempty"` // Only used by CallStatusChanged
	Members   []CallMember `json:"members,omitempty"`
}

type CallMember struct {
	Type      string  `json:"type"`
	Number    *string `json:"number,omitempty"`     // for extension
	From      *string `json:"from,omitempty"`       // for inbound/outbound
	To        *string `json:"to,omitempty"`         // for inbound/outbound
	TrunkName *string `json:"trunk_name,omitempty"` // for inbound/outbound
	ChannelID *string `json:"channel_id,omitempty"`
	Status    *string `json:"status,omitempty"`
	CallPath  *string `json:"call_path,omitempty"`
}

type NewCDREvent struct {
	TypeName      string  `json:"type"`
	EventType     *string `json:"event_type,omitempty"`
	SN            *string `json:"sn,omitempty"`
	CallID        *string `json:"call_id,omitempty"`
	TimeStart     *string `json:"time_start,omitempty"`
	CallFrom      *string `json:"call_from,omitempty"`
	CallTo        *string `json:"call_to,omitempty"`
	CallDuration  *int    `json:"call_duration,omitempty"`
	TalkDuration  *int    `json:"talk_duration,omitempty"`
	SrcTrunkName  *string `json:"src_trunk_name,omitempty"`
	DstTrunkName  *string `json:"dst_trunk_name,omitempty"`
	PinCode       *string `json:"pin_code,omitempty"`
	Status        *string `json:"status,omitempty"`
	CallType      *string `json:"call_type,omitempty"`
	Recording     *string `json:"recording,omitempty"`
	DIDNumber     *string `json:"did_number,omitempty"`
	AgentRingTime *int    `json:"agent_ring_time,omitempty"`
	UID           *string `json:"uid,omitempty"`
	CallNoteID    *string `json:"call_note_id,omitempty"`
	EnbCallNote   *int    `json:"enb_call_note,omitempty"`
}

type SatisfactionEvent struct {
	TypeName     string  `json:"type"`
	EventType    *string `json:"event_type,omitempty"`
	SN           *string `json:"sn,omitempty"`
	CallID       *string `json:"call_id,omitempty"`
	SurveyResult *string `json:"survey_result,omitempty"`
}

type UaCSTACallEvent struct {
	TypeName  string  `json:"type"`
	EventType *string `json:"event_type,omitempty"`
	SN        *string `json:"sn,omitempty"`
	Operation *string `json:"operation,omitempty"`
	Extension *string `json:"extension,omitempty"`
	CallID    *string `json:"call_id,omitempty"`
	IPAddress *string `json:"ip_address,omitempty"`
}

type ExtensionConfigurationEvent struct {
	TypeName  string  `json:"type"`
	EventType *string `json:"event_type,omitempty"`
	Extension *string `json:"extension,omitempty"`
	Option    *string `json:"option,omitempty"`
}

type AgentAutoPauseEvent struct {
	TypeName    string     `json:"type_name"`
	EventType   *string    `json:"event_type,omitempty"`
	QueueNumber *string    `json:"queue_number,omitempty"`
	AgentNumber *string    `json:"agent_number,omitempty"`
	Calls       []CallInfo `json:"calls,omitempty"`
}

type CallInfo struct {
	Type         string  `json:"type"`
	CallerNumber *string `json:"caller_number,omitempty"`
	CallID       *string `json:"call_id,omitempty"`
}

type AgentRingingTimeoutEvent struct {
	TypeName     string  `json:"type_name"`
	EventType    *string `json:"event_type,omitempty"`
	QueueNumber  *string `json:"queue_number,omitempty"`
	AgentNumber  *string `json:"agent_number,omitempty"`
	CallerNumber *string `json:"caller_number,omitempty"`
	CallID       *string `json:"call_id,omitempty"`
}

type CallNoteStatusEvent struct {
	TypeName   string  `json:"type_name"`
	EventType  *string `json:"event_type,omitempty"`
	Display    *string `json:"display,omitempty"`
	Trigger    *string `json:"trigger_call_note_pop_up,omitempty"`
	SipCallID  *string `json:"sip_call_id,omitempty"`
	CallNoteID *string `json:"call_note_id,omitempty"`
	GroupID    *string `json:"group_id,omitempty"`
	ExtNum     *string `json:"ext_num,omitempty"`
	Channel    *string `json:"channel,omitempty"`
}

type AgentStatusChangedEvent struct {
	TypeName    string  `json:"type_name"`
	EventType   *string `json:"event_type,omitempty"`
	SN          *string `json:"sn,omitempty"`
	QueueNumber *string `json:"queue_number,omitempty"`
	AgentNumber *string `json:"agent_number,omitempty"`
	Status      *string `json:"status,omitempty"`
	Reason      *string `json:"reason,omitempty"`
}
