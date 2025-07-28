package yeastarEvents

import (
	"context"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// YeastarIntegration holds our Yeastar components
type YeastarIntegration struct {
	eventReceiver  *EventReceiver
	eventProcessor *EventProcessor
	cortezaSender  *CortezaSender
	log            *zap.Logger
}

func NewYeastarIntegration(log *zap.Logger) *YeastarIntegration {
	// Create components
	cortezaSender := NewCortezaSender("http://your-corteza-url.com")
	eventProcessor := NewEventProcessor(cortezaSender)
	eventReceiver := NewEventReceiver(eventProcessor)

	return &YeastarIntegration{
		eventReceiver:  eventReceiver,
		eventProcessor: eventProcessor,
		cortezaSender:  cortezaSender,
		log:            log.Named("yeastar"),
	}
}

// Middleware returns a chi router middleware function
func (yi *YeastarIntegration) Middleware() func(chi.Router) {
	return func(r chi.Router) {
		// Register Yeastar webhook routes
		yi.eventReceiver.RegisterRoutes(r)

		yi.log.Info("Yeastar webhook routes registered")
	}
}

// Start begins the background processing
func (yi *YeastarIntegration) Start(ctx context.Context) error {
	yi.log.Info("Starting Yeastar integration...")

	// Start event processor
	go yi.eventProcessor.Start(ctx)

	// Start event receiver (WebSocket only)
	go yi.eventReceiver.Start(ctx)

	yi.log.Info("Yeastar integration started")
	return nil
}
