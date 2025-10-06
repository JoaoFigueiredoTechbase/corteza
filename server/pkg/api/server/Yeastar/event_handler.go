package Yeastar

import (
	"fmt"
	"log"
	"net"
	"strings"
)

const (
	EventExtensionRegistrationPath   = "/event/30007"
	EventExtensionCallStatusPath     = "/event/30008"
	EventExtensionPresenceStatusPath = "/event/30009"
	EventCallStatusChangedPath       = "/event/30011"
	EventNewCDRPath                  = "/event/30012"
	EventCallTransferPath            = "/event/30013"
	EventCallFowardPath              = "/event/30014"
	EventCallStatusPath              = "/event/30015"
	EventSatisfactionPath            = "/event/30019"
	EventUaCSTACallPath              = "/event/30020"
	EventExtensionConfigurationPath  = "/event/30022"
	EventAgentPausePath              = "/event/30025"
	EventAgentRingTimeoutPath        = "/event/30026"
	EventReportDownloadPath          = "/event/30027"
	EventCallNoteStatusChangedPath   = "/event/30028"
	EventAgentStatusChangedPath      = "/event/30029"
)

func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// Global base URL (can be overridden by environment variable)
var baseURL string

// Initialize base URL
func init() {
	baseURL, _ = getLocalIP()
	//baseURL = "https://webhook.site/f138fe58-2d58-4255-a3c3-9f92649e1339" // default

	// baseURL = os.Getenv("API_BASE_URL")
	// if baseURL == "" {

	// }
}

// Base URL function
func getBaseURL() string {
	return baseURL
}

func getSyncURL() string {
	syncURL := "http://" + getBaseURL()
	return syncURL
}

// Build endpoint URL
func buildURL(path string) string {
	base := strings.TrimRight(getBaseURL(), "/")
	path = strings.TrimLeft(path, "/")
	return fmt.Sprintf("http://%s/api/gateway/%s", base, path)
}

// 30007
func handleEventExtensionRegistration(event map[string]interface{}) (*ExtensionRegistrationEvent, error) {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToExtensionRegistration(event, msg)

	log.Printf("Successfully mapped ExtensionRegistration: %+v", eventData)

	if err := sendEventToEndpointWithRetry(eventData, buildURL(EventExtensionRegistrationPath), 3); err != nil {
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

	if err := sendEventToEndpointWithRetry(eventData, buildURL(EventExtensionCallStatusPath), 3); err != nil {
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

	if err := sendEventToEndpointWithRetry(eventData, buildURL(EventExtensionPresenceStatusPath), 3); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30011
func handleEventCallStatusChanged(event map[string]interface{}) (*CallEvent, error) {
	Log("info", "Got here - 30011")

	msg, err := verifyMessageWithCleaning(event)
	if err != nil {
		return nil, err
	}

	Log("info", "Mapper 30011")
	eventData := mapToCallEvent(event, msg, "CallStatusChanged")
	log.Printf("Successfully mapped CallStatusChanged: %+v", eventData)

	Log("info", "Sending 30011")
	if err := sendEventToEndpointWithRetry(eventData, buildURL(EventCallStatusChangedPath), 3); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	Log("info", "End")
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

	if err := sendEventToEndpointWithRetry(eventData, buildURL(EventNewCDRPath), 3); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	if eventData.UID != nil {
		err := SearchNewCDR(getSyncURL(), *eventData.UID)
		if err != nil {
			log.Printf("SearchNewCDR failed: %v", err)
		}
	} else {
		log.Println("UID is missing in eventData")
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

	if err := sendEventToEndpointWithRetry(eventData, buildURL(EventCallTransferPath), 3); err != nil {
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

	if err := sendEventToEndpointWithRetry(eventData, buildURL(EventCallFowardPath), 3); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30015
func handleEventCallFailedStatus(event map[string]interface{}) (*CallEvent, error) {
	msg, err := verifyMessageWithCleaning(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToCallEvent(event, msg, "CallFailed")
	log.Printf("Successfully mapped CallStatus: %+v", eventData)

	if err := sendEventToEndpointWithRetry(eventData, buildURL(EventCallStatusPath), 3); err != nil {
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

	if err := sendEventToEndpointWithRetry(eventData, buildURL(EventSatisfactionPath), 3); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30020
// func handleEventUaCSTACall(event map[string]interface{}) (*UaCSTACallEvent, error) {
// 	msg, err := verifyMessage(event)
// 	if err != nil {
// 		return nil, err
// 	}

// 	eventData := mapToUaCSTACall(event, msg)
// 	log.Printf("Successfully mapped UaCSTACall: %+v", eventData)

// 	if err := sendEventToEndpoint(eventData, buildURL(EventUaCSTACallPath)); err != nil {
// 		log.Printf("Failed to send event to endpoint: %v", err)
// 		return nil, err
// 	}

// 	return eventData, nil
// }

// 30022
func handleEventExtensionConfiguration(event map[string]interface{}) (*ExtensionConfigurationEvent, error) {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	var agent *Agent
	ext := getStringPointer(msg, "ext_number")
	if ext != nil {
		foundAgent, err := SearchExtensionContext(getSyncURL(), *ext)
		if err != nil {
			log.Printf("Search failed for extension %s: %v", *ext, err)
		} else {
			agent = &foundAgent
		}
	} else {
		log.Println("Extension number is missing in event message")
	}

	eventData := mapToExtensionConfiguration(event, msg, agent)
	log.Printf("Successfully mapped ExtensionConfigurationEvent: %+v", eventData)

	if err := sendEventToEndpointWithRetry(eventData, buildURL(EventExtensionConfigurationPath), 3); err != nil {
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

	if err := sendEventToEndpointWithRetry(eventData, buildURL(EventAgentPausePath), 3); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30026
func handleEventAgentRingTimeout(event map[string]interface{}) (*AgentRingingTimeoutEvent, error) {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToAgentRingingTimeout(event, msg)
	log.Printf("Successfully mapped AgentRingingTimeout: %+v", eventData)

	if err := sendEventToEndpointWithRetry(eventData, buildURL(EventAgentRingTimeoutPath), 3); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// //30027
// func handleEventReportDownload() error {

// }

// 30028
func handleEventCallNoteStatusChanged(event map[string]interface{}) (*CallNoteStatusEvent, error) {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToCallNoteStatus(event, msg)
	log.Printf("Successfully mapped CallNoteStatus: %+v", eventData)

	if err := sendEventToEndpointWithRetry(eventData, buildURL(EventCallNoteStatusChangedPath), 3); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30029
func handleEventAgentStatusChanged(event map[string]interface{}) (*AgentStatusChangedEvent, error) {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToAgentStatusChanged(event, msg)
	log.Printf("Successfully mapped AgentStatusChanged: %+v", eventData)

	if err := sendEventToEndpointWithRetry(eventData, buildURL(EventAgentStatusChangedPath), 3); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}
