package web

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"icscommemulator/pkg/logger"
	"icscommemulator/pkg/runner"
	"icscommemulator/pkg/scenario"
	"icscommemulator/pkg/service"
)

// Server representa el servidor HTTP con sus dependencias
type Server struct {
	host       string
	port       int
	httpServer *http.Server
	handlers   *Handlers
	templates  *template.Template
	staticPath string
}

// NewServer crea una nueva instancia del servidor con dependency injection
func NewServer(host string, port int) (*Server, error) {
	server := &Server{
		host:       host,
		port:       port,
		staticPath: "web/static",
	}

	// Cargar templates
	if err := server.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Crear dependencias
	scenarioStorage := scenario.NewStorage()
	runnerSvc := runner.NewService()

	// Crear servicio de escenarios (unifica toda la lógica)
	scenarioSvc := service.NewScenarioService("/tmp/ICSCommEmulator", runnerSvc)

	// Handlers ahora solo reciben storage y servicio
	server.handlers = NewHandlers(scenarioStorage, scenarioSvc)

	// Configurar rutas
	mux := http.NewServeMux()
	server.setupRoutes(mux)

	// Configurar servidor HTTP
	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Handler:      server.loggingMiddleware(server.corsMiddleware(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server, nil
}

func (s *Server) loadTemplates() error {
	funcMap := template.FuncMap{
		"toJSON": toJSONFunc,
	}

	templates, err := template.New("").Funcs(funcMap).ParseGlob("web/templates/*.html")
	if err != nil {
		logger.Warning("Failed to load templates: %v", err)
		s.templates = template.New("empty").Funcs(funcMap)
		return nil
	}

	s.templates = templates
	logger.Debug("Templates loaded successfully")
	return nil
}

func (s *Server) setupRoutes(mux *http.ServeMux) {
	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(s.staticPath))))

	// API routes
	mux.HandleFunc("/api/networks/", s.handlers.HandleNetworks)
	mux.HandleFunc("/api/run", s.handlers.HandleRun)

	// Web pages
	mux.HandleFunc("/", s.handleHome)
	mux.HandleFunc("/index.html", s.handleHome)
	mux.HandleFunc("/networks/", s.handleNetworkPage)

	logger.Debug("HTTP routes configured")
}

func (s *Server) Start() error {
	logger.Info("Server listening on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Handlers HTTP para páginas web
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}

	if err := s.templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		logger.Error("Template execution error: %v", err)
	}
}

func (s *Server) handleNetworkPage(w http.ResponseWriter, r *http.Request) {
	path := extractPathSegment(r.URL.Path, "/networks/")
	if path == "" {
		http.Error(w, "Network ID required", http.StatusBadRequest)
		return
	}

	networkData, err := s.handlers.scenarioStorage.GetCytoscapeScenarioAsMap(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	data := struct {
		NetworkData map[string]interface{}
	}{
		NetworkData: networkData,
	}

	if err := s.templates.ExecuteTemplate(w, "network.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		logger.Error("Template execution error: %v", err)
	}
}
