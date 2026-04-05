package session

import "sync"

// State represents the current conversational state of a user.
type State string

const (
	StateNone                State = ""
	StateWaitingForCity      State = "waiting_for_city"
	StateWaitingForReminder  State = "waiting_for_reminder"
	StateWaitingForTime      State = "waiting_for_data_reminder"
	StateWaitingForPlan      State = "waiting_for_plan"
	StateWaitingForOwnTime   State = "waiting_for_own_time"
)

// Session holds per-user conversational context.
type Session struct {
	State           State
	AlertMessage    string
	MentionedIDs    []int64
	PendingContent  string // content waiting for a time to be set
}

// Manager is a thread-safe in-memory session store.
type Manager struct {
	mu       sync.RWMutex
	sessions map[int64]Session
}

// NewManager creates a new session Manager.
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[int64]Session),
	}
}

// Get returns the session for a user. Returns a zero Session if not found.
func (m *Manager) Get(userID int64) Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[userID]
}

// Set overwrites the session for a user.
func (m *Manager) Set(userID int64, s Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[userID] = s
}

// Clear removes the session for a user.
func (m *Manager) Clear(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, userID)
}

// SetState is a convenience method that updates only the State field.
func (m *Manager) SetState(userID int64, state State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.sessions[userID]
	s.State = state
	m.sessions[userID] = s
}
