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
	// Initialize global managers
	InitializeGlobalManagers()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create Corteza client
	cortezaClient := NewCortezaClient("http://localhost:80")

	// Initialize service
	service := NewYeastarService(GlobalConfigManager, GlobalTokenManager, cortezaClient)

RESTART_SYNC:
	// Trigger config and token push
	fmt.Println("Triggering config and token push from Corteza...")
	if err := cortezaClient.TriggerConfigPush(); err != nil {
		return fmt.Errorf("failed to trigger config push: %w", err)
	}

	if err := cortezaClient.TriggerTokenPush(); err != nil {
		return fmt.Errorf("failed to trigger token push: %w", err)
	}

	// Wait for initialization
	fmt.Println("Waiting for config and token from Corteza...")
	if err := service.WaitForInitialization(ctx); err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}
	fmt.Println("Service initialized successfully!")

	// Check if token is valid
	if !GlobalTokenManager.IsTokenValid() {
		fmt.Println("❌ Token from Corteza is already expired. Fetching new token from Yeastar...")

		cfg := GlobalConfigManager.GetConfig()
		if cfg == nil {
			return fmt.Errorf("config not available for token refresh")
		}

		newToken, err := GlobalTokenManager.GetNewToken(ctx, cfg)
		if err != nil {
			return fmt.Errorf("failed to fetch new token from Yeastar: %w", err)
		}

		GlobalTokenManager.SetToken(newToken)

		if err := cortezaClient.SaveToken(ctx, newToken); err != nil {
			return fmt.Errorf("failed to push new token to Corteza: %w", err)
		}

		fmt.Println("✅ New token pushed to Corteza. Restarting sync process...")
		goto RESTART_SYNC
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
	fmt.Println("✅ agents processed and sent to Corteza successfully!")

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
	fmt.Println("✅ queues processed and sent to Corteza successfully!")

	// Members
	fmt.Println("\n--- Processing queue members ---")
	members, err := processQueueMembersData(rawQueuesData)
	if err != nil {
		return fmt.Errorf("failed to process queue members: %w", err)
	}

	if err := service.SendDataToCorteza(ctx, "member", members); err != nil {
		return fmt.Errorf("failed to send queue members to Corteza: %w", err)
	}
	fmt.Println("✅ queue members processed and sent to Corteza successfully!")

	// CDRs
	fmt.Println("\n--- Processing cdrs ---")
	rawCDRsData, err := service.ListMethod(ctx, "cdr")
	if err != nil {
		return fmt.Errorf("failed to fetch cdrs: %w", err)
	}

	cdrs, err := processCDRsData(rawCDRsData)
	if err != nil {
		return fmt.Errorf("failed to process cdrs: %w", err)
	}

	if err := service.SendDataToCorteza(ctx, "cdr", cdrs); err != nil {
		return fmt.Errorf("failed to send cdrs to Corteza: %w", err)
	}
	fmt.Println("✅ cdrs processed and sent to Corteza successfully!")

	return nil
}
