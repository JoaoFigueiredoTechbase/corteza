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

	// Process different data types
	dataTypes := []struct {
		endpoint   string
		moduleName string
		processor  func([]byte) (interface{}, error)
	}{
		{
			endpoint:   "extension",
			moduleName: "agents",
			processor: func(data []byte) (interface{}, error) {
				agents, err := processAgentsData(data)
				return agents, err
			},
		},
		{
			endpoint:   "queue",
			moduleName: "queues",
			processor: func(data []byte) (interface{}, error) {
				queues, err := processQueuesData(data)
				return queues, err
			},
		},
		{
			endpoint:   "cdr",
			moduleName: "cdrs",
			processor: func(data []byte) (interface{}, error) {
				cdrs, err := processCDRsData(data)
				return cdrs, err
			},
		},
	}

	for _, dt := range dataTypes {
		fmt.Printf("\n--- Processing %s ---\n", dt.moduleName)

		// Fetch data
		rawData, err := service.ListMethod(ctx, dt.endpoint)
		if err != nil {
			return fmt.Errorf("failed to fetch %s: %w", dt.moduleName, err)
		}

		// Process data
		processedData, err := dt.processor(rawData)
		if err != nil {
			return fmt.Errorf("failed to process %s: %w", dt.moduleName, err)
		}

		// Send to Corteza
		if err := service.SendDataToCorteza(ctx, dt.moduleName, processedData); err != nil {
			return fmt.Errorf("failed to send %s to Corteza: %w", dt.moduleName, err)
		}

		fmt.Printf("✅ %s processed and sent to Corteza successfully!\n", dt.moduleName)
	}

	return nil
}
