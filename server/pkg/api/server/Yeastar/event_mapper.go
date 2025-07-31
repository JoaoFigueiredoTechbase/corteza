package Yeastar

import "strings"

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

// func mapToEventCallStatusChanged(events map[string]interface{}, msg map[string]interface{}, members []CallMember) *CallStatusChangedEvent {
// 	return &CallStatusChangedEvent{
// 		TypeName:  "CallStatusChanged",
// 		EventType: getStringPointer(events, "type"),
// 		CallID:    getStringPointer(msg, "call_id"),
// 		Members:   members,
// 	}
// }

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
	// log.Printf("Events map: %+v", events)
	// log.Printf("Looking for 'type' in events: %v", events["type"])

	// eventNumber := getIntPointer(events, "type")
	// log.Printf("EventNumber result: %v", eventNumber)

	return &CallEvent{
		TypeName:    typeName,
		EventType:   getStringPointer(events, "type"),
		EventNumber: getIntPointer(events, "type"),
		SN:          getStringPointer(events, "sn"),
		CallID:      getStringPointer(msg, "call_id"),
		Reason:      getStringPointer(msg, "reason"),
		Members:     processMembers(msg),
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

func mapToUaCSTACall(events map[string]interface{}, msg map[string]interface{}) *UaCSTACallEvent {
	return &UaCSTACallEvent{
		TypeName:  "UaCSTACall",
		EventType: getStringPointer(events, "type"),
		SN:        getStringPointer(events, "sn"),
		Operation: getStringPointer(msg, "operation"),
		Extension: getStringPointer(msg, "extension"),
		CallID:    getStringPointer(msg, "call_id"),
		IPAddress: getStringPointer(msg, "ip_address"),
	}
}

func mapToExtensionConfiguration(events map[string]interface{}, msg map[string]interface{}) *ExtensionConfigurationEvent {
	return &ExtensionConfigurationEvent{
		TypeName:  "ExtensionConfiguration",
		EventType: getStringPointer(events, "type"),
		Extension: getStringPointer(msg, "ext_number"),
		Option:    getStringPointer(msg, "option"),
	}
}

func mapToAgentAutoPause(events map[string]interface{}, msg map[string]interface{}) *AgentAutoPauseEvent {
	return &AgentAutoPauseEvent{
		TypeName:    "AgentAutoPause",
		EventType:   getStringPointer(events, "type"),
		QueueNumber: getStringPointer(msg, "queue_number"),
		AgentNumber: getStringPointer(msg, "agent_number"),
		Calls:       processCalls(msg),
	}
}

func mapToAgentRingingTimeout(events map[string]interface{}, msg map[string]interface{}) *AgentRingingTimeoutEvent {
	return &AgentRingingTimeoutEvent{
		TypeName:     "AgentRingingTimeout",
		EventType:    getStringPointer(events, "type"),
		QueueNumber:  getStringPointer(msg, "queue_number"),
		AgentNumber:  getStringPointer(msg, "agent_number"),
		CallerNumber: getStringPointer(msg, "caller_number"),
		CallID:       getStringPointer(msg, "call_id"),
	}
}

func mapToCallNoteStatus(events map[string]interface{}, msg map[string]interface{}) *CallNoteStatusEvent {
	return &CallNoteStatusEvent{
		TypeName:   "CallNoteStatus",
		EventType:  getStringPointer(events, "type"),
		Display:    getIntPointer(msg, "display"),
		Trigger:    getIntPointer(msg, "trigger_call_note_popup"),
		SipCallID:  getStringPointer(msg, "sip_call_id"),
		CallNoteID: getStringPointer(msg, "call_note_id"),
		GroupID:    getStringPointer(msg, "group_id"),
		ExtNum:     getStringPointer(msg, "ext_num"),
		Channel:    getStringPointer(msg, "channel"),
	}
}

func mapToAgentStatusChanged(events map[string]interface{}, msg map[string]interface{}) *AgentStatusChangedEvent {
	return &AgentStatusChangedEvent{
		TypeName:    "AgentStatusChanged",
		EventType:   getStringPointer(events, "type"),
		SN:          getStringPointer(events, "sn"),
		QueueNumber: stripQueuePrefix(getStringPointer(msg, "queue_number")),
		AgentNumber: getStringPointer(msg, "agent_number"),
		Status:      getStringPointer(msg, "status"),
		Reason:      getStringPointer(msg, "reason"),
	}
}

func stripQueuePrefix(val *string) *string {
	if val == nil {
		return nil
	}
	str := *val
	if strings.HasPrefix(str, "queue-") {
		str = strings.TrimPrefix(str, "queue-")
	}
	return &str
}
