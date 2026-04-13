package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chaso/ai-usage-monitor/internal/cache"
)

func main() {
	path := cachePath()
	store := cache.New(path)

	snap, err := store.Read()
	if err != nil {
		fmt.Print("CLAUDE:-- CODEX:--")
		return
	}

	clFiveH := remaining(snap.Claude.FiveHour.UsedPercent)
	clWeekly := remaining(snap.Claude.Weekly.UsedPercent)
	cdFiveH := remaining(snap.Codex.FiveHour.UsedPercent)
	cdWeekly := remaining(snap.Codex.Weekly.UsedPercent)

	fmt.Printf("CLAUDE:%d/%d CODEX:%d/%d", clFiveH, clWeekly, cdFiveH, cdWeekly)
}

func remaining(usedPct float64) int {
	r := 100.0 - usedPct
	if r < 0 {
		r = 0
	}
	return int(r)
}

func cachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "ai-usage.json")
}
