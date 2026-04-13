package usage

import "time"

type WindowUsage struct {
	UsedPercent float64   `json:"used_percent"`
	ResetAt     time.Time `json:"reset_at"`
}

type ProviderUsage struct {
	FiveHour WindowUsage `json:"five_hour"`
	Weekly   WindowUsage `json:"weekly"`
}

type Snapshot struct {
	Claude    ProviderUsage `json:"claude"`
	Codex     ProviderUsage `json:"codex"`
	UpdatedAt time.Time     `json:"updated_at"`
}
