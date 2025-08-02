package health

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HealthTestSuite struct {
	suite.Suite
	manager *Manager
	ctx     context.Context
}

func (suite *HealthTestSuite) SetupTest() {
	suite.manager = NewManager()
	suite.ctx = context.Background()
}

func (suite *HealthTestSuite) TestNewManager() {
	manager := NewManager()

	require.NotNil(suite.T(), manager)
	assert.Empty(suite.T(), manager.checkers)
}

func (suite *HealthTestSuite) TestInterfaceCompliance() {
	var _ ManagerInterface = (*Manager)(nil)
}

func (suite *HealthTestSuite) TestRegister_SingleChecker() {
	mockChecker := &mockHealthChecker{
		name:   "test-checker",
		result: CheckResult{Status: StatusHealthy, Message: "OK"},
	}

	suite.manager.Register(mockChecker)

	suite.manager.mu.RLock()
	assert.Len(suite.T(), suite.manager.checkers, 1)
	assert.Equal(suite.T(), mockChecker, suite.manager.checkers[0])
	suite.manager.mu.RUnlock()
}

func (suite *HealthTestSuite) TestRegister_MultipleCheckers() {
	checker1 := &mockHealthChecker{name: "checker1", result: CheckResult{Status: StatusHealthy}}
	checker2 := &mockHealthChecker{name: "checker2", result: CheckResult{Status: StatusHealthy}}
	checker3 := &mockHealthChecker{name: "checker3", result: CheckResult{Status: StatusUnhealthy}}

	suite.manager.Register(checker1)
	suite.manager.Register(checker2)
	suite.manager.Register(checker3)

	suite.manager.mu.RLock()
	assert.Len(suite.T(), suite.manager.checkers, 3)
	suite.manager.mu.RUnlock()
}

func (suite *HealthTestSuite) TestCheckAll_NoCheckers() {
	results := suite.manager.CheckAll(suite.ctx)

	assert.NotNil(suite.T(), results)
	assert.Empty(suite.T(), results)
}

func (suite *HealthTestSuite) TestCheckAll_SingleHealthyChecker() {
	mockChecker := &mockHealthChecker{
		name:   "database",
		result: CheckResult{Status: StatusHealthy, Message: "Connection successful"},
		delay:  1 * time.Millisecond,
	}
	suite.manager.Register(mockChecker)

	results := suite.manager.CheckAll(suite.ctx)

	require.Len(suite.T(), results, 1)
	result, exists := results["database"]
	require.True(suite.T(), exists)
	assert.Equal(suite.T(), StatusHealthy, result.Status)
	assert.Equal(suite.T(), "Connection successful", result.Message)
	assert.Greater(suite.T(), result.Latency, time.Duration(0))
}

func (suite *HealthTestSuite) TestCheckAll_SingleUnhealthyChecker() {
	mockChecker := &mockHealthChecker{
		name:   "redis",
		result: CheckResult{Status: StatusUnhealthy, Message: "Connection failed", Error: "timeout"},
		delay:  1 * time.Millisecond,
	}
	suite.manager.Register(mockChecker)

	results := suite.manager.CheckAll(suite.ctx)

	require.Len(suite.T(), results, 1)
	result, exists := results["redis"]
	require.True(suite.T(), exists)
	assert.Equal(suite.T(), StatusUnhealthy, result.Status)
	assert.Equal(suite.T(), "Connection failed", result.Message)
	assert.Equal(suite.T(), "timeout", result.Error)
	assert.Greater(suite.T(), result.Latency, time.Duration(0))
}

func (suite *HealthTestSuite) TestCheckAll_MixedCheckers() {
	healthyChecker := &mockHealthChecker{
		name:   "database",
		result: CheckResult{Status: StatusHealthy, Message: "OK"},
	}
	unhealthyChecker := &mockHealthChecker{
		name:   "external-api",
		result: CheckResult{Status: StatusUnhealthy, Error: "service unavailable"},
	}

	suite.manager.Register(healthyChecker)
	suite.manager.Register(unhealthyChecker)

	results := suite.manager.CheckAll(suite.ctx)

	require.Len(suite.T(), results, 2)

	dbResult, exists := results["database"]
	require.True(suite.T(), exists)
	assert.Equal(suite.T(), StatusHealthy, dbResult.Status)

	apiResult, exists := results["external-api"]
	require.True(suite.T(), exists)
	assert.Equal(suite.T(), StatusUnhealthy, apiResult.Status)
}

func (suite *HealthTestSuite) TestCheckAll_LatencyMeasurement() {
	slowChecker := &mockHealthChecker{
		name:   "slow-service",
		result: CheckResult{Status: StatusHealthy},
		delay:  50 * time.Millisecond,
	}
	suite.manager.Register(slowChecker)

	start := time.Now()
	results := suite.manager.CheckAll(suite.ctx)
	totalDuration := time.Since(start)

	require.Len(suite.T(), results, 1)
	result := results["slow-service"]

	assert.GreaterOrEqual(suite.T(), result.Latency, 50*time.Millisecond)
	assert.GreaterOrEqual(suite.T(), totalDuration, 50*time.Millisecond)
}

