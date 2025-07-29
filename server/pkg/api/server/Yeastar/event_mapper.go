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
