package health

import "time"

type Status string

const (
	StatusPass Status = "pass"
	StatusFail Status = "fail"
	StatusWarn Status = "warn"
)

type LivenessResponse struct {
	Status    Status    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version,omitempty"`
}

type ReadinessResponse struct {
	Status    Status                   `json:"status"`
	Version   string                   `json:"version"`
	ReleaseId string                   `json:"releaseId,omitempty"`
	Notes     []string                 `json:"notes,omitempty"`
	Output    string                   `json:"output,omitempty"`
	Checks    map[string][]CheckDetail `json:"checks,omitempty"`
}

type CheckDetail struct {
	ComponentId   string    `json:"componentId,omitempty"`
	ComponentType string    `json:"componentType,omitempty"`
	Status        Status    `json:"status"`
	Time          time.Time `json:"time"`
	Output        string    `json:"output,omitempty"`
}
