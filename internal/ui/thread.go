package ui

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fulgidus/terminalpub/internal/services"
)

// ThreadModel represents the conversation thread view state
type ThreadModel struct {
	ctx             context.Context
	userID          int
	mastodonService *services.MastodonService
	rootStatus      services.MastodonStatus
	ancestors       []services.MastodonStatus
	descendants     []services.MastodonStatus
	flattenedThread []threadItem
	selectedIndex   int
	scrollOffset    int
	loading         bool
	statusMessage   string
	width           int
	height          int
	err             error
}

// threadItem represents a flattened thread item with depth information
type threadItem struct {
	status services.MastodonStatus
	depth  int
	isRoot bool
}

// threadLoadedMsg is sent when the thread context is fetched
type threadLoadedMsg struct {
	rootStatus  services.MastodonStatus
	ancestors   []services.MastodonStatus
	descendants []services.MastodonStatus
	err         error
}

// NewThreadModel creates a new thread view model
func NewThreadModel(ctx context.Context, userID int, mastodonService *services.MastodonService, rootStatus services.MastodonStatus) ThreadModel {
	return ThreadModel{
		ctx:             ctx,
		userID:          userID,
		mastodonService: mastodonService,
		rootStatus:      rootStatus,
		loading:         true,
		statusMessage:   "Loading thread...",
	}
}

// Init initializes the thread model and fetches the thread context
func (m ThreadModel) Init() tea.Cmd {
	return m.fetchThreadCmd()
}

// Update handles messages for the thread view
func (m ThreadModel) Update(msg tea.Msg) (ThreadModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case threadLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
			return m, nil
		}

		m.rootStatus = msg.rootStatus
		m.ancestors = msg.ancestors
		m.descendants = msg.descendants
		m.flattenedThread = m.buildFlattenedThread()
		m.statusMessage = ""

		// Select the root status by default
		for i, item := range m.flattenedThread {
			if item.isRoot {
				m.selectedIndex = i
				break
			}
		}

		return m, nil
	}

	return m, nil
}

