package server

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/cortezaproject/corteza/server/pkg/api/server/Yeastar"
	"github.com/cortezaproject/corteza/server/pkg/options"
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

		demux   *demux
		yeastar *Yeastar.YeastarIntegration
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

	s.yeastar = Yeastar.NewYeastarIntegration(
		log.Named("yeastar"),
		opts.HTTPServer.BaseUrl, // Use HTTP server's BaseURL
	)

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

func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// Activate reconfigures server to use active routes
func (s *server) Activate(mm ...func(chi.Router)) {
	s.demux.Router(active, activeRoutes(s.log, mm, s.opts))

	s.log.Debug("entering active state")
	s.demux.State(active)

	go func() {
		ip, err := getLocalIP()
		if err != nil {
			s.log.Error("Could not get local IP", zap.Error(err))
			return
		}

		if err := s.yeastar.Start(context.Background(), ip); err != nil {
			s.log.Error("Failed to start Yeastar integration", zap.Error(err))
		}
	}()
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

	go func() {
		for {
			select {
			case <-ctx.Done():
				s.log.Info("Stopping periodic sync")
				return
			default:
				start := time.Now()

				s.log.Info("Starting periodic sync")
				if err := Yeastar.StartPeriodicSync(); err != nil {
					s.log.Error("Periodic sync failed", zap.Error(err))
				}

				s.log.Info("Periodic sync completed",
					zap.Duration("duration", time.Since(start)),
				)

				// Wait 10 minutes after the work is finished
				select {
				case <-time.After(10 * time.Minute):
					// continue loop
				case <-ctx.Done():
					s.log.Info("Stopping periodic sync")
					return
				}
			}
		}
	}()

	<-ctx.Done()

	if s.yeastar != nil {
		if monitor := s.yeastar.GetEventMonitor(); monitor != nil {
			if err := monitor.Stop(); err != nil {
				s.log.Error("Failed to stop Yeastar event monitor", zap.Error(err))
			}
		}
	}

	if s.err == nil {
		s.err = ctx.Err()
		if s.err == context.Canceled {
			s.err = nil
		}
	}

	s.log.Info("HTTP server stopped", zap.Error(s.err))
}
