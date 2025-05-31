package treb

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// GetEventID returns the event signature hash for a given event name
// This is a helper method that works alongside the generated ABI bindings
func (treb *Treb) GetEventID(eventName string) (common.Hash, error) {
	event, exists := treb.abi.Events[eventName]
	if !exists {
		return common.Hash{}, fmt.Errorf("event %s not found", eventName)
	}
	return event.ID, nil
}