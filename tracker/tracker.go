package tracker

import (
	"sync"
)

// Tracker keeps track of the mapped opportunities to telegram message IDs.
type Tracker struct {
	mu        sync.RWMutex
	messages  map[int64]int // Map OpportunityID -> Telegram MessageID
	statusMap map[int64]string // Map OpportunityID -> Status (pending, win, lose)
}

// NewTracker creates a new tracker instance
func NewTracker() *Tracker {
	return &Tracker{
		messages:  make(map[int64]int),
		statusMap: make(map[int64]string),
	}
}

// SaveMessageID saves the telegram message ID for an opportunity
func (t *Tracker) SaveMessageID(oppID int64, msgID int, status string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.messages[oppID] = msgID
	t.statusMap[oppID] = status
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
}
