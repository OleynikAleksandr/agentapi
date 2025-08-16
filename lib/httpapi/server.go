package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/coder/agentapi/lib/logctx"
	"github.com/coder/agentapi/lib/termexec"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

// Server represents the HTTP server for raw terminal output
type Server struct {
	router    chi.Router
	api       huma.API
	port      int
	srv       *http.Server
	mu        sync.RWMutex
	logger    *slog.Logger
	process   *termexec.Process
}

func (s *Server) GetOpenAPI() string {
	jsonBytes, err := s.api.OpenAPI().MarshalJSON()
	if err != nil {
		return ""
	}
	// unmarshal the json and pretty print it
	var jsonObj any
	if err := json.Unmarshal(jsonBytes, &jsonObj); err != nil {
		return ""
	}
	prettyJSON, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		return ""
	}
	return string(prettyJSON)
}

// NewServer creates a new server instance - RAW TERMINAL ONLY
func NewServer(ctx context.Context, agentType string, process *termexec.Process, port int, chatBasePath string) *Server {
	router := chi.NewMux()

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	router.Use(corsMiddleware.Handler)

	humaConfig := huma.DefaultConfig("AgentAPI RAW", "1.0.0")
	humaConfig.Info.Description = "Raw terminal output API - no parsing, no filtering"
	api := humachi.New(router, humaConfig)

	s := &Server{
		router:  router,
		api:     api,
		port:    port,
		logger:  logctx.From(ctx),
		process: process,
	}

	// Register only essential routes
	s.registerRoutes()

	return s
}

// Handler returns the underlying chi.Router
func (s *Server) Handler() http.Handler {
	return s.router
}

// registerRoutes sets up only essential endpoints
func (s *Server) registerRoutes() {
	// GET /terminal - returns raw terminal output
	huma.Get(s.api, "/terminal", s.getTerminal, func(o *huma.Operation) {
		o.Description = "Returns raw terminal output without any processing"
	})

	// GET /messages - for backward compatibility, returns raw terminal as text/plain
	s.router.Get("/messages", s.getMessagesPlain)

	// POST /message - send raw input to terminal (with sessionId in query)
	s.router.Post("/message", s.sendMessageCompat)

	// GET /status - simple health check
	huma.Get(s.api, "/status", s.getStatus, func(o *huma.Operation) {
		o.Description = "Health check endpoint"
	})
}

// TerminalResponse is the response for terminal output
type TerminalResponse struct {
	Body struct {
		Content string `json:"content" description:"Raw terminal output"`
	}
}

// getTerminal returns raw terminal output
func (s *Server) getTerminal(ctx context.Context, input *struct{}) (*TerminalResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get raw screen content from the process
	rawOutput := s.process.ReadScreen()

	resp := &TerminalResponse{}
	resp.Body.Content = rawOutput

	return resp, nil
}

// getMessagesPlain - legacy endpoint, returns raw terminal as plain text
func (s *Server) getMessagesPlain(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get raw screen content from the process
	rawOutput := s.process.ReadScreen()

	// Return as plain text for backward compatibility with extension
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(rawOutput))
}

// MessageRequest for sending input - compatible with old format
type MessageRequest struct {
	Body struct {
		Content string `json:"content" required:"true" description:"Text to send to terminal"`
		Type    string `json:"type" description:"Message type (user/raw)"`
	}
}

// MessageResponse
type MessageResponse struct {
	Body struct {
		Success bool `json:"success"`
	}
}

// sendMessage sends raw text to terminal (Huma version for API docs)
func (s *Server) sendMessage(ctx context.Context, input *MessageRequest) (*MessageResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Send raw text directly to the process
	_, err := s.process.Write([]byte(input.Body.Content))
	if err != nil {
		return nil, err
	}

	resp := &MessageResponse{}
	resp.Body.Success = true

	return resp, nil
}

// sendMessageCompat - backward compatible version for extension
func (s *Server) sendMessageCompat(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Parse JSON body
	var reqBody struct {
		Content string `json:"content"`
		Type    string `json:"type"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Send raw text directly to the process
	_, err := s.process.Write([]byte(reqBody.Content))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// StatusResponse for health check
type StatusResponse struct {
	Body struct {
		Status string `json:"status"`
	}
}

// getStatus returns simple status
func (s *Server) getStatus(ctx context.Context, input *struct{}) (*StatusResponse, error) {
	resp := &StatusResponse{}
	resp.Body.Status = "running"
	return resp, nil
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.port)
	s.srv = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	s.logger.Info("starting raw terminal server", "address", s.srv.Addr)

	go func() {
		<-ctx.Done()
		_ = s.srv.Shutdown(context.Background())
	}()

	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Stop stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}
	return nil
}