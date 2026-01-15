package claudes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	claudecode "github.com/severity1/claude-agent-sdk-go"
)

type Manager struct {
	outputDir string
	clients   map[string]context.CancelFunc
	mu        sync.Mutex
}

func NewManager(outputDir string) (*Manager, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}
	return &Manager{
		outputDir: outputDir,
		clients:   make(map[string]context.CancelFunc),
	}, nil
}

func (m *Manager) Spawn(sess *Session) error {
	outputPath := filepath.Join(m.outputDir, sess.ID+".txt")
	sess.OutputFile = outputPath

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.clients[sess.ID] = cancel
	m.mu.Unlock()

	go func() {
		defer outFile.Close()
		defer func() {
			m.mu.Lock()
			delete(m.clients, sess.ID)
			m.mu.Unlock()
		}()

		// Change to worktree directory if specified
		var originalDir string
		if sess.WorktreePath != "" {
			originalDir, _ = os.Getwd()
			if err := os.Chdir(sess.WorktreePath); err != nil {
				fmt.Fprintf(outFile, "Warning: failed to change to worktree directory: %v\n", err)
			} else {
				defer func() {
					if originalDir != "" {
						os.Chdir(originalDir)
					}
				}()
			}
		}

		err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
			if err := client.QueryWithSession(ctx, sess.Prompt, sess.ID); err != nil {
				return err
			}

			msgChan := client.ReceiveMessages(ctx)
			for msg := range msgChan {
				switch m := msg.(type) {
				case *claudecode.AssistantMessage:
					for _, block := range m.Content {
						if tb, ok := block.(*claudecode.TextBlock); ok {
							fmt.Fprint(outFile, tb.Text)
						}
					}
				}
			}
			return nil
		})

		if err != nil {
			fmt.Fprintf(outFile, "\nSession error: %v\n", err)
		}
	}()

	return nil
}

func (m *Manager) Tail(sess *Session) (string, error) {
	data, err := os.ReadFile(sess.OutputFile)
	if err != nil {
		return "", err
	}
	if len(data) > 200 {
		return string(data[len(data)-200:]), nil
	}
	return string(data), nil
}

func (m *Manager) Stop(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cancel, ok := m.clients[id]; ok {
		cancel()
	}
}

func (m *Manager) Attach(ctx context.Context, sess *Session, followup string) error {
	outFile, err := os.OpenFile(sess.OutputFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer outFile.Close()

	fmt.Fprintf(outFile, "\n\n--- Follow-up ---\n")

	return claudecode.WithClient(ctx, func(client claudecode.Client) error {
		if err := client.QueryWithSession(ctx, followup, sess.ID); err != nil {
			return err
		}

		msgChan := client.ReceiveMessages(ctx)
		for msg := range msgChan {
			switch m := msg.(type) {
			case *claudecode.AssistantMessage:
				for _, block := range m.Content {
					if tb, ok := block.(*claudecode.TextBlock); ok {
						text := tb.Text
						fmt.Fprint(os.Stdout, text)
						fmt.Fprint(outFile, text)
					}
				}
			}
		}
		return nil
	})
}
