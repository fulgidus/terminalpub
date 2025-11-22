package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

const (
	host = "0.0.0.0"
	port = "22"
)

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%s", host, port)),
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
	log.Printf("Starting SSH server on %s:%s", host, port)
	go func() {
		if err = s.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()

	<-done
	log.Println("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil {
		log.Fatalln(err)
	}
}

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
