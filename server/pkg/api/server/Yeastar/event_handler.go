package Yeastar

import (
	"log"
)

// 30007
func handleEventExtensionRegistration(event map[string]interface{}) (*ExtensionRegistrationEvent, error) {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToExtensionRegistration(event, msg)

	log.Printf("Successfully mapped ExtensionRegistration: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30008
func handleEventExtensionCallStatus(event map[string]interface{}) (*ExtensionCallStatusEvent, error) {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToExtensionCallStatus(event, msg)
	log.Printf("Successfully mapped EventExtensionCallStatus: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 3009
func handleEventExtensionPresenceStatus(event map[string]interface{}) (*ExtensionPresenceStatusEvent, error) {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToExtensionPresenceStatus(event, msg)
	log.Printf("Successfully mapped ExtensionPresenceStatus: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30011
func handleEventCallStatusChanged(event map[string]interface{}) (*CallEvent, error) {
	msg, err := verifyMessageWithCleaning(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToCallEvent(event, msg, "CallStatusChanged")
	log.Printf("Successfully mapped CallStatusChanged: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30012
func handleEventNewCDR(event map[string]interface{}) (*NewCDREvent, error) {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToNewCDR(event, msg)
	log.Printf("Successfully mapped NewCDR: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30013
func handleEventCallTransfer(event map[string]interface{}) (*CallEvent, error) {
	msg, err := verifyMessageWithCleaning(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToCallEvent(event, msg, "CallTransfer")
	log.Printf("Successfully mapped CallTransfer: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30014
func handleEventCallFoward(event map[string]interface{}) (*CallEvent, error) {
	msg, err := verifyMessageWithCleaning(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToCallEvent(event, msg, "CallFoward")
	log.Printf("Successfully mapped CallFoward: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30015
func handleEventCallStatus(event map[string]interface{}) (*CallEvent, error) {
	msg, err := verifyMessageWithCleaning(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToCallEvent(event, msg, "CallStatus")
	log.Printf("Successfully mapped CallStatus: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30019
func handleEventSatisfaction(event map[string]interface{}) (*SatisfactionEvent, error) {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToSatisfaction(event, msg)
	log.Printf("Successfully mapped Satisfaction: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30020
func handleEventUaCSTACall(event map[string]interface{}) (*UaCSTACallEvent, error) {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToUaCSTACall(event, msg)
	log.Printf("Successfully mapped UaCSTACall: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30022
func handleEventExtensionConfiguration(event map[string]interface{}) (*ExtensionConfigurationEvent, error) {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToExtensionConfiguration(event, msg)
	log.Printf("Successfully mapped NewCDR: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30025
func handleEventAgentPause(event map[string]interface{}) (*AgentAutoPauseEvent, error) {
	msg, err := verifyMessageWithCleaning(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToAgentAutoPause(event, msg)
	log.Printf("Successfully mapped AgentAutoPause: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30026
func handleEventAgentRingTimeout(event map[string]interface{}) error {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	return nil
}

// //30027
// func handleEventReportDownload() error {

// }

// 30028
func handleEventCallNoteStatusChanged(event map[string]interface{}) error {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	return nil
}

// 30029
func handleEventAgentStatusChanged(event map[string]interface{}) error {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	return nil
}
