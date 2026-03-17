package models

import "time"

type SessionState string

const (
	SessionStateCreated   SessionState = "created"
	SessionStateBuilding  SessionState = "building"
	SessionStateRunning   SessionState = "running"
	SessionStateReloading SessionState = "reloading"
	SessionStateStopped   SessionState = "stopped"
	SessionStateDestroyed SessionState = "destroyed"
)

type Session struct {
	ID        string       `json:"id"`
	AgentID   string       `json:"agent_id"`
	Name      string       `json:"name,omitempty"`
	Platform  PlatformType `json:"platform"`
	State     SessionState `json:"state"`
	Device    *Device      `json:"device,omitempty"`
	WorkDir   string       `json:"work_dir,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

type Agent struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}
