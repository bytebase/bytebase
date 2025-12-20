package v1

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func convertToPlans(ctx context.Context, s *store.Store, plans []*store.PlanMessage) ([]*v1pb.Plan, error) {
	v1Plans := make([]*v1pb.Plan, len(plans))
	for i := range plans {
		p, err := convertToPlan(ctx, s, plans[i])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert plan")
		}
		v1Plans[i] = p
	}
	return v1Plans, nil
}

func convertToPlan(ctx context.Context, s *store.Store, plan *store.PlanMessage) (*v1pb.Plan, error) {
	p := &v1pb.Plan{
		Name:                    common.FormatPlan(plan.ProjectID, plan.UID),
		Title:                   plan.Name,
		Description:             plan.Description,
		Specs:                   convertToPlanSpecs(plan.ProjectID, plan.Config.Specs), // Use specs field for output
		Deployment:              convertToPlanDeployment(plan.Config.Deployment),
		CreateTime:              timestamppb.New(plan.CreatedAt),
		UpdateTime:              timestamppb.New(plan.UpdatedAt),
		State:                   convertDeletedToState(plan.Deleted),
		PlanCheckRunStatusCount: map[string]int32{},
	}

	creator, err := s.GetUserByEmail(ctx, plan.Creator)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get plan creator")
	}
	p.Creator = common.FormatUserEmail(creator.Email)

	issue, err := s.GetIssue(ctx, &store.FindIssueMessage{PlanUID: &plan.UID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get issue by plan uid %d", plan.UID)
	}
	if issue != nil {
		p.Issue = common.FormatIssue(issue.Project.ResourceID, issue.UID)
	}
	if plan.PipelineUID != nil {
		p.Rollout = common.FormatRollout(plan.ProjectID, *plan.PipelineUID)
	}
	planCheckRuns, err := s.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
		PlanUID: &plan.UID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list plan check runs for plan uid %d", plan.UID)
	}
	for _, run := range planCheckRuns {
		p.PlanCheckRunStatusCount[string(run.Status)]++
		for _, result := range run.Result.Results {
			p.PlanCheckRunStatusCount[storepb.Advice_Status_name[int32(result.Status)]]++
		}
	}
	return p, nil
}

func convertToPlanSpecs(projectID string, specs []*storepb.PlanConfig_Spec) []*v1pb.Plan_Spec {
	v1Specs := make([]*v1pb.Plan_Spec, len(specs))
	for i := range specs {
		v1Specs[i] = convertToPlanSpec(projectID, specs[i])
	}
	return v1Specs
}

func convertToPlanSpec(projectID string, spec *storepb.PlanConfig_Spec) *v1pb.Plan_Spec {
	v1Spec := &v1pb.Plan_Spec{
		Id: spec.Id,
	}

	switch v := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		v1Spec.Config = convertToPlanSpecCreateDatabaseConfig(v)
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		v1Spec.Config = convertToPlanSpecChangeDatabaseConfig(projectID, v)
	case *storepb.PlanConfig_Spec_ExportDataConfig:
		v1Spec.Config = convertToPlanSpecExportDataConfig(projectID, v)
	}

	return v1Spec
}

func convertToPlanSpecCreateDatabaseConfig(config *storepb.PlanConfig_Spec_CreateDatabaseConfig) *v1pb.Plan_Spec_CreateDatabaseConfig {
	c := config.CreateDatabaseConfig
	return &v1pb.Plan_Spec_CreateDatabaseConfig{
		CreateDatabaseConfig: &v1pb.Plan_CreateDatabaseConfig{
			Target:       c.Target,
			Database:     c.Database,
			Table:        c.Table,
			CharacterSet: c.CharacterSet,
			Collation:    c.Collation,
			Cluster:      c.Cluster,
			Owner:        c.Owner,
			Environment:  c.Environment,
		},
	}
}

func convertToPlanSpecChangeDatabaseConfig(projectID string, config *storepb.PlanConfig_Spec_ChangeDatabaseConfig) *v1pb.Plan_Spec_ChangeDatabaseConfig {
	c := config.ChangeDatabaseConfig
	return &v1pb.Plan_Spec_ChangeDatabaseConfig{
		ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
			Targets:           c.Targets,
			Sheet:             common.FormatSheet(projectID, c.SheetSha256),
			Release:           c.Release,
			Type:              convertToPlanSpecChangeDatabaseConfigType(c.Type),
			GhostFlags:        c.GhostFlags,
			EnablePriorBackup: c.EnablePriorBackup,
			EnableGhost:       c.EnableGhost,
		},
	}
}

