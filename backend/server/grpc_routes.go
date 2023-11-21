package server

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/runner/backuprun"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/runner/relay"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func configureGrpcRouters(
	ctx context.Context,
	mux *grpcruntime.ServeMux,
	grpcServer *grpc.Server,
	stores *store.Store,
	dbFactory *dbfactory.DBFactory,
	licenseService enterprise.LicenseService,
	profile *config.Profile,
	metricReporter *metricreport.Reporter,
	stateCfg *state.State,
	schemaSyncer *schemasync.Syncer,
	activityManager *activity.Manager,
	backupRunner *backuprun.Runner,
	relayRunner *relay.Runner,
	planCheckScheduler *plancheck.Scheduler,
	postCreateUser apiv1.CreateUserFunc,
	secret string,
	errorRecordRing *api.ErrorRecordRing,
	tokenDuration time.Duration) (*apiv1.RolloutService, *apiv1.IssueService, error) {
	// Register services.
	authService, err := apiv1.NewAuthService(stores, secret, tokenDuration, licenseService, metricReporter, profile, stateCfg, postCreateUser)
	if err != nil {
		return nil, nil, err
	}
	v1pb.RegisterAuthServiceServer(grpcServer, authService)
	v1pb.RegisterActuatorServiceServer(grpcServer, apiv1.NewActuatorService(stores, profile, errorRecordRing))
	v1pb.RegisterSubscriptionServiceServer(grpcServer, apiv1.NewSubscriptionService(
		stores,
		profile,
		metricReporter,
		licenseService))
	v1pb.RegisterEnvironmentServiceServer(grpcServer, apiv1.NewEnvironmentService(stores, licenseService))
	v1pb.RegisterInstanceServiceServer(grpcServer, apiv1.NewInstanceService(
		stores,
		licenseService,
		metricReporter,
		secret,
		stateCfg,
		dbFactory,
		schemaSyncer))
	v1pb.RegisterProjectServiceServer(grpcServer, apiv1.NewProjectService(stores, activityManager, licenseService))
	v1pb.RegisterDatabaseServiceServer(grpcServer, apiv1.NewDatabaseService(stores, backupRunner, schemaSyncer, licenseService, profile))
	v1pb.RegisterInstanceRoleServiceServer(grpcServer, apiv1.NewInstanceRoleService(stores, dbFactory))
	v1pb.RegisterOrgPolicyServiceServer(grpcServer, apiv1.NewOrgPolicyService(stores, licenseService))
	v1pb.RegisterIdentityProviderServiceServer(grpcServer, apiv1.NewIdentityProviderService(stores, licenseService))
	v1pb.RegisterSettingServiceServer(grpcServer, apiv1.NewSettingService(stores, profile, licenseService, stateCfg))
	v1pb.RegisterAnomalyServiceServer(grpcServer, apiv1.NewAnomalyService(stores))
	v1pb.RegisterSQLServiceServer(grpcServer, apiv1.NewSQLService(stores, schemaSyncer, dbFactory, activityManager, licenseService))
	v1pb.RegisterExternalVersionControlServiceServer(grpcServer, apiv1.NewExternalVersionControlService(stores))
	v1pb.RegisterRiskServiceServer(grpcServer, apiv1.NewRiskService(stores, licenseService))
	issueService := apiv1.NewIssueService(stores, activityManager, relayRunner, stateCfg, licenseService, metricReporter)
	v1pb.RegisterIssueServiceServer(grpcServer, issueService)
	rolloutService := apiv1.NewRolloutService(stores, licenseService, dbFactory, planCheckScheduler, stateCfg, activityManager)
	v1pb.RegisterRolloutServiceServer(grpcServer, rolloutService)
	v1pb.RegisterRoleServiceServer(grpcServer, apiv1.NewRoleService(stores, licenseService))
	v1pb.RegisterSheetServiceServer(grpcServer, apiv1.NewSheetService(stores, licenseService))
	v1pb.RegisterSchemaDesignServiceServer(grpcServer, apiv1.NewSchemaDesignService(stores, licenseService))
	v1pb.RegisterCelServiceServer(grpcServer, apiv1.NewCelService())
	v1pb.RegisterLoggingServiceServer(grpcServer, apiv1.NewLoggingService(stores))
	v1pb.RegisterInboxServiceServer(grpcServer, apiv1.NewInboxService(stores))
	v1pb.RegisterChangelistServiceServer(grpcServer, apiv1.NewChangelistService(stores))

	// REST gateway proxy.
	grpcEndpoint := fmt.Sprintf(":%d", profile.GrpcPort)
	grpcConn, err := grpc.Dial(grpcEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	if err := v1pb.RegisterAuthServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterActuatorServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterSubscriptionServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterEnvironmentServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterInstanceServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterProjectServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterDatabaseServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterInstanceRoleServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterOrgPolicyServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterIdentityProviderServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterSettingServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterAnomalyServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterSQLServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterExternalVersionControlServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterRoleServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterSheetServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterRolloutServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterIssueServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterLoggingServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterInboxServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	if err := v1pb.RegisterChangelistServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, err
	}
	return rolloutService, issueService, nil
}
