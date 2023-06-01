package v1

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ExternalVersionControlService represents a service for managing external version control.
type ExternalVersionControlService struct {
	v1pb.UnimplementedExternalVersionControlServiceServer
	store *store.Store
}

// NewExternalVersionControlService returns a new instance of ExternalVersionControlService.
func NewExternalVersionControlService(store *store.Store) *ExternalVersionControlService {
	return &ExternalVersionControlService{store: store}
}

// GetExternalVersionControl get a single external version control.
func (s *ExternalVersionControlService) GetExternalVersionControl(ctx context.Context, request *v1pb.GetExternalVersionControlRequest) (*v1pb.ExternalVersionControl, error) {
	externalVersionControlUID, err := getExternalVersionControlID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	externalVersionControl, err := s.store.GetExternalVersionControlV2(ctx, externalVersionControlUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve external version control: %v", err)
	}
	if externalVersionControl == nil {
		return nil, status.Errorf(codes.NotFound, "External version control not found: %v", err)
	}

	return convertToExternalVersionControl(externalVersionControl), nil
}

// ListExternalVersionControls lists external version controls.
func (s *ExternalVersionControlService) ListExternalVersionControls(ctx context.Context, _ *v1pb.ListExternalVersionControlsRequest) (*v1pb.ListExternalVersionControlsResponse, error) {
	externalVersionControls, err := s.store.ListExternalVersionControls(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve external version control: %v", err)
	}
	return &v1pb.ListExternalVersionControlsResponse{ExternalVersionControls: convertToExternalVersionControls(externalVersionControls)}, nil
}

// CreateExternalVersionControl creates a new external version control.
func (s *ExternalVersionControlService) CreateExternalVersionControl(ctx context.Context, request *v1pb.CreateExternalVersionControlRequest) (*v1pb.ExternalVersionControl, error) {
	externalVersionControl, err := checkAndConvertToStoreVersionControl(request.ExternalVersionControl)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	storeExternalVersionControl, err := s.store.CreateExternalVersionControlV2(ctx, ctx.Value(common.PrincipalIDContextKey).(int), externalVersionControl)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create external version control: %v", err)
	}
	return convertToExternalVersionControl(storeExternalVersionControl), nil
}

// UpdateExternalVersionControl updates an existing external version control.
func (s *ExternalVersionControlService) UpdateExternalVersionControl(ctx context.Context, request *v1pb.UpdateExternalVersionControlRequest) (*v1pb.ExternalVersionControl, error) {
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}
	externalVersionControlUID, err := getExternalVersionControlID(request.ExternalVersionControl.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	externalVersionControl, err := s.store.GetExternalVersionControlV2(ctx, externalVersionControlUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve external version control: %v", err)
	}
	if externalVersionControl == nil {
		return nil, status.Errorf(codes.NotFound, "External version control not found: %v", err)
	}

	update := &store.UpdateExternalVersionControlMessage{}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			if request.ExternalVersionControl.Title == "" {
				return nil, status.Errorf(codes.InvalidArgument, "title should not be empty")
			}
			update.Name = &request.ExternalVersionControl.Title
		case "application_id":
			if request.ExternalVersionControl.ApplicationId == "" {
				return nil, status.Errorf(codes.InvalidArgument, "application_id should not be empty")
			}
			update.ApplicationID = &request.ExternalVersionControl.ApplicationId
		case "secret":
			if request.ExternalVersionControl.Secret == "" {
				return nil, status.Errorf(codes.InvalidArgument, "secret should not be empty")
			}
			update.Secret = &request.ExternalVersionControl.Secret
		}
	}

	externalVersionControl, err = s.store.UpdateExternalVersionControlV2(ctx, ctx.Value(common.PrincipalIDContextKey).(int), externalVersionControlUID, update)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToExternalVersionControl(externalVersionControl), nil
}

// DeleteExternalVersionControl deletes an existing external version control.
func (s *ExternalVersionControlService) DeleteExternalVersionControl(ctx context.Context, request *v1pb.DeleteExternalVersionControlRequest) (*emptypb.Empty, error) {
	externalVersionControlUID, err := getExternalVersionControlID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	externalVersionControl, err := s.store.GetExternalVersionControlV2(ctx, externalVersionControlUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve external version control: %v", err)
	}
	if externalVersionControl == nil {
		return nil, status.Errorf(codes.NotFound, "External version control not found: %v", err)
	}

	if err := s.store.DeleteExternalVersionControlV2(ctx, externalVersionControlUID); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to delete external version control: %v", err)
	}
	return &emptypb.Empty{}, nil
}

