package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	// "net/url" // removed - was used for WebUI
	// "strings" // removed - was used for WebUI
	"sync"
	"time"

	"github.com/coder/agentapi/lib/logctx"
	mf "github.com/coder/agentapi/lib/msgfmt"
	st "github.com/coder/agentapi/lib/screentracker"
	"github.com/coder/agentapi/lib/termexec"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/danielgtaylor/huma/v2/sse"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"golang.org/x/xerrors"
)

// Server represents the HTTP server
type Server struct {
	router       chi.Router
	api          huma.API
	port         int
	srv          *http.Server
	mu           sync.RWMutex
	logger       *slog.Logger
	conversation *st.Conversation
	agentio      *termexec.Process
	agentType    mf.AgentType
	emitter      *EventEmitter
	splitter     *mf.ServiceSplitter
	// Store last sent chat content for delta calculation
	lastSentChat string
	// chatBasePath string // removed - no WebUI
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

// That's about 40 frames per second. It's slightly less
// because the action of taking a snapshot takes time too.
const snapshotInterval = 25 * time.Millisecond

type ServerConfig struct {
	AgentType    mf.AgentType
	Process      *termexec.Process
	Port         int
	// ChatBasePath string // removed - no WebUI
}

// NewServer creates a new server instance
func NewServer(ctx context.Context, config ServerConfig) *Server {
	router := chi.NewMux()

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
	router.Use(corsMiddleware.Handler)

	humaConfig := huma.DefaultConfig("AgentAPI", "v1.6.0")
	humaConfig.Info.Description = "HTTP API for Claude Code, Goose, and Aider.\n\nhttps://github.com/coder/agentapi"
	api := humachi.New(router, humaConfig)
	formatMessage := func(message string, userInput string) string {
		return mf.FormatAgentMessage(config.AgentType, message, userInput)
	}
	conversation := st.NewConversation(ctx, st.ConversationConfig{
		AgentIO: config.Process,
		GetTime: func() time.Time {
			return time.Now()
		},
		SnapshotInterval:      snapshotInterval,
		ScreenStabilityLength: 2 * time.Second,
		FormatMessage:         formatMessage,
	})
	emitter := NewEventEmitter(1024)
	s := &Server{
		router:       router,
		api:          api,
		port:         config.Port,
		conversation: conversation,
		logger:       logctx.From(ctx),
		agentio:      config.Process,
		agentType:    config.AgentType,
		emitter:      emitter,
		splitter:     mf.NewServiceSplitter(),
		lastSentChat: "",
		// chatBasePath: strings.TrimSuffix(config.ChatBasePath, "/"), // removed - no WebUI
	}

	// Register API routes
	s.registerRoutes()

	return s
}

// Handler returns the underlying chi.Router for testing purposes.
func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) StartSnapshotLoop(ctx context.Context) {
	s.conversation.StartSnapshotLoop(ctx)
	go func() {
		for {
			s.emitter.UpdateStatusAndEmitChanges(s.conversation.Status())
			s.emitter.UpdateMessagesAndEmitChanges(s.conversation.Messages())
			s.emitter.UpdateScreenAndEmitChanges(s.conversation.Screen())
			time.Sleep(snapshotInterval)
		}
	}()
}

