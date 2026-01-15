package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	sessions []*Session
	store    *Store
	agent    *Manager
	cursor   int
	err      error
	attach   *Session
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func New(store *Store, agent *Manager) Model {
	return Model{
		store: store,
		agent: agent,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadSessions(), tick())
}

func (m Model) loadSessions() tea.Cmd {
	return func() tea.Msg {
		sessions, err := m.store.List()
		if err != nil {
			return err
		}
		return sessions
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.sessions) > 0 {
				m.attach = m.sessions[m.cursor]
				return m, tea.Quit
			}
		case "d":
			if len(m.sessions) > 0 {
				sess := m.sessions[m.cursor]
				m.store.Delete(sess.ID)
				return m, m.loadSessions()
			}
		case "r":
			return m, m.loadSessions()
		}

	case []*Session:
		m.sessions = msg
		if m.cursor >= len(m.sessions) {
			m.cursor = len(m.sessions) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}

	case tickMsg:
		return m, tea.Batch(m.loadSessions(), tick())

	case error:
		m.err = msg
	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString("Claude Conductor\n\n")

	if len(m.sessions) == 0 {
		b.WriteString("No active sessions\n")
	}

	for i, sess := range m.sessions {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}

		elapsed := time.Since(sess.Started).Round(time.Second)
		preview, _ := m.agent.Tail(sess)
		preview = strings.TrimSpace(preview)
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}

		b.WriteString(fmt.Sprintf("%s %d  %-20s  %-10s  %8s  %s\n",
			cursor, i, truncate(sess.Task, 20), sess.Status, elapsed, preview))
	}

	b.WriteString("\n[↑↓] navigate  [enter] attach  [d] delete  [r] refresh  [q] quit\n")

	if m.err != nil {
		b.WriteString(fmt.Sprintf("\nError: %v\n", m.err))
	}

	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func (m Model) Attach() *Session {
	return m.attach
}
