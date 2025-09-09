package Yeastar

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
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
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

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

	batchSize := 10
	totalCDRs := len(cdrs)
	processed := 0

	log.Printf("Total CDRs to process: %d in batches of %d", totalCDRs, batchSize)

	for i := 0; i < totalCDRs; i += batchSize {
		end := i + batchSize
		if end > totalCDRs {
			end = totalCDRs
		}

		batch := cdrs[i:end]
		//batchNum := i/batchSize + 1
		batchCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// if err := processBatchWithRetry(ctx, service, batch, batchNum, i+1, end); err != nil {
		// 	log.Printf("Batch %d [%d-%d] failed after retries: %v", batchNum, i+1, end, err)
		// 	continue
		// }

		if err := service.SendDataToCorteza(batchCtx, "cdr", batch); err != nil {
			log.Printf("Error sending CDRs batch [%d-%d] to Corteza: %v", i+1, end, err)
			//failedBatches = append(failedBatches, i/batchSize+1)
			continue
		}

		processed += len(batch)
		fmt.Printf("Processed batch %d-%d of %d (%.1f%%)\n", i+1, end, totalCDRs, float64(processed)/float64(totalCDRs)*100)

		time.Sleep(2 * time.Second)
	}

	fmt.Println("CDRs processing completed.")
	log.Printf("✓ All batches attempted. %d/%d CDRs processed successfully", processed, totalCDRs)

	return nil
}

// func syncCDRs(ctx context.Context, service *YeastarService) error {
// 	fmt.Println("\n--- Processing cdrs ---")
// 	rawCDRsData, err := service.ListMethod(ctx, "cdr")
// 	if err != nil {
// 		return fmt.Errorf("failed to fetch cdrs: %w", err)
// 	}

// 	cdrs, err := processCDRsData(service, rawCDRsData)
// 	if err != nil {
// 		return fmt.Errorf("failed to process cdrs: %w", err)
// 	}

// 	batchSize := 10
// 	totalCDRs := len(cdrs)
// 	processed := 0
// 	var failedBatches []BatchError

// 	log.Printf("Total CDRs to process: %d in batches of %d", totalCDRs, batchSize)

// 	for i := 0; i < totalCDRs; i += batchSize {
// 		end := i + batchSize
// 		if end > totalCDRs {
// 			end = totalCDRs
// 		}

// 		batch := cdrs[i:end]
// 		batchNum := i/batchSize + 1

// 		if err := processBatchWithRetry(ctx, service, batch, batchNum, i+1, end); err != nil {
// 			failedBatches = append(failedBatches, BatchError{
// 				BatchNum: batchNum,
// 				Range:    fmt.Sprintf("%d-%d", i+1, end),
// 				Error:    err,
// 			})
// 			log.Printf("Batch %d [%d-%d] failed after all retries: %v", batchNum, i+1, end, err)
// 			continue
// 		}

// 		processed += len(batch)
// 		fmt.Printf("✓ Processed batch %d-%d of %d (%.1f%%)\n", i+1, end, totalCDRs, float64(processed)/float64(totalCDRs)*100)

// 		time.Sleep(2 * time.Second)
// 	}

// 	fmt.Println("CDRs processing completed.")

// 	if len(failedBatches) > 0 {
// 		log.Printf("Summary: %d/%d batches failed", len(failedBatches), (totalCDRs+batchSize-1)/batchSize)
// 		for _, fb := range failedBatches {
// 			log.Printf("  - Batch %d [%s]: %v", fb.BatchNum, fb.Range, fb.Error)
// 		}
// 		return fmt.Errorf("failed to process %d batches out of %d", len(failedBatches), (totalCDRs+batchSize-1)/batchSize)
// 	}

// 	log.Printf("✓ All batches processed successfully (%d CDRs)", totalCDRs)
// 	return nil
// }

// type BatchError struct {
// 	BatchNum int
// 	Range    string
// 	Error    error
// }

// func processBatchWithRetry(ctx context.Context, service *YeastarService, batch interface{}, batchNum, start, end int) error {
// 	const maxRetries = 3
// 	const baseDelay = 5 * time.Second
// 	const requestTimeout = 90 * time.Second

// 	for attempt := 1; attempt <= maxRetries; attempt++ {
// 		if ctx.Err() != nil {
// 			return fmt.Errorf("parent context cancelled before attempt %d: %w", attempt, ctx.Err())
// 		}

// 		batchCtx, cancel := context.WithTimeout(context.Background(), requestTimeout)

// 		log.Printf("Sending batch %d [%d-%d] (attempt %d/%d)", batchNum, start, end, attempt, maxRetries)

// 		err := service.SendDataToCorteza(batchCtx, "cdr", batch)
// 		cancel()

// 		if err == nil {
// 			return nil
// 		}

// 		log.Printf("Batch %d [%d-%d] attempt %d failed: %v", batchNum, start, end, attempt, err)

// 		if ctx.Err() != nil {
// 			return fmt.Errorf("parent context cancelled after attempt %d: %w", attempt, ctx.Err())
// 		}

// 		if attempt < maxRetries {
// 			delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt-1)))
// 			if delay > 30*time.Second {
// 				delay = 30 * time.Second
// 			}

// 			log.Printf("Retrying batch %d in %v...", batchNum, delay)

// 			select {
// 			case <-time.After(delay):
// 			case <-ctx.Done():
// 				return fmt.Errorf("parent context cancelled during retry wait: %w", ctx.Err())
// 			}
// 		}
// 	}

// 	return fmt.Errorf("batch failed after %d attempts", maxRetries)
// }

var syncInProgress int32

func SyncAll(baseUrl string) error {
	if !atomic.CompareAndSwapInt32(&syncInProgress, 0, 1) {
		return errors.New("sync already in progress")
	}
	defer atomic.StoreInt32(&syncInProgress, 0)

	fmt.Println("Starting sync...")

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

	err = SyncAll(baseURL)

	if err != nil {
		return fmt.Errorf("Synchronization failed")
	}

	return nil
}
