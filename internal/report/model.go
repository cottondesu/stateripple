package report

import (
	"github.com/cottondesu/stateripple/internal/graph"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

type Cause struct {
	Source string      `json:"source"`
	Edge   *graph.Edge `json:"edge,omitempty"`
	Reason string      `json:"reason"`
}
type Cell struct {
	ID                    string          `json:"id"`
	Kind                  string          `json:"kind"`
	Risk                  int             `json:"risk"`
	Seeds                 []snapshot.Seed `json:"seeds"`
	FirstActiveGeneration *int            `json:"first_active_generation"`
	Causes                []Cause         `json:"causes"`
}
type Change struct {
	ID     string  `json:"id"`
	Before int     `json:"before"`
	After  int     `json:"after"`
	Causes []Cause `json:"causes"`
}
type Wave struct {
	Generation int      `json:"generation"`
	Changes    []Change `json:"changes"`
}
type TestTarget struct {
	ID   string `json:"id"`
	Risk int    `json:"risk"`
}
type BuildContext struct {
	GoVersion   string   `json:"go_version"`
	GOOS        string   `json:"goos"`
	GOARCH      string   `json:"goarch"`
	CGOEnabled  string   `json:"cgo_enabled"`
	Module      string   `json:"module"`
	BuildTags   []string `json:"build_tags"`
	ToolTags    []string `json:"tool_tags"`
	ReleaseTags []string `json:"release_tags"`
}
type FileChange struct {
	Path           string `json:"path"`
	Status         string `json:"status"`
	Classification string `json:"classification"`
}
type Report struct {
	SchemaVersion   int           `json:"schema_version"`
	Ruleset         string        `json:"ruleset"`
	Status          string        `json:"status"`
	InputSHA256     string        `json:"input_sha256"`
	GenerationCount int           `json:"generation_count"`
	Cells           []Cell        `json:"cells"`
	TestTargets     []TestTarget  `json:"test_targets"`
	Waves           []Wave        `json:"waves"`
	UnknownChanges  []string      `json:"unknown_changes"`
	Warnings        []string      `json:"warnings"`
	Changes         []FileChange  `json:"changes"`
	BuildContext    *BuildContext `json:"build_context"`
	StopReason      string        `json:"stop_reason"`
}
