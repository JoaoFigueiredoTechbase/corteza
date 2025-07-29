package Yeastar

type ExtensionRegistrationEvent struct {
	TypeName     string  `json:"type_name"`
	EventType    *string `json:"event_type,omitempty"`
	Extension    *string `json:"extension,omitempty"`
	Kind         *string `json:"kind,omitempty"`
	Status       *string `json:"status,omitempty"`
	RegisteredIP *string `json:"registered_ip,omitempty"`
}
