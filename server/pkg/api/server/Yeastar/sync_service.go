package Yeastar

import (
	"context"
	"fmt"
	"io"
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
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	protocol := "http"
	if r.TLS != nil {
		protocol = "https"
	}

	baseURL := fmt.Sprintf("%s://%s", protocol, r.Host)

	go func() {
		if err := SyncAll(baseURL); err != nil {
			log.Printf("Error during full sync: %v", err)
			return
		}
		log.Println("Full synchronization process completed successfully.")
	}()

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Synchronization process initiated successfully. Check logs for details."))
	log.Println("Full synchronization process completed successfully.")
}

func setupSyncService(baseUrl string) (*YeastarService, context.Context, context.CancelFunc, error) {
	InitializeGlobalManagers()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	cortezaClient := NewCortezaClient(baseUrl)
	service := NewYeastarService(GlobalConfigManager, GlobalTokenManager, cortezaClient)

	return service, ctx, cancel, nil
}

func setupAuth(ctx context.Context, service *YeastarService) error {
	fmt.Println("[setupAuth] Getting config from Corteza...")
	config, err := service.cortezaClient.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	service.configManager.SetConfig(config)

	fmt.Println("[setupAuth] Getting token from Corteza...")
	token, err := service.cortezaClient.GetToken()
	if err != nil {
		fmt.Printf("[setupAuth] Failed to get token from Corteza: %v\n", err)
		token = nil
	}

	if token != nil && time.Now().Unix() < int64(token.AccessTokenExpireTime) {
		fmt.Printf("Token set. Access token expires in: %ds\n", int64(token.AccessTokenExpireTime)-time.Now().Unix())
		service.tokenManager.SetToken(token)
	} else {
		fmt.Println("[setupAuth] Token is missing or expired, getting fresh token from Yeastar...")
		token, err = service.tokenManager.GetNewToken(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to get new token from Yeastar: %w", err)
		}

		service.tokenManager.SetToken(token)

		fmt.Println("[setupAuth] Saving new token to Corteza...")
		if err := service.cortezaClient.SaveToken(ctx, token); err != nil {
			fmt.Printf("[setupAuth] Warning: failed to save token to Corteza: %v\n", err)
		}

		fmt.Printf("New token set. Access token expires in: %ds\n", int64(token.AccessTokenExpireTime)-time.Now().Unix())
	}

	if !service.tokenManager.IsTokenValid() {
		return fmt.Errorf("no valid token available after all attempts")
	}

	fmt.Println("[setupAuth] Auth setup completed successfully.")
	return nil
}

func syncAgents(ctx context.Context, service *YeastarService) error {
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
	return nil
}

func syncQueues(ctx context.Context, service *YeastarService) error {
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
	return nil
}

func syncQueueMembers(ctx context.Context, service *YeastarService) error {
	fmt.Println("\n--- Processing queue members ---")
	rawQueuesData, err := service.ListMethod(ctx, "queue")
	if err != nil {
		return fmt.Errorf("failed to fetch queues: %w", err)
	}

	members, err := processQueueMembersData(rawQueuesData)
	if err != nil {
		return fmt.Errorf("failed to process queue members: %w", err)
	}

	if err := service.SendDataToCorteza(ctx, "member", members); err != nil {
		return fmt.Errorf("failed to send queue members to Corteza: %w", err)
	}
	fmt.Println("queue members processed and sent to Corteza successfully!")
	return nil
}

