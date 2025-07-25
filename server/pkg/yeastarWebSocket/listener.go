package yeastar

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

func StartListener(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/integrations/yeastar/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var evt CallEvent
		if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		zap.L().Info("Received Yeastar event", zap.Any("event", evt))

		// Optional: trigger internal processing, workflows, etc.

		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    ":9090", // configure this if needed
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}
