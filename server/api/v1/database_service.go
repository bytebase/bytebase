package v1

import (
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/store"
)

// DatabaseService implements the database service.
type DatabaseService struct {
	v1pb.UnimplementedDatabaseServiceServer
	store *store.Store
}

// NewDatabaseService creates a new DatabaseService.
func NewDatabaseService(store *store.Store) *DatabaseService {
	return &DatabaseService{
		store: store,
	}
}