// SearchExternalVersionControlProjects searches external version control projects, for example, GitHub repository.
func (s *ExternalVersionControlService) SearchExternalVersionControlProjects(ctx context.Context, request *v1pb.SearchExternalVersionControlProjectsRequest) (*v1pb.SearchExternalVersionControlProjectsResponse, error) {
	externalVersionControlUID, err := getExternalVersionControlID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	externalVersionControl, err := s.store.GetExternalVersionControlV2(ctx, externalVersionControlUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve external version control: %v", err)
	}
	if externalVersionControl == nil {
		return nil, status.Errorf(codes.NotFound, "External version control not found: %v", err)
	}

	apiExternalProjectList, err := vcs.Get(externalVersionControl.Type, vcs.ProviderConfig{}).FetchAllRepositoryList(
		ctx,
		common.OauthContext{
			ClientID:     externalVersionControl.ApplicationID,
			ClientSecret: externalVersionControl.Secret,
			AccessToken:  request.AccessToken,
			RefreshToken: request.RefreshToken,
			Refresher:    nil,
		},
		externalVersionControl.InstanceURL,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch external project list: %v", err)
	}

	var externalProjects []*v1pb.SearchExternalVersionControlProjectsResponse_Project
	for _, apiExternalProject := range apiExternalProjectList {
		externalProjects = append(externalProjects, &v1pb.SearchExternalVersionControlProjectsResponse_Project{
			Id:       apiExternalProject.ID,
			Title:    apiExternalProject.Name,
			Fullpath: apiExternalProject.FullPath,
			WebUrl:   apiExternalProject.WebURL,
		})
	}

	return &v1pb.SearchExternalVersionControlProjectsResponse{
		Projects: externalProjects,
	}, nil
}

// ListProjectGitOpsInfo lists GitOps info of a project.
func (s *ExternalVersionControlService) ListProjectGitOpsInfo(ctx context.Context, request *v1pb.ListProjectGitOpsInfoRequest) (*v1pb.ListProjectGitOpsInfoResponse, error) {
	externalVersionControlUID, err := getExternalVersionControlID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	repositoryFind := &api.RepositoryFind{
		VCSID: &externalVersionControlUID,
	}
	repoList, err := s.store.FindRepository(ctx, repositoryFind)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch external repository list: %v", err)
	}

	resp := &v1pb.ListProjectGitOpsInfoResponse{}
	for _, repo := range repoList {
		resp.ProjectGitopsInfo = append(resp.ProjectGitopsInfo, convertToProjectGitOpsInfo(fmt.Sprintf("%s%s", projectNamePrefix, repo.Project.ResourceID), repo))
	}

	return resp, nil
}

func convertToExternalVersionControls(externalVersionControls []*store.ExternalVersionControlMessage) []*v1pb.ExternalVersionControl {
	var res []*v1pb.ExternalVersionControl
	for _, externalVersionControl := range externalVersionControls {
		res = append(res, convertToExternalVersionControl(externalVersionControl))
	}
	return res
}

func convertToExternalVersionControl(externalVersionControl *store.ExternalVersionControlMessage) *v1pb.ExternalVersionControl {
	tp := v1pb.ExternalVersionControl_TYPE_UNSPECIFIED
	switch externalVersionControl.Type {
	case vcs.GitHub:
		tp = v1pb.ExternalVersionControl_TYPE_GITHUB
	case vcs.GitLab:
		tp = v1pb.ExternalVersionControl_TYPE_GITLAB
	case vcs.Bitbucket:
		tp = v1pb.ExternalVersionControl_TYPE_BITBUCKET
	}

	return &v1pb.ExternalVersionControl{
		Name:          fmt.Sprintf("%s/%d", externalVersionControlPrefix, externalVersionControl.ID),
		Title:         externalVersionControl.Name,
		Type:          tp,
		Url:           externalVersionControl.InstanceURL,
		ApiUrl:        externalVersionControl.APIURL,
		ApplicationId: externalVersionControl.ApplicationID,
	}
}

func checkAndConvertToStoreVersionControl(externalVersionControl *v1pb.ExternalVersionControl) (*store.ExternalVersionControlMessage, error) {
	if externalVersionControl.Title == "" {
		return nil, errors.Errorf("Empty ExternalVersionControl.Title")
	}
	if externalVersionControl.Type == v1pb.ExternalVersionControl_TYPE_UNSPECIFIED {
		return nil, errors.Errorf("Empty ExternalVersionControl.Type")
	}
	if externalVersionControl.Url == "" {
		return nil, errors.Errorf("Empty ExternalVersionControl.Url")
	}
	if externalVersionControl.ApplicationId == "" {
		return nil, errors.Errorf("Empty ExternalVersionControl.ApplicationId")
	}
	if externalVersionControl.Secret == "" {
		return nil, errors.Errorf("Empty ExternalVersionControl.Secret")
	}
	storeExternalVersionControl := &store.ExternalVersionControlMessage{
		Name:          externalVersionControl.Title,
		ApplicationID: externalVersionControl.ApplicationId,
		Secret:        externalVersionControl.Secret,
	}

	tp, err := convertExternalVersionControlTypeToVCSType(externalVersionControl.Type)
	if err != nil {
		return nil, err
	}

	storeExternalVersionControl.InstanceURL = strings.TrimRight(externalVersionControl.Url, "/")
	storeExternalVersionControl.APIURL = vcs.Get(tp, vcs.ProviderConfig{}).APIURL(externalVersionControl.Url)
	storeExternalVersionControl.Type = tp
	return storeExternalVersionControl, nil
}

func convertExternalVersionControlTypeToVCSType(tp v1pb.ExternalVersionControl_Type) (vcs.Type, error) {
	switch tp {
	case v1pb.ExternalVersionControl_TYPE_GITHUB:
		return vcs.GitHub, nil
	case v1pb.ExternalVersionControl_TYPE_GITLAB:
		return vcs.GitLab, nil
	case v1pb.ExternalVersionControl_TYPE_BITBUCKET:
		return vcs.Bitbucket, nil
	}
	return "", errors.Errorf("unknown external version control type: %v", tp)
}
