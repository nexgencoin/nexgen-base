package tdag

import (
	"github.com/nexgencoin/nexgen-base/hash"
	"github.com/nexgencoin/nexgen-base/inter/dag"
)

type TestEvent struct {
	dag.MutableBaseEvent
	Name string
}

func (e *TestEvent) AddParent(id hash.Event) {
	parents := e.Parents()
	parents.Add(id)
	e.SetParents(parents)
}