func convertToPlanSpecChangeDatabaseConfigType(t storepb.PlanConfig_ChangeDatabaseConfig_Type) v1pb.DatabaseChangeType {
	switch t {
	case storepb.PlanConfig_ChangeDatabaseConfig_TYPE_UNSPECIFIED:
		return v1pb.DatabaseChangeType_DATABASE_CHANGE_TYPE_UNSPECIFIED
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
		return v1pb.DatabaseChangeType_MIGRATE
	case storepb.PlanConfig_ChangeDatabaseConfig_SDL:
		return v1pb.DatabaseChangeType_SDL
	default:
		return v1pb.DatabaseChangeType_DATABASE_CHANGE_TYPE_UNSPECIFIED
	}
}

func convertToPlanSpecExportDataConfig(projectID string, config *storepb.PlanConfig_Spec_ExportDataConfig) *v1pb.Plan_Spec_ExportDataConfig {
	c := config.ExportDataConfig
	return &v1pb.Plan_Spec_ExportDataConfig{
		ExportDataConfig: &v1pb.Plan_ExportDataConfig{
			Targets:  c.Targets,
			Sheet:    common.FormatSheet(projectID, c.SheetSha256),
			Format:   convertExportFormat(c.Format),
			Password: c.Password,
		},
	}
}

func convertToPlanDeployment(deployment *storepb.PlanConfig_Deployment) *v1pb.Plan_Deployment {
	if deployment == nil {
		return nil
	}
	return &v1pb.Plan_Deployment{
		Environments:          deployment.Environments,
		DatabaseGroupMappings: convertToDatabaseGroupMappings(deployment.DatabaseGroupMappings),
	}
}

func convertToDatabaseGroupMappings(mappings []*storepb.PlanConfig_Deployment_DatabaseGroupMapping) []*v1pb.Plan_Deployment_DatabaseGroupMapping {
	v1Mappings := make([]*v1pb.Plan_Deployment_DatabaseGroupMapping, len(mappings))
	for i := range mappings {
		v1Mappings[i] = convertToDatabaseGroupMapping(mappings[i])
	}
	return v1Mappings
}

func convertToDatabaseGroupMapping(mapping *storepb.PlanConfig_Deployment_DatabaseGroupMapping) *v1pb.Plan_Deployment_DatabaseGroupMapping {
	if mapping == nil {
		return nil
	}
	return &v1pb.Plan_Deployment_DatabaseGroupMapping{
		DatabaseGroup: mapping.DatabaseGroup,
		Databases:     mapping.Databases,
	}
}

func convertPlan(plan *v1pb.Plan) *storepb.PlanConfig {
	if plan == nil {
		return nil
	}

	// At this point, plan.Specs should always be populated
	// (either originally or converted from steps at API entry point)
	return &storepb.PlanConfig{
		Specs:      convertPlanSpecs(plan.Specs),
		Deployment: nil,
	}
}

func convertPlanSpecs(specs []*v1pb.Plan_Spec) []*storepb.PlanConfig_Spec {
	storeSpecs := make([]*storepb.PlanConfig_Spec, len(specs))
	for i := range specs {
		storeSpecs[i] = convertPlanSpec(specs[i])
	}
	return storeSpecs
}

func convertPlanSpec(spec *v1pb.Plan_Spec) *storepb.PlanConfig_Spec {
	storeSpec := &storepb.PlanConfig_Spec{
		Id: spec.Id,
	}

	switch v := spec.Config.(type) {
	case *v1pb.Plan_Spec_CreateDatabaseConfig:
		storeSpec.Config = convertPlanSpecCreateDatabaseConfig(v)
	case *v1pb.Plan_Spec_ChangeDatabaseConfig:
		storeSpec.Config = convertPlanSpecChangeDatabaseConfig(v)
	case *v1pb.Plan_Spec_ExportDataConfig:
		storeSpec.Config = convertPlanSpecExportDataConfig(v)
	}
	return storeSpec
}