// View renders the thread view
func (m ThreadModel) View() string {
	if m.loading {
		return m.statusMessage
	}

	if m.err != nil {
		return fmt.Sprintf("Error loading thread: %v\n\nPress ESC to go back", m.err)
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	b.WriteString(titleStyle.Render("Conversation Thread") + "\n\n")

	// Calculate available height for content
	headerLines := 3 // title + controls
	footerLines := 1
	availableHeight := m.height - headerLines - footerLines
	if availableHeight < 5 {
		availableHeight = 5
	}

	// Ensure scroll offset keeps selected item visible
	if m.selectedIndex < m.scrollOffset {
		m.scrollOffset = m.selectedIndex
	}
	if m.selectedIndex >= m.scrollOffset+availableHeight {
		m.scrollOffset = m.selectedIndex - availableHeight + 1
	}

	// Render visible thread items
	endIndex := m.scrollOffset + availableHeight
	if endIndex > len(m.flattenedThread) {
		endIndex = len(m.flattenedThread)
	}

	for i := m.scrollOffset; i < endIndex; i++ {
		item := m.flattenedThread[i]
		b.WriteString(m.renderThreadItem(item, i == m.selectedIndex))
		if i < endIndex-1 {
			b.WriteString("\n")
		}
	}

	// Controls
	b.WriteString("\n")
	keyColor := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	subtleColor := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	controls := fmt.Sprintf("  %s Navigate  %s Reply  %s Back  %s View in Browser",
		subtleColor.Render("↑/↓"),
		keyColor.Render("[R]"),
		keyColor.Render("[ESC]"),
		keyColor.Render("[O]"))
	b.WriteString(controls)

	return b.String()
}

// renderThreadItem renders a single thread item with indentation
func (m ThreadModel) renderThreadItem(item threadItem, selected bool) string {
	var b strings.Builder

	// Colors
	cyanColor := lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
	grayColor := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	greenColor := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	selectionColor := lipgloss.NewStyle().Foreground(lipgloss.Color("12"))

	// Indentation based on depth
	indent := strings.Repeat("  ", item.depth)
	if item.depth > 0 {
		indent += "└─▶ "
	}

	// Selection indicator
	selector := "  "
	if selected {
		selector = selectionColor.Render("► ")
	}

	// Display name and handle
	displayName := item.status.Account.DisplayName
	if displayName == "" {
		displayName = item.status.Account.Username
	}
	author := cyanColor.Render(displayName) + " " + grayColor.Render("@"+item.status.Account.Acct)

	// Mark if this is the root post
	rootMarker := ""
	if item.isRoot {
		rootMarker = " " + greenColor.Render("[Original Post]")
	}

	b.WriteString(selector + indent + author + rootMarker + "\n")

	// Content (strip HTML and trim)
	content := stripHTMLFromContent(item.status.Content)
	content = strings.TrimSpace(content)
	if len(content) > 200 {
		content = content[:197] + "..."
	}
	b.WriteString(selector + indent + content + "\n")

	// Stats and interactions
	stats := fmt.Sprintf("Likes: %d  Boosts: %d  Replies: %d",
		item.status.FavouritesCount,
		item.status.ReblogsCount,
		item.status.RepliesCount)

	// Add interaction markers
	if item.status.Favourited || item.status.Reblogged {
		stats += " " + greenColor.Render("[*]")
	}

	b.WriteString(selector + indent + grayColor.Render(stats))

	return b.String()
}

// buildFlattenedThread creates a flat list of thread items with depth information
func (m ThreadModel) buildFlattenedThread() []threadItem {
	var items []threadItem

	// Add ancestors (in chronological order)
	for i, status := range m.ancestors {
		items = append(items, threadItem{
			status: status,
			depth:  i,
			isRoot: false,
		})
	}

	// Add root status
	rootDepth := len(m.ancestors)
	items = append(items, threadItem{
		status: m.rootStatus,
		depth:  rootDepth,
		isRoot: true,
	})

	// Add descendants (replies)
	// Build a tree structure and flatten it with proper depth
	descendantsTree := m.buildDescendantsTree(m.rootStatus.ID, m.descendants)
	flatDescendants := m.flattenDescendantsTree(descendantsTree, rootDepth+1)
	items = append(items, flatDescendants...)

	return items
}

// descendantNode represents a node in the descendants tree
type descendantNode struct {
	status   services.MastodonStatus
	children []*descendantNode
}

// buildDescendantsTree builds a tree structure from descendants
func (m ThreadModel) buildDescendantsTree(parentID string, descendants []services.MastodonStatus) []*descendantNode {
	var roots []*descendantNode
	nodeMap := make(map[string]*descendantNode)

	// Create nodes
	for _, status := range descendants {
		node := &descendantNode{
			status:   status,
			children: []*descendantNode{},
		}
		nodeMap[status.ID] = node
	}

	// Build tree
	for _, status := range descendants {
		node := nodeMap[status.ID]
		if status.InReplyToID != nil && *status.InReplyToID == parentID {
			// Direct reply to parent
			roots = append(roots, node)
		} else if status.InReplyToID != nil {
			// Reply to another descendant
			if parentNode, exists := nodeMap[*status.InReplyToID]; exists {
				parentNode.children = append(parentNode.children, node)
			}
		}
	}

	return roots
}

// flattenDescendantsTree flattens the descendants tree into a list with depth
func (m ThreadModel) flattenDescendantsTree(roots []*descendantNode, startDepth int) []threadItem {
	var items []threadItem

	var flatten func(node *descendantNode, depth int)
	flatten = func(node *descendantNode, depth int) {
		// Cap depth at a reasonable level for readability
		displayDepth := depth
		if displayDepth > startDepth+5 {
			displayDepth = startDepth + 5
		}

		items = append(items, threadItem{
			status: node.status,
			depth:  displayDepth,
			isRoot: false,
		})

		for _, child := range node.children {
			flatten(child, depth+1)
		}
	}

	for _, root := range roots {
		flatten(root, startDepth)
	}

	return items
}

// fetchThreadCmd fetches the thread context
func (m ThreadModel) fetchThreadCmd() tea.Cmd {
	return func() tea.Msg {
		context, err := m.mastodonService.GetStatusContext(m.ctx, m.userID, m.rootStatus.ID)
		if err != nil {
			return threadLoadedMsg{err: err}
		}

		return threadLoadedMsg{
			rootStatus:  m.rootStatus,
			ancestors:   context.Ancestors,
			descendants: context.Descendants,
		}
	}
}

// GetSelectedStatus returns the currently selected status
func (m ThreadModel) GetSelectedStatus() *services.MastodonStatus {
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.flattenedThread) {
		return &m.flattenedThread[m.selectedIndex].status
	}
	return nil
}

// stripHTMLFromContent removes HTML tags from content (specific to thread view)
func stripHTMLFromContent(content string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	stripped := re.ReplaceAllString(content, "")

	// Decode HTML entities
	stripped = html.UnescapeString(stripped)

	// Replace multiple spaces/newlines with single space
	stripped = strings.Join(strings.Fields(stripped), " ")

	return stripped
}
