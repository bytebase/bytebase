// Package state contains the synchronization state shared within the server.
package state

import (
	"sync"

	"github.com/bytebase/bytebase/api"
)

var (
	// InstanceDatabaseSyncChan is the channel for synchronizing schemas for instances.
	InstanceDatabaseSyncChan = make(chan *api.Instance, 100)
	// RollbackGenerateMap is the set of tasks for generating rollback statements.
	RollbackGenerateMap sync.Map
)
