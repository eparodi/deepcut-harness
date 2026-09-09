package wizard

import "sync"

// Manager holds live wizard sessions in memory, keyed by an opaque id.
// Harness is single-user (loopback), so sessions are short-lived and never
// persisted; a restart drops them.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*Session
}

// NewManager returns an empty Manager.
func NewManager() *Manager {
	return &Manager{sessions: map[string]*Session{}}
}

// Put stores a session under id.
func (m *Manager) Put(id string, s *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[id] = s
}

// Get returns the session under id.
func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	return s, ok
}

// Delete removes the session under id.
func (m *Manager) Delete(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}
