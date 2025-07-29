package Yeastar

func mapToExtensionRegistration(events map[string]interface{}, msg map[string]interface{}) *ExtensionRegistrationEvent {
	return &ExtensionRegistrationEvent{
		TypeName:     "ExtensionRegistration",
		EventType:    getStringPointer(events, "type"),
		Extension:    getStringPointer(msg, "extension"),
		Kind:         getStringPointer(msg, "kind"),
		Status:       getStringPointer(msg, "status"),
		RegisteredIP: getStringPointer(msg, "registered_ip"),
	}
}

func mapToExtensionCallStatus(events map[string]interface{}, msg map[string]interface{}) *ExtensionCallStatusEvent {
	return &ExtensionCallStatusEvent{
		TypeName:  "ExtensionCallStatus",
		EventType: getStringPointer(events, "type"),
		SN:        getStringPointer(events, "sn"),
		Extension: getStringPointer(msg, "extension"),
		Status:    getStringPointer(msg, "status"),
	}
}
