// events.go - Kill Event Participation
//
// Fetches active kill events from the server and lets the user select
// which event to participate in. The selected event ID is sent along
// with SaveKills requests so kills count towards the event leaderboard.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

// KillEvent represents an active kill event from the server.
type KillEvent struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	Description      string `json:"description"`
	StartDate        string `json:"startDate"`
	EndDate          string `json:"endDate"`
	Status           string `json:"status"`
	ScoringMetric    string `json:"scoringMetric"`
	GameModeFilter   *string `json:"gameModeFilter"`
	Prize            string `json:"prize"`
	ImageURL         *string `json:"imageUrl"`
	ParticipantCount int    `json:"participantCount"`
}

// KillEventsResponse is the server response for GET /api/kill-events.
type KillEventsResponse struct {
	Events []KillEvent `json:"events"`
}

var (
	activeEvents    []KillEvent
	selectedEventID string // Empty = no event selected
	eventsMutex     sync.RWMutex
)

// FetchActiveEvents queries the server for currently active kill events.
func FetchActiveEvents() ([]KillEvent, error) {
	url := strings.Replace(APIURL, "/api/ocr", "/api/kill-events?status=active", 1)

	resp, err := apiClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("events API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read events response: %v", err)
	}

	var eventsResp KillEventsResponse
	if err := json.Unmarshal(body, &eventsResp); err != nil {
		return nil, fmt.Errorf("failed to parse events: %v", err)
	}

	return eventsResp.Events, nil
}

// RefreshEvents fetches active events and updates the global list.
func RefreshEvents() {
	events, err := FetchActiveEvents()
	if err != nil {
		debugLog("[EVENTS] Failed to fetch: %v\n", err)
		return
	}

	eventsMutex.Lock()
	activeEvents = events
	// If selected event is no longer active, clear selection
	if selectedEventID != "" && !isEventActive(selectedEventID) {
		selectedEventID = ""
	}
	eventsMutex.Unlock()

	debugLog("[EVENTS] Loaded %d active events\n", len(events))
}

// GetActiveEvents returns a copy of the current active events list.
func GetActiveEvents() []KillEvent {
	eventsMutex.RLock()
	defer eventsMutex.RUnlock()
	result := make([]KillEvent, len(activeEvents))
	copy(result, activeEvents)
	return result
}

// GetSelectedEventID returns the currently selected event ID (empty = none).
func GetSelectedEventID() string {
	eventsMutex.RLock()
	defer eventsMutex.RUnlock()
	return selectedEventID
}

// SetSelectedEventID sets the active event for kill tracking.
func SetSelectedEventID(id string) {
	eventsMutex.Lock()
	selectedEventID = id
	eventsMutex.Unlock()
	if id == "" {
		debugLn("[EVENTS] Event deselected")
	} else {
		debugLog("[EVENTS] Selected event: %s\n", id)
	}
}

// isEventActive checks if an event ID is in the active list. Must hold eventsMutex.
func isEventActive(id string) bool {
	for _, e := range activeEvents {
		if e.ID == id {
			return true
		}
	}
	return false
}

// IsEventError checks if a save error is event-related and returns true if
// the kills should be retried without event.
func IsEventError(errMsg string) bool {
	eventErrors := []string{
		"Kill Event nicht gefunden",
		"Kill Event ist nicht aktiv",
		"Kill Event ist außerhalb des Zeitraums",
	}
	for _, e := range eventErrors {
		if strings.Contains(errMsg, e) {
			return true
		}
	}
	return false
}
