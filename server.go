package main

import (
	"crypto/rand"
	"crypto/tls"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/handlers"
)

//go:embed static/*
var staticContent embed.FS

//go:embed templates/*.gohtml
var templatesContent embed.FS

type ServerOpts struct {
	Host string
	Port int
}

func (s *ServerOpts) OK() error {
	switch {
	case s.Host == "":
		return fmt.Errorf("must supply Host")
	case s.Port == 0:
		return fmt.Errorf("must supply Port")
	}

	return nil
}

type Server struct {
	Opts     *ServerOpts
	Mux      *http.ServeMux
	Server   *http.Server
	Renderer *TemplateRenderer
	Rolls    Rolls
	Stats    *Stats

	secretKey   []byte
	rollMutex   sync.Mutex
	clientMutex sync.RWMutex
	clients     map[chan EventMessage]bool
}

func NewServer(opts *ServerOpts) (*Server, error) {
	if err := opts.OK(); err != nil {
		return nil, fmt.Errorf("invalid Server options: %w", err)
	}

	// generate a random secret key
	secretKey := make([]byte, 32)
	n, err := rand.Read(secretKey)
	if n != 32 {
		return nil, fmt.Errorf("invalid number of secret key bytes (%d)", n)
	} else if err != nil {
		return nil, fmt.Errorf("failed to generate secret key: %w", err)
	}

	mux := http.NewServeMux()

	renderer, err := NewTemplateRenderer()
	if err != nil {
		return nil, fmt.Errorf("failed to set up template renderer: %w", err)
	}

	server := &Server{
		Opts: opts,
		Mux:  mux,
		Server: &http.Server{
			Addr:        fmt.Sprintf("%s:%d", opts.Host, opts.Port),
			Handler:     handlers.CombinedLoggingHandler(os.Stdout, mux),
			ReadTimeout: time.Second * 5,
			TLSConfig: &tls.Config{
				CipherSuites: []uint16{
					//  TLSv1.3
					tls.TLS_AES_128_GCM_SHA256,
					tls.TLS_AES_256_GCM_SHA384,
					tls.TLS_CHACHA20_POLY1305_SHA256,
					//  TLSv1.2
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
					tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				},
				MinVersion: tls.VersionTLS12,
				MaxVersion: tls.VersionTLS13,
			},
		},
		Renderer: renderer,
		Stats: &Stats{
			Momentum: 0,
			Threat:   0,
		},

		secretKey:   secretKey,
		rollMutex:   sync.Mutex{},
		clientMutex: sync.RWMutex{},
		clients:     make(map[chan EventMessage]bool),
	}

	if err := server.setupRoutes(); err != nil {
		return nil, fmt.Errorf("failed to set up routes: %w", err)
	}

	return server, nil
}

func (s *Server) setupRoutes() error {
	staticFS, err := fs.Sub(staticContent, "static")
	if err != nil {
		return fmt.Errorf("failed to set up static files: %w", err)
	}

	s.Mux.HandleFunc("/", s.UserMiddleware(false, s.IndexHandler))
	s.Mux.HandleFunc("GET /dice", s.UserMiddleware(true, s.DiceHandler))
	s.Mux.HandleFunc("GET /sse", s.UserMiddleware(true, s.SSEHandler))
	s.Mux.HandleFunc("GET /history", s.UserMiddleware(true, s.HistoryHandler))
	s.Mux.HandleFunc("GET /stats", s.UserMiddleware(true, s.StatsHandler))
	s.Mux.HandleFunc("POST /roll", s.UserMiddleware(true, s.RollHandler))
	s.Mux.HandleFunc("POST /private-roll", s.UserMiddleware(true, s.GameMasterMiddleware(s.PrivateRollHandler)))
	s.Mux.HandleFunc("POST /game-master", s.UserMiddleware(true, s.GameMasterMiddleware(s.GameMasterHandler)))
	s.Mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	return nil
}

func (s *Server) Start() error {
	fmt.Printf("Listening on %s:%d...\n", s.Opts.Host, s.Opts.Port)

	if err := s.Server.ListenAndServe(); err != nil {
		return fmt.Errorf("listen error: %w", err)
	}

	return nil
}