// registerRoutes sets up all API endpoints
func (s *Server) registerRoutes() {
	// GET /status endpoint
	huma.Get(s.api, "/status", s.getStatus, func(o *huma.Operation) {
		o.Description = "Returns the current status of the agent."
	})

	// GET /messages endpoint
	huma.Get(s.api, "/messages", s.getMessages, func(o *huma.Operation) {
		o.Description = "Returns a list of messages representing the conversation history with the agent."
	})

	// GET /service-info endpoint
	huma.Get(s.api, "/service-info", s.getServiceInfo, func(o *huma.Operation) {
		o.Description = "Returns service information lines (model, permissions, cooking status, etc.)"
	})

	// GET /messages/delta endpoint
	huma.Get(s.api, "/messages/delta", s.getMessagesDelta, func(o *huma.Operation) {
		o.Description = "Returns only new chat messages since last request (delta)"
	})

	// POST /message endpoint
	huma.Post(s.api, "/message", s.createMessage, func(o *huma.Operation) {
		o.Description = "Send a message to the agent. For messages of type 'user', the agent's status must be 'stable' for the operation to complete successfully. Otherwise, this endpoint will return an error."
	})

	// GET /events endpoint
	sse.Register(s.api, huma.Operation{
		OperationID: "subscribeEvents",
		Method:      http.MethodGet,
		Path:        "/events",
		Summary:     "Subscribe to events",
		Description: "The events are sent as Server-Sent Events (SSE). Initially, the endpoint returns a list of events needed to reconstruct the current state of the conversation and the agent's status. After that, it only returns events that have occurred since the last event was sent.\n\nNote: When an agent is running, the last message in the conversation history is updated frequently, and the endpoint sends a new message update event each time.",
	}, map[string]any{
		// Mapping of event type name to Go struct for that event.
		"message_update": MessageUpdateBody{},
		"status_change":  StatusChangeBody{},
	}, s.subscribeEvents)

	sse.Register(s.api, huma.Operation{
		OperationID: "subscribeScreen",
		Method:      http.MethodGet,
		Path:        "/internal/screen",
		Summary:     "Subscribe to screen",
		Hidden:      true,
	}, map[string]any{
		"screen": ScreenUpdateBody{},
	}, s.subscribeScreen)

	// WebUI removed - no redirect to chat
	// s.router.Handle("/", http.HandlerFunc(s.redirectToChat))
	// s.registerStaticFileRoutes()
}

// getStatus handles GET /status
func (s *Server) getStatus(ctx context.Context, input *struct{}) (*StatusResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := s.conversation.Status()
	agentStatus := convertStatus(status)

	resp := &StatusResponse{}
	resp.Body.Status = agentStatus

	return resp, nil
}

// getMessages handles GET /messages
func (s *Server) getMessages(ctx context.Context, input *struct{}) (*MessagesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp := &MessagesResponse{}
	resp.Body.Messages = make([]Message, len(s.conversation.Messages()))
	for i, msg := range s.conversation.Messages() {
		resp.Body.Messages[i] = Message{
			Id:      msg.Id,
			Role:    msg.Role,
			Content: msg.Message,
			Time:    msg.Time,
		}
	}

	return resp, nil
}

// ServiceInfoResponse is the response for GET /service-info
type ServiceInfoResponse struct {
	Body struct {
		ServiceInfo string `json:"service_info" doc:"Service information lines (model, permissions, cooking status, etc.)"`
	}
}

// getServiceInfo handles GET /service-info
func (s *Server) getServiceInfo(ctx context.Context, input *struct{}) (*ServiceInfoResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get current terminal screen
	screen := s.conversation.Screen()
	
	// Split into chat and service lines
	_, serviceLines := s.splitter.SplitOutput(screen)
	
	// Format service lines
	serviceInfo := s.splitter.GetServiceInfo(serviceLines)
	
	resp := &ServiceInfoResponse{}
	resp.Body.ServiceInfo = serviceInfo
	
	return resp, nil
}

// MessagesDeltaResponse is the response for GET /messages/delta
type MessagesDeltaResponse struct {
	Body struct {
		Delta string `json:"delta" doc:"New chat content since last request"`
	}
}

// getMessagesDelta handles GET /messages/delta
func (s *Server) getMessagesDelta(ctx context.Context, input *struct{}) (*MessagesDeltaResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get current terminal screen
	screen := s.conversation.Screen()
	
	// Split into chat and service lines (ignore service lines for delta)
	chatContent, _ := s.splitter.SplitOutput(screen)
	
	// Calculate delta
	delta := ""
	if len(chatContent) > len(s.lastSentChat) {
		// We have new content
		delta = chatContent[len(s.lastSentChat):]
	}
	
	// Update last sent state
	s.lastSentChat = chatContent
	
	resp := &MessagesDeltaResponse{}
	resp.Body.Delta = delta
	
	return resp, nil
}

