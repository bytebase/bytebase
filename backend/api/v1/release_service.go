package v1

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type ReleaseService struct {
	v1pb.UnimplementedReleaseServiceServer
	store *store.Store
}

func NewReleaseService(store *store.Store) *ReleaseService {
	return &ReleaseService{
		store: store,
	}
}

func (s *ReleaseService) CreateRelease(ctx context.Context, request *v1pb.CreateReleaseRequest) (*v1pb.Release, error) {
	if request.Release == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request.Release cannot be nil")
	}

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}

	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get project id, err: %v", err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find project, err: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %v not found", projectID)
	}

	releaseMessage := &store.ReleaseMessage{
		ProjectUID: project.UID,
		Payload: &storepb.ReleasePayload{
			Title:     request.Release.Title,
			Files:     convertReleaseFiles(request.Release.Files),
			VcsSource: convertReleaseVcsSource(request.Release.VcsSource),
		},
	}

	release, err := s.store.CreateRelease(ctx, releaseMessage, user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create release, err: %v", err)
	}

	converted, err := convertToRelease(ctx, s.store, release)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert release, err: %v", err)
	}

	return converted, nil
}

func (s *ReleaseService) ListReleases(ctx context.Context, request *v1pb.ListReleasesRequest) (*v1pb.ListReleasesResponse, error) {
	if request.PageSize < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "page size must be non-negative: %d", request.PageSize)
	}

	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get project id, err: %v", err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find project, err: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %v not found", projectID)
	}

	limit, offset, err := parseLimitAndOffset(request.PageToken, int(request.PageSize))
	if err != nil {
		return nil, err
	}
	limitPlusOne := limit + 1

	releaseFind := &store.FindReleaseMessage{
		ProjectUID: project.UID,
		Limit:      &limitPlusOne,
		Offset:     &offset,
	}

	releaseMessages, err := s.store.ListReleases(ctx, releaseFind)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list releases, err: %v", err)
	}

	var nextPageToken string
	if len(releaseMessages) == limitPlusOne {
		pageToken, err := getPageToken(limit, offset+limit)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
		nextPageToken = pageToken
		releaseMessages = releaseMessages[:limit]
	}

	releases, err := convertToReleases(ctx, s.store, releaseMessages)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to release, err: %v", err)
	}

	return &v1pb.ListReleasesResponse{
		Releases:      releases,
		NextPageToken: nextPageToken,
	}, nil
}

func (*ReleaseService) UpdateRelease(_ context.Context, _ *v1pb.UpdateReleaseRequest) (*v1pb.Release, error) {
	// TODO(p0ny): implement me please.
	return nil, status.Errorf(codes.Unimplemented, "implement me")
}

func convertToReleases(ctx context.Context, s *store.Store, releases []*store.ReleaseMessage) ([]*v1pb.Release, error) {
	var rs []*v1pb.Release
	for _, release := range releases {
		r, err := convertToRelease(ctx, s, release)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert to release")
		}
		rs = append(rs, r)
	}
	return rs, nil
}

func convertToRelease(ctx context.Context, s *store.Store, release *store.ReleaseMessage) (*v1pb.Release, error) {
	r := &v1pb.Release{
		Title:      release.Payload.Title,
		CreateTime: timestamppb.New(release.CreatedTs),
		Files:      convertToReleaseFiles(release.Payload.Files),
		VcsSource:  convertToReleaseVcsSource(release.Payload.VcsSource),
	}

	project, err := s.GetProjectV2(ctx, &store.FindProjectMessage{UID: &release.ProjectUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find project")
	}
	if project == nil {
		return nil, errors.Wrapf(err, "project %v not found", release.ProjectUID)
	}
	r.Name = common.FormatReleaseName(project.ResourceID, release.UID)

	creator, err := s.GetUserByID(ctx, release.CreatorUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get release creator")
	}
	r.Creator = common.FormatUserEmail(creator.Email)

	return r, nil
}

func convertToReleaseFiles(files []*storepb.ReleasePayload_File) []*v1pb.Release_File {
	var rFiles []*v1pb.Release_File
	for _, f := range files {
		rFiles = append(rFiles, &v1pb.Release_File{
			Name:      f.Name,
			Sheet:     f.Sheet,
			SheetSha1: f.SheetSha1,
			Type:      v1pb.ReleaseFileType(f.Type),
			Version:   f.Version,
		})
	}
	return rFiles
}

func convertToReleaseVcsSource(vs *storepb.ReleasePayload_VCSSource) *v1pb.Release_VCSSource {
	return &v1pb.Release_VCSSource{
		VcsType:        v1pb.VCSType(vs.VcsType),
		PullRequestUrl: vs.PullRequestUrl,
	}
}

func convertReleaseFiles(files []*v1pb.Release_File) []*storepb.ReleasePayload_File {
	var rFiles []*storepb.ReleasePayload_File
	for _, f := range files {
		rFiles = append(rFiles, &storepb.ReleasePayload_File{
			Name:      f.Name,
			Sheet:     f.Sheet,
			SheetSha1: f.SheetSha1,
			Type:      storepb.ReleaseFileType(f.Type),
			Version:   f.Version,
		})
	}
	return rFiles
}

func convertReleaseVcsSource(vs *v1pb.Release_VCSSource) *storepb.ReleasePayload_VCSSource {
	return &storepb.ReleasePayload_VCSSource{
		VcsType:        storepb.VCSType(vs.VcsType),
		PullRequestUrl: vs.PullRequestUrl,
	}
}
