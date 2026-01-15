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
		taskFile = flag.String("f", "", "read tasks from file (- for stdin)")
		help     = flag.Bool("h", false, "show help")
	)
	flag.Parse()

	if *help {
		usage()
		return
	}

	tasks, err := parseTasks(*taskFile, flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(tasks) == 0 {
		if err := runTUI(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := spawnTasks(tasks); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := runTUI(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`conductor - manage multiple Claude sessions

Usage:
  conductor [options] "task1" "task2" "task3"
  conductor -f tasks.txt
  conductor (starts TUI for existing sessions)

Options:
  -f FILE    Read tasks from file (- for stdin)
  -h         Show help`)
}

func parseTasks(file string, args []string) ([]string, error) {
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

func spawnTasks(tasks []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	baseDir := filepath.Join(home, ".conductor")
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

	for _, task := range tasks {
		task = strings.TrimSpace(task)
		if task == "" {
			continue
		}

		sess := &Session{
			ID:      genID(),
			Task:    task,
			Status:  Running,
			Started: time.Now(),
		}

		if err := mgr.Spawn(sess); err != nil {
			return err
		}

		if err := store.Save(sess); err != nil {
			return err
		}

		fmt.Printf("Started: %s (%s)\n", truncate(task, 50), sess.ID)
	}

	return nil
}

func runTUI() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	baseDir := filepath.Join(home, ".conductor")
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
		fmt.Printf("\nAttaching to: %s\n", sess.Task)
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