// createMessage handles POST /message
func (s *Server) createMessage(ctx context.Context, input *MessageRequest) (*MessageResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch input.Body.Type {
	case MessageTypeUser:
		if err := s.conversation.SendMessage(FormatMessage(s.agentType, input.Body.Content)...); err != nil {
			return nil, xerrors.Errorf("failed to send message: %w", err)
		}
	case MessageTypeRaw:
		if _, err := s.agentio.Write([]byte(input.Body.Content)); err != nil {
			return nil, xerrors.Errorf("failed to send message: %w", err)
		}
	}

	resp := &MessageResponse{}
	resp.Body.Ok = true

	return resp, nil
}

// subscribeEvents is an SSE endpoint that sends events to the client
func (s *Server) subscribeEvents(ctx context.Context, input *struct{}, send sse.Sender) {
	subscriberId, ch, stateEvents := s.emitter.Subscribe()
	defer s.emitter.Unsubscribe(subscriberId)
	s.logger.Info("New subscriber", "subscriberId", subscriberId)
	for _, event := range stateEvents {
		if event.Type == EventTypeScreenUpdate {
			continue
		}
		if err := send.Data(event.Payload); err != nil {
			s.logger.Error("Failed to send event", "subscriberId", subscriberId, "error", err)
			return
		}
	}
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				s.logger.Info("Channel closed", "subscriberId", subscriberId)
				return
			}
			if event.Type == EventTypeScreenUpdate {
				continue
			}
			if err := send.Data(event.Payload); err != nil {
				s.logger.Error("Failed to send event", "subscriberId", subscriberId, "error", err)
				return
			}
		case <-ctx.Done():
			s.logger.Info("Context done", "subscriberId", subscriberId)
			return
		}
	}
}

func (s *Server) subscribeScreen(ctx context.Context, input *struct{}, send sse.Sender) {
	subscriberId, ch, stateEvents := s.emitter.Subscribe()
	defer s.emitter.Unsubscribe(subscriberId)
	s.logger.Info("New screen subscriber", "subscriberId", subscriberId)
	for _, event := range stateEvents {
		if event.Type != EventTypeScreenUpdate {
			continue
		}
		if err := send.Data(event.Payload); err != nil {
			s.logger.Error("Failed to send screen event", "subscriberId", subscriberId, "error", err)
			return
		}
	}
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				s.logger.Info("Screen channel closed", "subscriberId", subscriberId)
				return
			}
			if event.Type != EventTypeScreenUpdate {
				continue
			}
			if err := send.Data(event.Payload); err != nil {
				s.logger.Error("Failed to send screen event", "subscriberId", subscriberId, "error", err)
				return
			}
		case <-ctx.Done():
			s.logger.Info("Screen context done", "subscriberId", subscriberId)
			return
		}
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	s.srv = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	return s.srv.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}
	return nil
}

// WebUI removed - these functions are no longer needed
/*
func (s *Server) registerStaticFileRoutes() {
	chatHandler := FileServerWithIndexFallback(s.chatBasePath)
	s.router.Handle("/chat", http.StripPrefix("/chat", chatHandler))
	s.router.Handle("/chat/*", http.StripPrefix("/chat", chatHandler))
}

func (s *Server) redirectToChat(w http.ResponseWriter, r *http.Request) {
	rdir, err := url.JoinPath(s.chatBasePath, "embed")
	if err != nil {
		s.logger.Error("Failed to construct redirect URL", "error", err)
		http.Error(w, "Failed to redirect", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, rdir, http.StatusTemporaryRedirect)
}
*/
