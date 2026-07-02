package core

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
)

// DailyStats holds traffic consumption for a single day.
type DailyStats struct {
	Date     string `json:"date"`     // YYYY-MM-DD
	Uplink   int64  `json:"uplink"`   // bytes
	Downlink int64  `json:"downlink"` // bytes
}

// StatsHistory holds the entire traffic history.
type StatsHistory struct {
	Days          []DailyStats `json:"days"`
	TotalUplink   int64        `json:"totalUplink"`
	TotalDownlink int64        `json:"totalDownlink"`
	LastUpdated   string       `json:"lastUpdated"`
}

// statsManager handles traffic stats persistence.
var (
	statsOnce    sync.Once
	statsManager *StatsManager
)

// StatsManager manages traffic stats accumulation and persistence.
type StatsManager struct {
	mu           sync.Mutex
	history      *StatsHistory
	lastUplink   int64 // last seen cumulative uplink from xray
	lastDownlink int64 // last seen cumulative downlink from xray
	initialized  bool
}

// GetStatsManager returns the singleton stats manager.
func GetStatsManager() *StatsManager {
	statsOnce.Do(func() {
		statsManager = &StatsManager{}
		statsManager.load()
	})
	return statsManager
}

func (sm *StatsManager) load() {
	sm.history = &StatsHistory{}
	data, err := os.ReadFile(config.StatsPath())
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, sm.history)
}

// Save persists the stats history to disk.
func (sm *StatsManager) Save() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.saveLocked()
}

func (sm *StatsManager) saveLocked() error {
	sm.history.LastUpdated = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(sm.history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(config.StatsPath(), data, 0644)
}

// RecordTraffic updates the traffic history with current xray stats.
// Should be called periodically with cumulative counters from xray.
func (sm *StatsManager) RecordTraffic(stats *TrafficStats) {
	if stats == nil {
		return
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()

	today := time.Now().Format("2006-01-02")

	if !sm.initialized {
		// First call — just record the baseline
		sm.lastUplink = stats.Uplink
		sm.lastDownlink = stats.Downlink
		sm.initialized = true
		return
	}

	// Calculate delta since last recording
	upDelta := stats.Uplink - sm.lastUplink
	dnDelta := stats.Downlink - sm.lastDownlink

	// Handle counter resets (xray restart)
	if upDelta < 0 {
		upDelta = stats.Uplink
	}
	if dnDelta < 0 {
		dnDelta = stats.Downlink
	}

	sm.lastUplink = stats.Uplink
	sm.lastDownlink = stats.Downlink

	if upDelta == 0 && dnDelta == 0 {
		return
	}

	// Find or create today's entry
	found := false
	for i := range sm.history.Days {
		if sm.history.Days[i].Date == today {
			sm.history.Days[i].Uplink += upDelta
			sm.history.Days[i].Downlink += dnDelta
			found = true
			break
		}
	}
	if !found {
		sm.history.Days = append(sm.history.Days, DailyStats{
			Date:     today,
			Uplink:   upDelta,
			Downlink: dnDelta,
		})
	}

	sm.history.TotalUplink += upDelta
	sm.history.TotalDownlink += dnDelta

	// Prune old entries (keep last 90 days)
	sm.pruneOldEntries(90)
}

func (sm *StatsManager) pruneOldEntries(keepDays int) {
	if len(sm.history.Days) <= keepDays {
		return
	}
	sm.history.Days = sm.history.Days[len(sm.history.Days)-keepDays:]
}

// GetHistory returns a copy of the stats history.
func (sm *StatsManager) GetHistory() StatsHistory {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	h := *sm.history
	days := make([]DailyStats, len(sm.history.Days))
	copy(days, sm.history.Days)
	h.Days = days
	return h
}

// TodayStats returns today's traffic stats.
func (sm *StatsManager) TodayStats() DailyStats {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	today := time.Now().Format("2006-01-02")
	for _, d := range sm.history.Days {
		if d.Date == today {
			return d
		}
	}
	return DailyStats{Date: today}
}

// MonthStats returns the total traffic for the current month.
func (sm *StatsManager) MonthStats() DailyStats {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	prefix := time.Now().Format("2006-01")
	var total DailyStats
	total.Date = prefix
	for _, d := range sm.history.Days {
		if len(d.Date) >= 7 && d.Date[:7] == prefix {
			total.Uplink += d.Uplink
			total.Downlink += d.Downlink
		}
	}
	return total
}

// FormatStatsReport returns a human-readable stats report.
func FormatStatsReport(history StatsHistory) string {
	var result string

	today := time.Now().Format("2006-01-02")
	monthPrefix := time.Now().Format("2006-01")

	var todayUp, todayDn, monthUp, monthDn int64
	for _, d := range history.Days {
		if d.Date == today {
			todayUp = d.Uplink
			todayDn = d.Downlink
		}
		if len(d.Date) >= 7 && d.Date[:7] == monthPrefix {
			monthUp += d.Uplink
			monthDn += d.Downlink
		}
	}

	result += fmt.Sprintf("Today:     ↑ %s  ↓ %s\n", FormatBytes(todayUp), FormatBytes(todayDn))
	result += fmt.Sprintf("This month: ↑ %s  ↓ %s\n", FormatBytes(monthUp), FormatBytes(monthDn))
	result += fmt.Sprintf("All time:   ↑ %s  ↓ %s\n", FormatBytes(history.TotalUplink), FormatBytes(history.TotalDownlink))

	if len(history.Days) > 0 {
		result += fmt.Sprintf("\nLast %d days:\n", len(history.Days))
		// Show last 7 days
		start := len(history.Days) - 7
		if start < 0 {
			start = 0
		}
		for i := start; i < len(history.Days); i++ {
			d := history.Days[i]
			result += fmt.Sprintf("  %s  ↑ %-10s  ↓ %s\n", d.Date, FormatBytes(d.Uplink), FormatBytes(d.Downlink))
		}
	}

	return result
}