func (suite *HealthTestSuite) TestIsHealthy_AllHealthy() {
	checker1 := &mockHealthChecker{name: "db", result: CheckResult{Status: StatusHealthy}}
	checker2 := &mockHealthChecker{name: "redis", result: CheckResult{Status: StatusHealthy}}

	suite.manager.Register(checker1)
	suite.manager.Register(checker2)

	isHealthy := suite.manager.IsHealthy(suite.ctx)
	assert.True(suite.T(), isHealthy)
}

func (suite *HealthTestSuite) TestIsHealthy_SomeUnhealthy() {
	healthyChecker := &mockHealthChecker{name: "db", result: CheckResult{Status: StatusHealthy}}
	unhealthyChecker := &mockHealthChecker{name: "api", result: CheckResult{Status: StatusUnhealthy}}

	suite.manager.Register(healthyChecker)
	suite.manager.Register(unhealthyChecker)

	isHealthy := suite.manager.IsHealthy(suite.ctx)
	assert.False(suite.T(), isHealthy)
}

func (suite *HealthTestSuite) TestIsHealthy_NoCheckers() {
	isHealthy := suite.manager.IsHealthy(suite.ctx)
	assert.True(suite.T(), isHealthy)
}

func (suite *HealthTestSuite) TestConcurrentAccess() {
	const numGoroutines = 10
	const numCheckers = 5

	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numCheckers; j++ {
				checker := &mockHealthChecker{
					name:   "checker",
					result: CheckResult{Status: StatusHealthy},
				}
				suite.manager.Register(checker)
			}
		}(i)
	}

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = suite.manager.IsHealthy(suite.ctx)
				_ = suite.manager.CheckAll(suite.ctx)
			}
		}()
	}

	wg.Wait()

	suite.manager.mu.RLock()
	totalCheckers := len(suite.manager.checkers)
	suite.manager.mu.RUnlock()

	assert.Equal(suite.T(), numGoroutines*numCheckers, totalCheckers)
}

func TestHealthTestSuite(t *testing.T) {
	suite.Run(t, new(HealthTestSuite))
}

func TestStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected string
	}{
		{
			name:     "healthy status",
			status:   StatusHealthy,
			expected: "healthy",
		},
		{
			name:     "unhealthy status",
			status:   StatusUnhealthy,
			expected: "unhealthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestCheckResult_Structure(t *testing.T) {
	result := CheckResult{
		Status:  StatusHealthy,
		Message: "All systems operational",
		Latency: 100 * time.Millisecond,
		Error:   "",
	}

	assert.Equal(t, StatusHealthy, result.Status)
	assert.Equal(t, "All systems operational", result.Message)
	assert.Equal(t, 100*time.Millisecond, result.Latency)
	assert.Empty(t, result.Error)
}

func TestManager_EdgeCases(t *testing.T) {
	t.Run("register nil checker", func(t *testing.T) {
		manager := NewManager()

		assert.NotPanics(t, func() {
			manager.Register(nil)
		})

		assert.Panics(t, func() {
			manager.CheckAll(context.Background())
		})
	})

	t.Run("context cancellation", func(t *testing.T) {
		manager := NewManager()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		checker := &mockHealthChecker{
			name:   "test",
			result: CheckResult{Status: StatusHealthy},
		}
		manager.Register(checker)

		results := manager.CheckAll(ctx)
		assert.Len(t, results, 1)
	})
}

func TestManager_MemoryUsage(t *testing.T) {
	manager := NewManager()

	for i := 0; i < 1000; i++ {
		checker := &mockHealthChecker{
			name:   fmt.Sprintf("checker-%d", i),
			result: CheckResult{Status: StatusHealthy},
		}
		manager.Register(checker)
	}

	manager.mu.RLock()
	assert.Len(t, manager.checkers, 1000)
	manager.mu.RUnlock()

	results := manager.CheckAll(context.Background())
	assert.Len(t, results, 1000)
}

type mockHealthChecker struct {
	name   string
	result CheckResult
	delay  time.Duration
	mu     sync.Mutex
	calls  int
}

func (m *mockHealthChecker) Name() string {
	return m.name
}

func (m *mockHealthChecker) Check(context.Context) CheckResult {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()

	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	return m.result
}

func (m *mockHealthChecker) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestMockHealthChecker(t *testing.T) {
	checker := &mockHealthChecker{
		name:   "test",
		result: CheckResult{Status: StatusHealthy, Message: "OK"},
	}

	assert.Equal(t, "test", checker.Name())
	assert.Equal(t, 0, checker.CallCount())

	result := checker.Check(context.Background())
	assert.Equal(t, StatusHealthy, result.Status)
	assert.Equal(t, "OK", result.Message)
	assert.Equal(t, 1, checker.CallCount())
}
