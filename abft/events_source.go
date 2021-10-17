package abft

import (
	"github.com/nexgencoin/nexgen-base/hash"
	"github.com/nexgencoin/nexgen-base/inter/dag"
)

// EventSource is a callback for getting events from an external storage.
type EventSource interface {
	HasEvent(hash.Event) bool
	GetEvent(hash.Event) dag.Event
}
