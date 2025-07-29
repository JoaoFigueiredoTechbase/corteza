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
