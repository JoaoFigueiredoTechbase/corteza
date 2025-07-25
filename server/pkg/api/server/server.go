package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/cortezaproject/corteza/server/pkg/options"
	yeastar "github.com/cortezaproject/corteza/server/pkg/yeastarWebSocket"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type (
	server struct {
		log       *zap.Logger
		opts      *options.Options
		endpoints []func(r chi.Router)

		// last error
		err error

		demux *demux
	}
)

const (
	waiting uint32 = iota
	active
	shutdown
)

// New initializes new HTTP server with special powers
// Server is started as early as possible and with a special request handler
// that demultiplexes request to one of the configured routers according to the server state.
//
// Waiting state
// This is initial state with the ofllowing route handlers:
//  - /version
//  - /healthcheck

func New(log *zap.Logger, opts *options.Options) *server {
	s := &server{
		endpoints: make([]func(r chi.Router), 0),
		log:       log.Named("http"),

		opts: opts,
	}

	s.demux = Demux(waiting, waitingRoutes(s.log.Named("waiting"), s.opts.HTTPServer))
	s.demux.Router(shutdown, shutdownRoutes())

	return s
}

func (s *server) LastError() error {
	return s.err
}

func Test(o *options.Options) error {
	listener, err := net.Listen("tcp", o.HTTPServer.Addr)
	if err != nil {
		return err
	}

	if err = listener.Close(); err != nil {
		return err
	}

	return nil
}

// Activate reconfigures server to use active routes
func (s *server) Activate(mm ...func(chi.Router)) {
	s.demux.Router(active, activeRoutes(s.log, mm, s.opts))

	s.log.Debug("entering active state")
	s.demux.State(active)
}

// Shutdown reconfigures server to use shutdown routes
func (s *server) Shutdown() {
	s.log.Debug("entering shutdown state")
	s.demux.State(shutdown)
}

func (s server) Serve(ctx context.Context) {
	var (
		listener net.Listener
	)

	yeastar.InitializeGlobalManagers()
	yeastarClient := yeastar.NewClient(s.opts.HTTPServer.BaseUrl, "")
	processor := yeastar.NewEventProcessor(s.log, yeastarClient)

	processor.AddHandler(30011, func(event map[string]interface{}) error {
		// Process call status event
		msg, ok := event["msg"].(string)
		if !ok {
			return fmt.Errorf("missing msg field")
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(msg), &data); err != nil {
			return err
		}

		return processor.BroadcastEvent(ctx, "call.status", data)
	})

	processor.AddHandler(30012, func(event map[string]interface{}) error {
		// Process CDR event
		return processor.BroadcastEvent(ctx, "cdr", event)
	})

	s.log.Info(
		"starting HTTP server",
		zap.String("path-prefix", s.opts.HTTPServer.BaseUrl),
		zap.String("address", s.opts.HTTPServer.Addr),
	)

	listener, s.err = net.Listen("tcp", s.opts.HTTPServer.Addr)
	if s.err != nil {
		s.log.Error("cannot start server", zap.Error(s.err))
		return
	}

	go func() {
		srv := http.Server{
			Handler: s.demux,

			// use root context as server's base context and as a basis for
			// context for all requests
			// this enables us to send cancellation down to every request
			BaseContext: func(listener net.Listener) context.Context { return ctx },
		}
		s.err = srv.Serve(listener)
	}()

	processor.Start(ctx)

	go func() {
		s.log.Info("Starting Yeastar WebSocket client...")
		if err := yeastar.StartWebSocketClient(ctx, processor); err != nil {
			s.log.Error("Yeastar WebSocket client error", zap.Error(err))
		}
	}()

	go func() {
		s.log.Info("Starting Yeastar listener...")
		if err := yeastar.StartListener(ctx); err != nil {
			s.log.Error("Yeastar listener error", zap.Error(err))
		}
	}()

	<-ctx.Done()

	if s.err == nil {
		s.err = ctx.Err()
		if s.err == context.Canceled {
			s.err = nil
		}
	}

	s.log.Info("HTTP server stopped", zap.Error(s.err))
}
