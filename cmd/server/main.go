package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/fulgidus/terminalpub/internal/auth"
	"github.com/fulgidus/terminalpub/internal/config"
	"github.com/fulgidus/terminalpub/internal/db"
	"github.com/fulgidus/terminalpub/internal/handlers"
	"github.com/fulgidus/terminalpub/internal/ui"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Load configuration
	cfg := config.LoadOrDefault("config/config.yaml")
	log.Printf("Loaded configuration for domain: %s", cfg.Server.Domain)

	// Connect to databases (optional for now, can fail gracefully)
	var database *db.DB
	var err error
	database, err = db.Connect(cfg)
	if err != nil {
		log.Printf("Warning: Failed to connect to databases: %v", err)
		log.Printf("SSH server will run without database support")
	} else {
		defer database.Close()
		log.Println("Connected to PostgreSQL and Redis")

		// Initialize app context for TUI
		initAppContext(cfg, database)
	}

	// Setup HTTP server
	httpServer := setupHTTPServer(cfg, database)
	go func() {
		addr := fmt.Sprintf(":%s", cfg.Server.HTTPPort)
		log.Printf("Starting HTTP server on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Setup SSH server
	sshServer, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("0.0.0.0:%s", cfg.Server.SSHPort)),
		wish.WithHostKeyPath(".ssh/term_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Fatalln(err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Starting SSH server on 0.0.0.0:%s", cfg.Server.SSHPort)
	go func() {
		if err = sshServer.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()

	<-done
	log.Println("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Shutdown SSH server
	if err := sshServer.Shutdown(ctx); err != nil {
		log.Printf("SSH server shutdown error: %v", err)
	}

	log.Println("Servers stopped")
}

func setupHTTPServer(cfg *config.Config, database *db.DB) *http.Server {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>terminalpub</title>
    <style>
        body { 
            font-family: monospace; 
            max-width: 800px; 
            margin: 50px auto; 
            padding: 20px;
            background: #1a1a1a;
            color: #00ff00;
        }
        h1 { color: #00ff00; }
        pre { background: #000; padding: 20px; border-radius: 5px; }
        a { color: #00ffff; }
    </style>
</head>
<body>
    <h1>terminalpub</h1>
    <p>ActivityPub for your terminal</p>
    <h2>Connect via SSH:</h2>
    <pre>ssh %s</pre>
    <p><a href="/health">Health Check</a></p>
    <p><a href="https://github.com/fulgidus/terminalpub">GitHub</a></p>
</body>
</html>`, cfg.Server.Domain)
	})

	// Health check endpoint
	healthHandler := handlers.NewHealthHandler(database)
	r.Handle("/health", healthHandler)

	// OAuth Device Flow routes
	if database != nil {
		oauthHandler := handlers.NewOAuthHandler(database.Postgres, database.Redis, cfg)
		r.Handle("/device", oauthHandler)
		r.HandleFunc("/oauth/callback", oauthHandler.HandleCallback)
	} else {
		r.Get("/device", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("OAuth Device Flow - Database not available"))
		})
		r.Get("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("OAuth Callback - Database not available"))
		})
	}

	// ActivityPub routes
	if database != nil {
		apHandler := handlers.NewActivityPubHandler(database.Postgres, cfg)
		r.Get("/.well-known/webfinger", apHandler.WebFinger)
		r.Get("/users/{username}", apHandler.Actor)
		r.Post("/users/{username}/inbox", apHandler.Inbox)
		r.Get("/users/{username}/inbox", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Inbox is write-only", http.StatusMethodNotAllowed)
		})
		r.Get("/users/{username}/outbox", apHandler.Outbox)
		r.Get("/users/{username}/followers", apHandler.Followers)
		r.Get("/users/{username}/following", apHandler.Following)
	} else {
		r.Get("/.well-known/webfinger", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("WebFinger - Database not available"))
		})
		r.Get("/users/{username}", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ActivityPub Actor - Database not available"))
		})
	}

	addr := fmt.Sprintf(":%s", cfg.Server.HTTPPort)
	return &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// Global app context for TUI
var appCtx *ui.AppContext

// initAppContext initializes the app context
func initAppContext(cfg *config.Config, database *db.DB) {
	if database == nil {
		return
	}

	deviceFlowService := auth.NewDeviceFlowService(
		database.Postgres,
		fmt.Sprintf("http://%s/device", cfg.Server.Domain),
	)
	sshKeyService := auth.NewSSHKeyService(database.Postgres)
	sessionManager := auth.NewSessionManager(database.Postgres, database.Redis)

	appCtx = &ui.AppContext{
		DB:                database.Postgres,
		Redis:             database.Redis,
		Config:            cfg,
		DeviceFlowService: deviceFlowService,
		SSHKeyService:     sshKeyService,
		SessionManager:    sessionManager,
	}
}

// teaHandler creates a new TUI model for each SSH session
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	if appCtx == nil {
		// Fallback if no database connection
		return ui.NewModel(nil, s), []tea.ProgramOption{tea.WithAltScreen()}
	}

	return ui.NewModel(appCtx, s), []tea.ProgramOption{tea.WithAltScreen()}
}
