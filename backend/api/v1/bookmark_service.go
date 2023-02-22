package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"

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

// CreateBookmark creates a new bookmark.
func (s *BookmarkService) CreateBookmark(ctx context.Context, request *v1pb.CreateBookmarkRequest) (*v1pb.Bookmark, error) {
	currentPincipalUID := ctx.Value(common.PrincipalIDContextKey).(int)
	princialUID, err := getUserID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid parent: %v", err)
	}
	if princialUID != currentPincipalUID {
		return nil, status.Errorf(codes.PermissionDenied, "Permission denied")
	}
	if request.Bookmark == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Missing bookmark")
	}

	bookmark, err := s.store.CreateBookmarkV2(ctx, convertToStoreBookmark(request.Bookmark), princialUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create bookmark: %v", err)
	}

	return convertToAPIBookmark(bookmark, princialUID), nil
}

// ListBookmarks lists bookmarks.
func (s *BookmarkService) ListBookmarks(ctx context.Context, request *v1pb.ListBookmarksRequest) (*v1pb.ListBookmarksResponse, error) {
	currentPincipalUID := ctx.Value(common.PrincipalIDContextKey).(int)
	princialUID, err := getUserID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid parent: %v", err)
	}
	if princialUID != currentPincipalUID {
		return nil, status.Errorf(codes.PermissionDenied, "Permission denied")
	}

	bookmarkList, err := s.store.ListBookmarkV2(ctx, princialUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to list bookmarks: %v", err)
	}

	var bookmarks []*v1pb.Bookmark
	for _, bookmark := range bookmarkList {
		bookmarks = append(bookmarks, convertToAPIBookmark(bookmark, princialUID))
	}

	return &v1pb.ListBookmarksResponse{
		Bookmarks: bookmarks,
	}, nil
}

// DeleteBookmark deletes a bookmark.
func (s *BookmarkService) DeleteBookmark(ctx context.Context, request *v1pb.DeleteBookmarkRequest) (*emptypb.Empty, error) {
	currentPincipalUID := ctx.Value(common.PrincipalIDContextKey).(int)
	principalUID, bookmarkUID, err := getUserBookmarkID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid book mark name: %v", err)
	}
	if principalUID != currentPincipalUID {
		return nil, status.Errorf(codes.PermissionDenied, "Permission denied")
	}

	if err := s.store.DeleteBookmarkV2(ctx, bookmarkUID); err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return nil, status.Errorf(codes.NotFound, "Bookmark not found: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "Failed to delete bookmark: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func convertToStoreBookmark(request *v1pb.Bookmark) *store.BookmarkMessage {
	return &store.BookmarkMessage{
		Name: request.Title,
		Link: request.Link,
	}
}

func convertToAPIBookmark(bookmark *store.BookmarkMessage, principalUID int) *v1pb.Bookmark {
	return &v1pb.Bookmark{
		Name:  fmt.Sprintf("%s%d/%s%d", userNamePrefix, principalUID, bookmarkPrefix, bookmark.ID),
		Title: bookmark.Name,
		Link:  bookmark.Link,
	}
}
