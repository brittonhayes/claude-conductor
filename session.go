package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Status string

const (
	Running  Status = "Running"
	Complete Status = "Complete"
	Failed   Status = "Failed"
)

type Session struct {
	ID            string    `json:"id"`
	Prompt        string    `json:"prompt"`
	Status        Status    `json:"status"`
	OutputFile    string    `json:"output_file"`
	Started       time.Time `json:"started"`
	AgentID       string    `json:"agent_id,omitempty"`
	WorktreePath  string    `json:"worktree_path,omitempty"`
	WorktreeName  string    `json:"worktree_name,omitempty"`
	BranchName    string    `json:"branch_name,omitempty"`
}

type Store struct {
	dir string
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Save(sess *Session) error {
	path := filepath.Join(s.dir, sess.ID+".json")
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *Store) Load(id string) (*Session, error) {
	path := filepath.Join(s.dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) List() ([]*Session, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var sessions []*Session
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		id := e.Name()[:len(e.Name())-5]
		sess, err := s.Load(id)
		if err != nil {
			continue
		}
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

func (s *Store) Delete(id string) error {
	// Load session to check for worktree
	sess, err := s.Load(id)
	if err == nil && sess.WorktreePath != "" {
		// Try to remove the worktree, but don't fail if it errors
		removeWorktree(sess.WorktreePath)
	}

	path := filepath.Join(s.dir, id+".json")
	return os.Remove(path)
}