func convertPlanSpecCreateDatabaseConfig(config *v1pb.Plan_Spec_CreateDatabaseConfig) *storepb.PlanConfig_Spec_CreateDatabaseConfig {
	c := config.CreateDatabaseConfig
	return &storepb.PlanConfig_Spec_CreateDatabaseConfig{
		CreateDatabaseConfig: convertPlanConfigCreateDatabaseConfig(c),
	}
}

func convertPlanConfigCreateDatabaseConfig(c *v1pb.Plan_CreateDatabaseConfig) *storepb.PlanConfig_CreateDatabaseConfig {
	return &storepb.PlanConfig_CreateDatabaseConfig{
		Target:       c.Target,
		Database:     c.Database,
		Table:        c.Table,
		CharacterSet: c.CharacterSet,
		Collation:    c.Collation,
		Cluster:      c.Cluster,
		Owner:        c.Owner,
		Environment:  c.Environment,
	}
}

func convertPlanSpecChangeDatabaseConfig(config *v1pb.Plan_Spec_ChangeDatabaseConfig) *storepb.PlanConfig_Spec_ChangeDatabaseConfig {
	c := config.ChangeDatabaseConfig

	// Convert v1 DatabaseChangeType to store Type
	var storeType storepb.PlanConfig_ChangeDatabaseConfig_Type
	switch c.Type {
	case v1pb.DatabaseChangeType_MIGRATE:
		storeType = storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE
	case v1pb.DatabaseChangeType_SDL:
		storeType = storepb.PlanConfig_ChangeDatabaseConfig_SDL
	default:
		storeType = storepb.PlanConfig_ChangeDatabaseConfig_TYPE_UNSPECIFIED
	}

	// Sheet can be empty when using Release-based workflow (SQL comes from release files).
	// Plans can use either Sheet-based or Release-based approach, but not both.
	var sheetSha256 string
	if c.Sheet != "" {
		_, sha256, err := common.GetProjectResourceIDSheetSha256(c.Sheet)
		if err != nil {
			return nil
		}
		sheetSha256 = sha256
	}
	return &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
		ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
			Targets:           c.Targets,
			SheetSha256:       sheetSha256,
			Release:           c.Release,
			Type:              storeType,
			GhostFlags:        c.GhostFlags,
			EnablePriorBackup: c.EnablePriorBackup,
			EnableGhost:       c.EnableGhost,
		},
	}
}

func convertPlanSpecExportDataConfig(config *v1pb.Plan_Spec_ExportDataConfig) *storepb.PlanConfig_Spec_ExportDataConfig {
	c := config.ExportDataConfig
	// Sheet can be empty if not yet attached to the export data config.
	var sheetSha256 string
	if c.Sheet != "" {
		_, sha256, err := common.GetProjectResourceIDSheetSha256(c.Sheet)
		if err != nil {
			return nil
		}
		sheetSha256 = sha256
	}
	return &storepb.PlanConfig_Spec_ExportDataConfig{
		ExportDataConfig: &storepb.PlanConfig_ExportDataConfig{
			Targets:     c.Targets,
			SheetSha256: sheetSha256,
			Format:      convertToExportFormat(c.Format),
			Password:    c.Password,
		},
	}
}

func convertToPlanCheckRuns(ctx context.Context, s *store.Store, projectID string, planUID int64, runs []*store.PlanCheckRunMessage) ([]*v1pb.PlanCheckRun, error) {
	var planCheckRuns []*v1pb.PlanCheckRun
	for _, run := range runs {
		converted, err := convertToPlanCheckRun(ctx, s, projectID, planUID, run)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert plan check run")
		}
		planCheckRuns = append(planCheckRuns, converted)
	}
	return planCheckRuns, nil
}

func convertToPlanCheckRun(ctx context.Context, s *store.Store, projectID string, planUID int64, run *store.PlanCheckRunMessage) (*v1pb.PlanCheckRun, error) {
	converted := &v1pb.PlanCheckRun{
		Name:       common.FormatPlanCheckRun(projectID, planUID, int64(run.UID)),
		CreateTime: timestamppb.New(run.CreatedAt),
		Type:       convertToPlanCheckRunType(run.Type),
		Status:     convertToPlanCheckRunStatus(run.Status),
		Target:     common.FormatDatabase(run.Config.InstanceId, run.Config.DatabaseName),
		Sheet:      "",
		Results:    convertToPlanCheckRunResults(run.Result.Results),
		Error:      run.Result.Error,
	}

	if sheetSha256 := run.Config.GetSheetSha256(); sheetSha256 != "" {
		_, err := s.GetSheetMetadata(ctx, sheetSha256)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet")
		}
		// Sheets are now project-agnostic, no need to check projectID
		converted.Sheet = common.FormatSheet(projectID, sheetSha256)
	}

	return converted, nil
}

