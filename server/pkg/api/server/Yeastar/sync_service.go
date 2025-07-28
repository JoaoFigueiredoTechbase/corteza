package Yeastar

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Initialize global managers
var (
	initOnce sync.Once
)

// InitializeGlobalManagers initializes the global managers
func InitializeGlobalManagers() {
	initOnce.Do(func() {
		GlobalConfigManager = NewConfigManager()
		GlobalTokenManager = NewTokenManager()
	})
}

func HandleSyncAllHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { // Typically, sync triggers are GET or POST depending on idempotency
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Call your existing HandleSyncAll logic
	err := SyncAll() // This is your core sync logic

	if err != nil {
		log.Printf("Error during full sync: %v", err) // Log the detailed error
		http.Error(w, fmt.Sprintf("Synchronization failed: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Synchronization process initiated successfully. Check logs for details."))
	log.Println("Full synchronization process completed successfully.")
}

func SyncAll() error {
	InitializeGlobalManagers()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cortezaClient := NewCortezaClient("http://localhost:80")
	service := NewYeastarService(GlobalConfigManager, GlobalTokenManager, cortezaClient)

	// Get config first
	fmt.Println("Triggering config push from Corteza...")
	if err := cortezaClient.TriggerConfigPush(); err != nil {
		return fmt.Errorf("failed to trigger config push: %w", err)
	}

	fmt.Println("Waiting for config from Corteza...")
	config, err := service.configManager.WaitForConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	// Try to get token from Corteza first
	fmt.Println("Triggering token push from Corteza...")
	if err := cortezaClient.TriggerTokenPush(); err != nil {
		fmt.Printf("Warning: failed to trigger token push: %v\n", err)
	}

	// Wait for token with short timeout
	tokenCtx, tokenCancel := context.WithTimeout(ctx, 10*time.Second)
	defer tokenCancel()

	fmt.Println("Waiting for token from Corteza...")
	token, err := service.tokenManager.WaitForToken(tokenCtx)
	if err != nil || token == nil || token.AccessToken == "" {
		fmt.Println("No valid token from Corteza, getting fresh one from Yeastar...")
		token, err = GlobalTokenManager.GetNewToken(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to get token from Yeastar: %w", err)
		}

		fmt.Println("Saving fresh token to Corteza...")
		if err := cortezaClient.SaveToken(ctx, token); err != nil {
			return fmt.Errorf("failed to save token to Corteza: %w", err)
		}

		// Set the token directly since we just got it
		GlobalTokenManager.SetToken(token)
	}

	// Verify we have a valid token
	if !GlobalTokenManager.IsTokenValid() {
		return fmt.Errorf("no valid token available after all attempts")
	}

	// ----------- Begin Data Processing -----------

	// Agents
	fmt.Println("\n--- Processing agents ---")
	rawAgentsData, err := service.ListMethod(ctx, "extension")
	if err != nil {
		return fmt.Errorf("failed to fetch agents: %w", err)
	}

	agents, err := processAgentsData(rawAgentsData)
	if err != nil {
		return fmt.Errorf("failed to process agents: %w", err)
	}

	if err := service.SendDataToCorteza(ctx, "agent", agents); err != nil {
		return fmt.Errorf("failed to send agents to Corteza: %w", err)
	}
	fmt.Println("agents processed and sent to Corteza successfully!")

	// Queues
	fmt.Println("\n--- Processing queues ---")
	rawQueuesData, err := service.ListMethod(ctx, "queue")
	if err != nil {
		return fmt.Errorf("failed to fetch queues: %w", err)
	}

	queues, err := processQueuesData(rawQueuesData)
	if err != nil {
		return fmt.Errorf("failed to process queues: %w", err)
	}

	if err := service.SendDataToCorteza(ctx, "queue", queues); err != nil {
		return fmt.Errorf("failed to send queues to Corteza: %w", err)
	}
	fmt.Println("queues processed and sent to Corteza successfully!")

	// Members
	fmt.Println("\n--- Processing queue members ---")
	members, err := processQueueMembersData(rawQueuesData)
	if err != nil {
		return fmt.Errorf("failed to process queue members: %w", err)
	}

	if err := service.SendDataToCorteza(ctx, "member", members); err != nil {
		return fmt.Errorf("failed to send queue members to Corteza: %w", err)
	}
	fmt.Println("queue members processed and sent to Corteza successfully!")

	// CDRs
	fmt.Println("\n--- Processing cdrs ---")
	rawCDRsData, err := service.ListMethod(ctx, "cdr")
	if err != nil {
		return fmt.Errorf("failed to fetch cdrs: %w", err)
	}

	cdrs, err := processCDRsData(service, rawCDRsData)
	if err != nil {
		return fmt.Errorf("failed to process cdrs: %w", err)
	}

	if err := dumpCDRsToFile(cdrs); err != nil {
		log.Printf("Warning: failed to dump CDRs to file: %v", err)
	}

	if err := service.SendDataToCorteza(ctx, "cdr", cdrs); err != nil {
		return fmt.Errorf("failed to send cdrs to Corteza: %w", err)
	}
	fmt.Println("cdrs processed and sent to Corteza successfully!")

	return nil
}
