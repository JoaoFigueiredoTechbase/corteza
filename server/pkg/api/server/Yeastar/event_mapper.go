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

func mapToExtensionPresenceStatus(events map[string]interface{}, msg map[string]interface{}) *ExtensionPresenceStatusEvent {
	return &ExtensionPresenceStatusEvent{
		TypeName:  "ExtensionPresenceStatus",
		EventType: getStringPointer(events, "type"),
		Extension: getStringPointer(msg, "extension"),
		Status:    getStringPointer(msg, "status"),
	}
}

func mapToEventCallStatusChanged(events map[string]interface{}, msg map[string]interface{}, members []CallMember) *CallStatusChangedEvent {
	return &CallStatusChangedEvent{
		TypeName:  "CallStatusChanged",
		EventType: getStringPointer(events, "type"),
		CallID:    getStringPointer(msg, "call_id"),
		Members:   members,
	}
}

func mapToNewCDR(events map[string]interface{}, msg map[string]interface{}) *NewCDREvent {
	return &NewCDREvent{
		TypeName:      "NewCDR",
		EventType:     getStringPointer(events, "type"),
		SN:            getStringPointer(events, "sn"),
		CallID:        getStringPointer(msg, "call_id"),
		TimeStart:     getStringPointer(msg, "time_start"),
		CallFrom:      getStringPointer(msg, "call_from"),
		CallTo:        getStringPointer(msg, "call_to"),
		CallDuration:  getIntPointer(msg, "call_duration"),
		TalkDuration:  getIntPointer(msg, "talk_duration"),
		SrcTrunkName:  getStringPointer(msg, "src_trunk_name"),
		DstTrunkName:  getStringPointer(msg, "dst_trunk_name"),
		PinCode:       getStringPointer(msg, "pin_code"),
		Status:        getStringPointer(msg, "status"),
		CallType:      getStringPointer(msg, "type"),
		Recording:     getStringPointer(msg, "recording"),
		DIDNumber:     getStringPointer(msg, "did_number"),
		AgentRingTime: getIntPointer(msg, "agent_ring_time"),
		UID:           getStringPointer(msg, "uid"),
		CallNoteID:    getStringPointer(msg, "call_note_id"),
		EnbCallNote:   getIntPointer(msg, "enb_call_note"),
	}
}

func mapToCallEvent(events map[string]interface{}, msg map[string]interface{}, typeName string) *CallEvent {
	return &CallEvent{
		TypeName:  typeName,
		EventType: getStringPointer(events, "type"),
		SN:        getStringPointer(events, "sn"), // Will be nil for CallStatusChanged
		CallID:    getStringPointer(msg, "call_id"),
		Reason:    getStringPointer(msg, "reason"), // Will be nil for CallTransfer
		Members:   processMembers(msg),
	}
}

func mapToSatisfaction(events map[string]interface{}, msg map[string]interface{}) *SatisfactionEvent {
	return &SatisfactionEvent{
		TypeName:     "Satisfaction",
		EventType:    getStringPointer(events, "type"),
		SN:           getStringPointer(events, "sn"),
		CallID:       getStringPointer(msg, "call_id"),
		SurveyResult: getStringPointer(msg, "survey_result"),
	}
}
