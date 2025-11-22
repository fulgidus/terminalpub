package ui

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fulgidus/terminalpub/internal/services"
)

// NotificationsModel represents the notifications view state
type NotificationsModel struct {
	ctx             context.Context
	userID          int
	mastodonService *services.MastodonService
	notifications   []services.MastodonNotification
	selectedIndex   int
	scrollOffset    int
	loading         bool
	loadingMore     bool
	hasMore         bool
	statusMessage   string
	width           int
	height          int
	err             error
}

// notificationsLoadedMsg is sent when notifications are fetched
type notificationsLoadedMsg struct {
	notifications []services.MastodonNotification
	isLoadMore    bool
	err           error
}

// dismissNotificationMsg is sent when a notification is dismissed
type dismissNotificationMsg struct {
	notificationID string
	err            error
}

// NewNotificationsModel creates a new notifications view model
func NewNotificationsModel(ctx context.Context, userID int, mastodonService *services.MastodonService) NotificationsModel {
	return NotificationsModel{
		ctx:             ctx,
		userID:          userID,
		mastodonService: mastodonService,
		loading:         true,
		statusMessage:   "Loading notifications...",
		hasMore:         true,
	}
}

// Init initializes the notifications model and fetches notifications
func (m NotificationsModel) Init() tea.Cmd {
	return m.fetchNotificationsCmd(false)
}

// Update handles messages for the notifications view
func (m NotificationsModel) Update(msg tea.Msg) (NotificationsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case notificationsLoadedMsg:
		m.loading = false
		m.loadingMore = false
		if msg.err != nil {
			m.err = msg.err
			m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
			return m, nil
		}

		if msg.isLoadMore {
			// Append new notifications
			m.notifications = append(m.notifications, msg.notifications...)
			if len(msg.notifications) < 20 {
				m.hasMore = false
			}
			m.statusMessage = fmt.Sprintf("Loaded %d more notifications", len(msg.notifications))
		} else {
			// Replace with new notifications
			m.notifications = msg.notifications
			m.selectedIndex = 0
			m.scrollOffset = 0
			m.hasMore = len(msg.notifications) >= 20
			m.statusMessage = ""
		}
		return m, nil

	case dismissNotificationMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error dismissing: %v", msg.err)
		} else {
			// Remove notification from list
			for i, notif := range m.notifications {
				if notif.ID == msg.notificationID {
					m.notifications = append(m.notifications[:i], m.notifications[i+1:]...)
					// Adjust selection
					if m.selectedIndex >= len(m.notifications) && m.selectedIndex > 0 {
						m.selectedIndex--
					}
					break
				}
			}
			m.statusMessage = "Notification dismissed"
		}
		return m, nil
	}

	return m, nil
}

