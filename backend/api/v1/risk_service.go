package v1

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/enterprise"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// RiskService implements the risk service.
type RiskService struct {
	v1connect.UnimplementedRiskServiceHandler
	store          *store.Store
	iamManager     *iam.Manager
	licenseService *enterprise.LicenseService
}

// NewRiskService creates a new RiskService.
func NewRiskService(store *store.Store, iamManager *iam.Manager, licenseService *enterprise.LicenseService) *RiskService {
	return &RiskService{
		store:          store,
		iamManager:     iamManager,
		licenseService: licenseService,
	}
}

func convertToRisk(risk *store.RiskMessage) *v1pb.Risk {
	return &v1pb.Risk{
		Name:      fmt.Sprintf("%s%v", common.RiskPrefix, risk.ID),
		Source:    ConvertToV1Source(risk.Source),
		Title:     risk.Name,
		Level:     risk.Level,
		Condition: risk.Expression,
		Active:    risk.Active,
	}
}

// ListRisks lists risks.
func (s *RiskService) ListRisks(ctx context.Context, _ *connect.Request[v1pb.ListRisksRequest]) (*connect.Response[v1pb.ListRisksResponse], error) {
	risks, err := s.store.ListRisks(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &v1pb.ListRisksResponse{}
	for _, risk := range risks {
		r := convertToRisk(risk)
		response.Risks = append(response.Risks, r)
	}
	return connect.NewResponse(response), nil
}

// GetRisk gets the risk.
func (s *RiskService) GetRisk(ctx context.Context, request *connect.Request[v1pb.GetRiskRequest]) (*connect.Response[v1pb.Risk], error) {
	risk, err := s.getRiskByName(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	r := convertToRisk(risk)
	return connect.NewResponse(r), nil
}

// CreateRisk creates a risk.
func (s *RiskService) CreateRisk(ctx context.Context, request *connect.Request[v1pb.CreateRiskRequest]) (*connect.Response[v1pb.Risk], error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_RISK_ASSESSMENT); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	// Validate the condition.
	if _, err := common.ConvertUnparsedRisk(request.Msg.Risk.Condition); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to validate risk expression"))
	}

	risk, err := s.store.CreateRisk(ctx, &store.RiskMessage{
		Source:     convertToSource(request.Msg.Risk.Source),
		Level:      request.Msg.Risk.Level,
		Name:       request.Msg.Risk.Title,
		Active:     request.Msg.Risk.Active,
		Expression: request.Msg.Risk.Condition,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	r := convertToRisk(risk)
	return connect.NewResponse(r), nil
}

// UpdateRisk updates a risk.
func (s *RiskService) UpdateRisk(ctx context.Context, request *connect.Request[v1pb.UpdateRiskRequest]) (*connect.Response[v1pb.Risk], error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_RISK_ASSESSMENT); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	if request.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask must be set"))
	}

	riskID, err := common.GetRiskID(request.Msg.Risk.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}

	risk, err := s.store.GetRisk(ctx, riskID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get risk"))
	}
	if risk == nil {
		if request.Msg.AllowMissing {
			ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionRisksCreate, user)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to check permission"))
			}
			if !ok {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionRisksCreate))
			}
			return s.CreateRisk(ctx, connect.NewRequest(&v1pb.CreateRiskRequest{
				Risk: request.Msg.Risk,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("risk %q not found", request.Msg.Risk.Name))
	}

	patch := &store.UpdateRiskMessage{}
	for _, path := range request.Msg.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Name = &request.Msg.Risk.Title
		case "active":
			patch.Active = &request.Msg.Risk.Active
		case "level":
			patch.Level = &request.Msg.Risk.Level
		case "condition":
			if _, err := common.ConvertUnparsedRisk(request.Msg.Risk.Condition); err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to validate risk expression"))
			}
			patch.Expression = request.Msg.Risk.Condition
		case "source":
			source := convertToSource(request.Msg.Risk.Source)
			if risk.Source != source {
				patch.Source = &source
			}
		default:
		}
	}

	risk, err = s.store.UpdateRisk(ctx, patch, risk.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	r := convertToRisk(risk)
	return connect.NewResponse(r), nil
}

// DeleteRisk deletes a risk.
func (s *RiskService) DeleteRisk(ctx context.Context, request *connect.Request[v1pb.DeleteRiskRequest]) (*connect.Response[emptypb.Empty], error) {
	riskID, err := common.GetRiskID(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := s.store.DeleteRisk(ctx, riskID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *RiskService) getRiskByName(ctx context.Context, name string) (*store.RiskMessage, error) {
	riskID, err := common.GetRiskID(name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	risk, err := s.store.GetRisk(ctx, riskID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get risk"))
	}
	if risk == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("risk %v not found", name))
	}
	return risk, nil
}

func ConvertToV1Source(source store.RiskSource) v1pb.Risk_Source {
	switch source {
	case store.RiskSourceDatabaseCreate:
		return v1pb.Risk_CREATE_DATABASE
	case store.RiskSourceDatabaseSchemaUpdate:
		return v1pb.Risk_DDL
	case store.RiskSourceDatabaseDataUpdate:
		return v1pb.Risk_DML
	case store.RiskSourceDatabaseDataExport:
		return v1pb.Risk_DATA_EXPORT
	case store.RiskRequestRole:
		return v1pb.Risk_REQUEST_ROLE
	default:
	}
	return v1pb.Risk_SOURCE_UNSPECIFIED
}

func convertToSource(source v1pb.Risk_Source) store.RiskSource {
	switch source {
	case v1pb.Risk_CREATE_DATABASE:
		return store.RiskSourceDatabaseCreate
	case v1pb.Risk_DDL:
		return store.RiskSourceDatabaseSchemaUpdate
	case v1pb.Risk_DML:
		return store.RiskSourceDatabaseDataUpdate
	case v1pb.Risk_DATA_EXPORT:
		return store.RiskSourceDatabaseDataExport
	case v1pb.Risk_REQUEST_ROLE:
		return store.RiskRequestRole
	default:
	}
	return store.RiskSourceUnknown
}