func syncCDRs(ctx context.Context, service *YeastarService) error {
	fmt.Println("\n--- Processing cdrs ---")
	rawCDRsData, err := service.ListMethod(ctx, "cdr")
	if err != nil {
		return fmt.Errorf("failed to fetch cdrs: %w", err)
	}

	cdrs, err := processCDRsData(service, rawCDRsData)
	if err != nil {
		return fmt.Errorf("failed to process cdrs: %w", err)
	}

	batchSize := 20
	totalCDRs := len(cdrs)
	processed := 0
	var failedBatches []int

	for i := 0; i < totalCDRs; i += batchSize {
		end := i + batchSize
		if end > totalCDRs {
			end = totalCDRs
		}

		batch := cdrs[i:end]

		batchCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
		defer cancel()

		if err := service.SendDataToCorteza(batchCtx, "cdr", batch); err != nil {
			log.Printf("Error sending CDRs batch [%d-%d] to Corteza: %v", i+1, end, err)
			failedBatches = append(failedBatches, i/batchSize+1)
			continue
		}

		processed += len(batch)
		fmt.Printf("Processed batch %d-%d of %d (%.1f%%)\n",
			i+1, end, totalCDRs, float64(processed)/float64(totalCDRs)*100)

		time.Sleep(1 * time.Second)
	}

	fmt.Println("cdrs processed and sent to Corteza (with possible errors).")
	if len(failedBatches) > 0 {
		return fmt.Errorf("failed batches: %v", failedBatches)
	}
	return nil
}

func SyncAll(baseUrl string) error {
	service, ctx, cancel, err := setupSyncService(baseUrl)
	if err != nil {
		return err
	}
	defer cancel()

	if err := setupAuth(ctx, service); err != nil {
		return err
	}

	if err := syncAgents(ctx, service); err != nil {
		return err
	}

	if err := syncQueues(ctx, service); err != nil {
		return err
	}

	if err := syncQueueMembers(ctx, service); err != nil {
		return err
	}

	if err := syncCDRs(ctx, service); err != nil {
		return err
	}

	if err := service.cortezaClient.CallCDRCalc(); err != nil {
		return err
	}

	return nil
}

func (cc *CortezaClient) CallCDRCalc() error {
	url := fmt.Sprintf("%s/api/gateway/cdr/calc", cc.baseURL)
	fmt.Printf("Cdr Calc Endpoint: %s\n", url)

	req, err := cc.client.Get(url)
	if err != nil {
		fmt.Printf("Failed to create HTTP request: %v\n", err)
		return fmt.Errorf("Failed to create HTTP request: %v\n", err)
	}
	defer req.Body.Close()

	if req.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(req.Body)
		return fmt.Errorf("unexpected status %d: %s", req.StatusCode, string(body))
	}

	fmt.Printf("[Sync_Service] CDR Calculated called complete")
	return nil
}

// Now you can easily create individual sync functions
func SyncAgentsOnly(baseUrl string) error {
	service, ctx, cancel, err := setupSyncService(baseUrl)
	if err != nil {
		return err
	}
	defer cancel()

	if err := setupAuth(ctx, service); err != nil {
		return err
	}

	return syncAgents(ctx, service)
}

func SyncQueuesOnly(baseUrl string) error {
	service, ctx, cancel, err := setupSyncService(baseUrl)
	if err != nil {
		return err
	}
	defer cancel()

	if err := setupAuth(ctx, service); err != nil {
		return err
	}

	return syncQueues(ctx, service)
}

func SyncCDRsOnly(baseUrl string) error {
	service, ctx, cancel, err := setupSyncService(baseUrl)
	if err != nil {
		return err
	}
	defer cancel()

	if err := setupAuth(ctx, service); err != nil {
		return err
	}

	return syncCDRs(ctx, service)
}

func findCDRsByUID(cdrs []CDR, uid string) ([]CDR, error) {
	var results []CDR
	for _, cdr := range cdrs {
		if cdr.UID == uid {
			results = append(results, cdr)
		}
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no CDRs found with UID %s", uid)
	}
	return results, nil
}

func StartPeriodicSync() error {
	ip, err := getLocalIP()
	if err != nil {
		return fmt.Errorf("Could not get local IP")
	}

	baseURL := fmt.Sprintf("http://%s", ip)

	// Call your existing HandleSyncAll logic
	err = SyncAll(baseURL)

	if err != nil {
		return fmt.Errorf("Synchronization failed")
	}

	return nil
}
