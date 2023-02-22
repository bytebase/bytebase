package v1

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// BookmarkService implments the v1pb.BookmarkServiceServer interface.
type BookmarkService struct {
	v1pb.UnimplementedBookmarkServiceServer
	store *store.Store
}

// NewBookmarkService returns a new BookmarkService.
func NewBookmarkService(store *store.Store) *BookmarkService {
	return &BookmarkService{
		store: store,
	}
}

func (s *BookmarkService) CreateBookmark(ctx context.Context, request *v1pb.CreateBookmarkRequest) (*v1pb.Bookmark, error) {
	panic("not implemented")
}

func (s *BookmarkService) ListBookmarks(ctx context.Context, request *v1pb.ListBookmarksRequest) (*v1pb.ListBookmarksResponse, error) {
	panic("not implemented")
}

func (s *BookmarkService) DeleteBookmark(ctx context.Context, request *v1pb.DeleteBookmarkRequest) (*emptypb.Empty, error) {
	panic("not implemented")
}