func convertToPlanCheckRunType(t store.PlanCheckRunType) v1pb.PlanCheckRun_Type {
	switch t {
	case store.PlanCheckDatabaseStatementFakeAdvise:
		return v1pb.PlanCheckRun_DATABASE_STATEMENT_FAKE_ADVISE
	case store.PlanCheckDatabaseStatementAdvise:
		return v1pb.PlanCheckRun_DATABASE_STATEMENT_ADVISE
	case store.PlanCheckDatabaseStatementSummaryReport:
		return v1pb.PlanCheckRun_DATABASE_STATEMENT_SUMMARY_REPORT
	case store.PlanCheckDatabaseConnect:
		return v1pb.PlanCheckRun_DATABASE_CONNECT
	case store.PlanCheckDatabaseGhostSync:
		return v1pb.PlanCheckRun_DATABASE_GHOST_SYNC
	default:
		return v1pb.PlanCheckRun_TYPE_UNSPECIFIED
	}
}

func convertToPlanCheckRunStatus(status store.PlanCheckRunStatus) v1pb.PlanCheckRun_Status {
	switch status {
	case store.PlanCheckRunStatusCanceled:
		return v1pb.PlanCheckRun_CANCELED
	case store.PlanCheckRunStatusDone:
		return v1pb.PlanCheckRun_DONE
	case store.PlanCheckRunStatusFailed:
		return v1pb.PlanCheckRun_FAILED
	case store.PlanCheckRunStatusRunning:
		return v1pb.PlanCheckRun_RUNNING
	default:
		return v1pb.PlanCheckRun_STATUS_UNSPECIFIED
	}
}

func convertToPlanCheckRunResults(results []*storepb.PlanCheckRunResult_Result) []*v1pb.PlanCheckRun_Result {
	var resultsV1 []*v1pb.PlanCheckRun_Result
	for _, result := range results {
		resultsV1 = append(resultsV1, convertToPlanCheckRunResult(result))
	}
	return resultsV1
}

func convertToPlanCheckRunResult(result *storepb.PlanCheckRunResult_Result) *v1pb.PlanCheckRun_Result {
	resultV1 := &v1pb.PlanCheckRun_Result{
		Status:  convertToPlanCheckRunResultStatus(result.Status),
		Title:   result.Title,
		Content: result.Content,
		Code:    result.Code,
		Report:  nil,
	}
	switch report := result.Report.(type) {
	case *storepb.PlanCheckRunResult_Result_SqlSummaryReport_:
		resultV1.Report = &v1pb.PlanCheckRun_Result_SqlSummaryReport_{
			SqlSummaryReport: &v1pb.PlanCheckRun_Result_SqlSummaryReport{
				StatementTypes: report.SqlSummaryReport.StatementTypes,
				AffectedRows:   report.SqlSummaryReport.AffectedRows,
			},
		}
	case *storepb.PlanCheckRunResult_Result_SqlReviewReport_:
		resultV1.Report = &v1pb.PlanCheckRun_Result_SqlReviewReport_{
			SqlReviewReport: &v1pb.PlanCheckRun_Result_SqlReviewReport{
				StartPosition: convertToPosition(report.SqlReviewReport.StartPosition),
				EndPosition:   convertToPosition(report.SqlReviewReport.EndPosition),
			},
		}
	}
	return resultV1
}

func convertToPlanCheckRunResultStatus(status storepb.Advice_Status) v1pb.Advice_Level {
	switch status {
	case storepb.Advice_STATUS_UNSPECIFIED:
		return v1pb.Advice_ADVICE_LEVEL_UNSPECIFIED
	case storepb.Advice_SUCCESS:
		return v1pb.Advice_SUCCESS
	case storepb.Advice_WARNING:
		return v1pb.Advice_WARNING
	case storepb.Advice_ERROR:
		return v1pb.Advice_ERROR
	default:
		return v1pb.Advice_ADVICE_LEVEL_UNSPECIFIED
	}
}
