package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	var (
		file       = flag.String("f", "", "read prompts from file (- for stdin)")
		help       = flag.Bool("h", false, "show help")
		useWorktree = flag.Bool("w", false, "create git worktrees for each session")
		worktreeDir = flag.String("d", "", "directory for worktrees (default: ~/.claudes-work)")
	)
	flag.Parse()

	if *help {
		usage()
		return
	}

	prompts, err := parsePrompts(*file, flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Start spawning sessions in background (non-blocking)
	if len(prompts) > 0 {
		go func() {
			if err := spawn(prompts, *useWorktree, *worktreeDir); err != nil {
				fmt.Fprintf(os.Stderr, "Error spawning sessions: %v\n", err)
			}
		}()
	}

	// Start TUI immediately
	if err := runTUI(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`claudes - manage multiple Claude sessions

Usage:
  claudes [options] "prompt1" "prompt2" "prompt3"
  claudes -f prompts.txt
  claudes (starts TUI for existing sessions)

Options:
  -f FILE    Read prompts from file (- for stdin)
  -w         Create git worktrees for each session
  -d DIR     Directory for worktrees (default: ~/.claudes-work)
  -h         Show help`)
}

func parsePrompts(file string, args []string) ([]string, error) {
	if file != "" {
		var r io.Reader
		if file == "-" {
			r = os.Stdin
		} else {
			f, err := os.Open(file)
			if err != nil {
				return nil, err
			}
			defer f.Close()
			r = f
		}
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		return strings.Split(string(data), "\n\n\n"), nil
	}
	return args, nil
}

func spawn(prompts []string, useWorktree bool, worktreeDir string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	baseDir := filepath.Join(home, ".claudes")
	sessionDir := filepath.Join(baseDir, "sessions")
	outputDir := filepath.Join(baseDir, "outputs")

	// Set default worktree directory
	if worktreeDir == "" {
		worktreeDir = filepath.Join(home, ".claudes-work")
	}

	store, err := NewStore(sessionDir)
	if err != nil {
		return err
	}

	mgr, err := NewManager(outputDir)
	if err != nil {
		return err
	}

	// Spawn each session concurrently
	for _, prompt := range prompts {
		prompt = strings.TrimSpace(prompt)
		if prompt == "" {
			continue
		}

		// Spawn in goroutine for parallel execution
		go func(p string) {
			sess := &Session{
				ID:      genID(),
				Prompt:  p,
				Status:  Running,
				Started: time.Now(),
			}

			// Generate worktree name and create worktree if enabled
			if useWorktree {
				ctx := context.Background()
				worktreeName, err := generateWorktreeName(ctx, p, sess.ID)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to generate worktree name: %v\n", err)
					// Continue without worktree on error
				} else {
					// Create the worktree
					info, err := createWorktree(worktreeDir, worktreeName, worktreeName)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to create worktree: %v\n", err)
						// Continue without worktree on error
					} else {
						sess.WorktreePath = info.Path
						sess.WorktreeName = info.Name
						sess.BranchName = info.Branch
						fmt.Printf("Created worktree: %s (branch: %s)\n", info.Path, info.Branch)
					}
				}
			}

			if err := mgr.Spawn(sess); err != nil {
				fmt.Fprintf(os.Stderr, "Error spawning session: %v\n", err)
				return
			}

			if err := store.Save(sess); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving session: %v\n", err)
				return
			}

			fmt.Printf("Started: %s (%s)\n", truncate(p, 50), sess.ID)
		}(prompt)
	}

	return nil
}

func runTUI() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	baseDir := filepath.Join(home, ".claudes")
	sessionDir := filepath.Join(baseDir, "sessions")
	outputDir := filepath.Join(baseDir, "outputs")

	store, err := NewStore(sessionDir)
	if err != nil {
		return err
	}

	mgr, err := NewManager(outputDir)
	if err != nil {
		return err
	}

	for {
		m := New(store, mgr)
		p := tea.NewProgram(m)
		final, err := p.Run()
		if err != nil {
			return err
		}

		model := final.(Model)
		if model.Attach() == nil {
			break
		}

		sess := model.Attach()
		fmt.Printf("\nAttaching to: %s\n", sess.Prompt)
		fmt.Print("Follow-up: ")

		var followup string
		fmt.Scanln(&followup)

		if followup == "" {
			continue
		}

		ctx := context.Background()
		if err := mgr.Attach(ctx, sess, followup); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}

		fmt.Println("\n\nPress Enter to return to TUI...")
		fmt.Scanln()
	}

	return nil
}

func genID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}
