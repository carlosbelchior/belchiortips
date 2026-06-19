package tracker

import (
	"encoding/json"
	"os"
	"sync"
)

const saveFile = "tracker_data.json"

// Tracker keeps track of the mapped opportunities to telegram message IDs.
type Tracker struct {
	mu        sync.RWMutex
	messages  map[int64]int    // Map OpportunityID -> Telegram MessageID
	statusMap map[int64]string // Map OpportunityID -> Status (pending, win, lose)
}

type trackerData struct {
	Messages  map[int64]int    `json:"messages"`
	StatusMap map[int64]string `json:"statusMap"`
}

// NewTracker creates a new tracker instance and loads from disk
func NewTracker() *Tracker {
	t := &Tracker{
		messages:  make(map[int64]int),
		statusMap: make(map[int64]string),
	}
	t.load()
	return t
}

func (t *Tracker) load() {
	data, err := os.ReadFile(saveFile)
	if err == nil {
		var td trackerData
		if err := json.Unmarshal(data, &td); err == nil {
			if td.Messages != nil {
				t.messages = td.Messages
			}
			if td.StatusMap != nil {
				t.statusMap = td.StatusMap
			}
		}
	}
}

func (t *Tracker) save() {
	td := trackerData{
		Messages:  t.messages,
		StatusMap: t.statusMap,
	}
	data, _ := json.MarshalIndent(td, "", "  ")
	_ = os.WriteFile(saveFile, data, 0644)
}

// SaveMessageID saves the telegram message ID for an opportunity
func (t *Tracker) SaveMessageID(oppID int64, msgID int, status string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.messages[oppID] = msgID
	t.statusMap[oppID] = status
	t.save()
}

// GetMessageID retrieves the message ID for a given opportunity
func (t *Tracker) GetMessageID(oppID int64) (int, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	msgID, exists := t.messages[oppID]
	return msgID, exists
}

// GetStatus retrieves the last known status for a given opportunity
func (t *Tracker) GetStatus(oppID int64) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	status, exists := t.statusMap[oppID]
	return status, exists
}

// UpdateStatus updates the status for a tracked opportunity
func (t *Tracker) UpdateStatus(oppID int64, newStatus string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.statusMap[oppID] = newStatus
	t.save()
}
