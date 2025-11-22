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
	"github.com/fulgidus/terminalpub/internal/config"
	"github.com/fulgidus/terminalpub/internal/db"
	"github.com/fulgidus/terminalpub/internal/handlers"
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

	// Placeholder routes for future OAuth
	r.Get("/device", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OAuth Device Flow - Coming in Phase 2"))
	})
	r.Get("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OAuth Callback - Coming in Phase 2"))
	})

	// Placeholder routes for future ActivityPub
	r.Get("/.well-known/webfinger", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("WebFinger - Coming in Phase 3"))
	})
	r.Get("/users/{username}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ActivityPub Actor - Coming in Phase 3"))
	})

	addr := fmt.Sprintf(":%s", cfg.Server.HTTPPort)
	return &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// TUI code remains the same
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	m := model{
		username: s.User(),
		screen:   screenWelcome,
	}
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

type screenType int

const (
	screenWelcome screenType = iota
	screenAnonymous
	screenLogin
)

type model struct {
	username string
	screen   screenType
	message  string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.screen {
		case screenWelcome:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "l", "L":
				m.screen = screenLogin
				m.message = "Login feature coming in Phase 2!"
			case "a", "A":
				m.screen = screenAnonymous
				m.message = "Anonymous mode activated!"
			}
		case screenAnonymous:
			switch msg.String() {
			case "q", "ctrl+c", "esc":
				return m, tea.Quit
			case "b", "B":
				m.screen = screenWelcome
				m.message = ""
			}
		case screenLogin:
			switch msg.String() {
			case "q", "ctrl+c", "esc", "b", "B":
				m.screen = screenWelcome
				m.message = ""
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case screenWelcome:
		return m.renderWelcome()
	case screenAnonymous:
		return m.renderAnonymous()
	case screenLogin:
		return m.renderLogin()
	default:
		return "Unknown screen"
	}
}

func (m model) renderWelcome() string {
	return fmt.Sprintf(`
╔════════════════════════════════════════════╗
║        Welcome to terminalpub!             ║
║        ActivityPub for terminals           ║
╠════════════════════════════════════════════╣
║                                            ║
║  Connected as: %-27s ║
║                                            ║
║  Press a key to continue:                  ║
║                                            ║
║  [L] Login with Mastodon (Coming soon)     ║
║  [A] Continue anonymously                  ║
║  [Q] Quit                                  ║
║                                            ║
╚════════════════════════════════════════════╝

%s
`, m.username, m.message)
}

func (m model) renderAnonymous() string {
	return fmt.Sprintf(`
╔════════════════════════════════════════════╗
║           Anonymous Mode                   ║
╠════════════════════════════════════════════╣
║                                            ║
║  %s                                        ║
║                                            ║
║  You're browsing as: anonymous             ║
║                                            ║
║  Available features:                       ║
║  • View public feed (Coming soon)          ║
║  • Chat roulette (Coming soon)             ║
║  • Browse hashtags (Coming soon)           ║
║                                            ║
║  Commands:                                 ║
║  [B] Back to menu                          ║
║  [Q] Quit                                  ║
║                                            ║
╚════════════════════════════════════════════╝

🚧 This is a work in progress!
Phase 1: Infrastructure ✅
Phase 2: Authentication (Next)
Phase 3: ActivityPub Integration
`, m.message)
}

func (m model) renderLogin() string {
	return fmt.Sprintf(`
╔════════════════════════════════════════════╗
║        Login with Mastodon                 ║
╠════════════════════════════════════════════╣
║                                            ║
║  %s                                        ║
║                                            ║
║  OAuth Device Flow authentication will     ║
║  be implemented in Phase 2!                ║
║                                            ║
║  This will allow you to:                   ║
║  • Login with your Mastodon account        ║
║  • Access your federated feed              ║
║  • Post and interact with the fediverse    ║
║  • Import your following/followers         ║
║                                            ║
║  Stay tuned!                               ║
║                                            ║
║  [B] Back to menu                          ║
║  [Q] Quit                                  ║
║                                            ║
╚════════════════════════════════════════════╝
`, m.message)
}
