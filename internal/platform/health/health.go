package health

import (
	"context"
	"sync"
	"time"
)

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
)

type CheckResult struct {
	Status  Status        `json:"status"`
	Message string        `json:"message,omitempty"`
	Latency time.Duration `json:"latency"`
	Error   string        `json:"error,omitempty"`
}

type Checker interface {
	Name() string
	Check(ctx context.Context) CheckResult
}

type ManagerInterface interface {
	Register(checker Checker)
	CheckAll(ctx context.Context) map[string]CheckResult
	IsHealthy(ctx context.Context) bool
}

type Manager struct {
	checkers []Checker
	mu       sync.RWMutex
}

// Compile-time interface check
var _ ManagerInterface = (*Manager)(nil)

func NewManager() *Manager {
	return &Manager{
		checkers: make([]Checker, 0),
	}
}

func (m *Manager) Register(checker Checker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkers = append(m.checkers, checker)
}

func (m *Manager) CheckAll(ctx context.Context) map[string]CheckResult {
	m.mu.RLock()
	checkers := make([]Checker, len(m.checkers))
	copy(checkers, m.checkers)
	m.mu.RUnlock()

	results := make(map[string]CheckResult)

	for _, checker := range checkers {
		start := time.Now()
		result := checker.Check(ctx)
		result.Latency = time.Since(start)

		results[checker.Name()] = result
	}

	return results
}

func (m *Manager) IsHealthy(ctx context.Context) bool {
	results := m.CheckAll(ctx)

	for _, result := range results {
		if result.Status == StatusUnhealthy {
			return false
		}
	}

	return true
}