// View renders the notifications view
func (m NotificationsModel) View() string {
	if m.loading {
		return m.statusMessage
	}

	if m.err != nil {
		return fmt.Sprintf("Error loading notifications: %v\n\nPress ESC to go back", m.err)
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	unreadCount := len(m.notifications)
	title := "Notifications"
	if unreadCount > 0 {
		title = fmt.Sprintf("Notifications (%d)", unreadCount)
	}
	b.WriteString(titleStyle.Render(title) + "\n\n")

	if len(m.notifications) == 0 {
		b.WriteString("No notifications\n\n")
		keyColor := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
		b.WriteString(keyColor.Render("[ESC]") + " Back\n")
		return b.String()
	}

	// Calculate available height for notifications
	headerLines := 3
	footerLines := 2
	availableHeight := m.height - headerLines - footerLines
	if availableHeight < 3 {
		availableHeight = 3
	}

	// Each notification takes ~4 lines
	notifsPerScreen := availableHeight / 4
	if notifsPerScreen < 1 {
		notifsPerScreen = 1
	}

	// Ensure scroll offset keeps selected item visible
	if m.selectedIndex < m.scrollOffset {
		m.scrollOffset = m.selectedIndex
	}
	if m.selectedIndex >= m.scrollOffset+notifsPerScreen {
		m.scrollOffset = m.selectedIndex - notifsPerScreen + 1
	}

	endIndex := m.scrollOffset + notifsPerScreen
	if endIndex > len(m.notifications) {
		endIndex = len(m.notifications)
	}

	// Render notifications
	for i := m.scrollOffset; i < endIndex; i++ {
		notif := m.notifications[i]
		b.WriteString(m.renderNotification(notif, i == m.selectedIndex))
		if i < endIndex-1 {
			b.WriteString("\n")
		}
	}

	// Controls
	b.WriteString("\n")
	keyColor := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	subtleColor := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	loadMoreText := ""
	if m.hasMore && !m.loadingMore {
		loadMoreText = "  " + subtleColor.Render("(scroll to load more)")
	} else if !m.hasMore {
		loadMoreText = "  " + subtleColor.Render("(all loaded)")
	}

	controls := fmt.Sprintf("  %s Navigate  %s View  %s Dismiss  %s Clear All  %s Back%s",
		subtleColor.Render("↑/↓"),
		keyColor.Render("[Enter]"),
		keyColor.Render("[D]"),
		keyColor.Render("[C]"),
		keyColor.Render("[ESC]"),
		loadMoreText)
	b.WriteString(controls)

	if m.statusMessage != "" {
		statusColor := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		if strings.Contains(m.statusMessage, "Error") {
			statusColor = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		}
		b.WriteString("\n  " + statusColor.Render(m.statusMessage))
	}

	return b.String()
}

// renderNotification renders a single notification
func (m NotificationsModel) renderNotification(notif services.MastodonNotification, selected bool) string {
	var b strings.Builder

	// Colors
	grayColor := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selectionColor := lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	greenColor := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	orangeColor := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	cyanColor := lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

	// Selection indicator
	selector := "  "
	if selected {
		selector = selectionColor.Render("► ")
	}

	// Notification icon and text based on type
	var icon, action string
	var iconColor lipgloss.Style

	switch notif.Type {
	case services.NotificationMention:
		icon = "Reply:"
		iconColor = cyanColor
		action = "mentioned you"
	case services.NotificationReblog:
		icon = "Boost:"
		iconColor = greenColor
		action = "boosted your post"
	case services.NotificationFavourite:
		icon = "Like:"
		iconColor = orangeColor
		action = "liked your post"
	case services.NotificationFollow:
		icon = "Follow:"
		iconColor = greenColor
		action = "started following you"
	case services.NotificationPoll:
		icon = "Poll:"
		iconColor = cyanColor
		action = "poll ended"
	case services.NotificationFollowRequest:
		icon = "Request:"
		iconColor = orangeColor
		action = "requested to follow you"
	default:
		icon = "Update:"
		iconColor = grayColor
		action = "notification"
	}

	// First line: icon + account + action
	displayName := notif.Account.DisplayName
	if displayName == "" {
		displayName = notif.Account.Username
	}

	line1 := fmt.Sprintf("%s %s %s",
		iconColor.Render(icon),
		cyanColor.Render(displayName),
		action)
	b.WriteString(selector + line1 + "\n")

	// Second line: content (if status exists)
	if notif.Status != nil {
		content := stripHTMLNotif(notif.Status.Content)
		if len(content) > 100 {
			content = content[:97] + "..."
		}
		b.WriteString(selector + "  " + content + "\n")
	}

	// Third line: timestamp
	timeAgo := formatTimeAgo(notif.CreatedAt)
	b.WriteString(selector + "  " + grayColor.Render(timeAgo) + "\n")

	// Separator
	b.WriteString(selector + grayColor.Render("────────────────────────────"))

	return b.String()
}

// fetchNotificationsCmd fetches notifications
func (m NotificationsModel) fetchNotificationsCmd(isLoadMore bool) tea.Cmd {
	return func() tea.Msg {
		maxID := ""
		if isLoadMore && len(m.notifications) > 0 {
			maxID = m.notifications[len(m.notifications)-1].ID
		}

		notifications, err := m.mastodonService.GetNotifications(m.ctx, m.userID, 20, maxID)
		if err != nil {
			return notificationsLoadedMsg{err: err}
		}

		return notificationsLoadedMsg{
			notifications: notifications,
			isLoadMore:    isLoadMore,
		}
	}
}

// GetSelectedNotification returns the currently selected notification
func (m NotificationsModel) GetSelectedNotification() *services.MastodonNotification {
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.notifications) {
		return &m.notifications[m.selectedIndex]
	}
	return nil
}

// stripHTMLNotif removes HTML tags from notification content
func stripHTMLNotif(content string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	stripped := re.ReplaceAllString(content, "")

	// Decode HTML entities
	stripped = html.UnescapeString(stripped)

	// Replace multiple spaces/newlines with single space
	stripped = strings.Join(strings.Fields(stripped), " ")

	return stripped
}

// formatTimeAgo formats a time as "X minutes/hours/days ago"
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
