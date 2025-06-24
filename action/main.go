package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/google/uuid"

	"github.com/bytebase/bytebase/action/args"
	"github.com/bytebase/bytebase/action/github"
	"github.com/bytebase/bytebase/backend/common/log"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	Config struct {
		// bytebase-action flags
		Output               string
		URL                  string
		ServiceAccount       string
		ServiceAccountSecret string
		Project              string // projects/{project}
		Targets              []string
		FilePattern          string

		// bytebase-action check flags
		// An enum to determine should we fail on warning or error.
		// Valid values:
		// - SKIP
		// - FAIL_ON_WARNING
		// - FAIL_ON_ERROR
		CheckRelease string

		// bytebase-action rollout flags
		ReleaseTitle string // The title of the release
		RolloutTitle string // The title of the rollout
		// An enum to determine should we run plan checks and fail on warning or error.
		// Valid values:
		// - SKIP
		// - FAIL_ON_WARNING
		// - FAIL_ON_ERROR
		CheckPlan string
		// Rollout up to the target-stage.
		// Format: environments/{environment}
		TargetStage string
		Plan        string
	}
	cmd = &cobra.Command{
		Use:                "bytebase-action",
		Short:              "Bytebase action",
		PersistentPreRunE:  validateSharedFlags,
		PersistentPostRunE: writeOutputJSON,
	}
	outputMap = map[string]string{}
)

func init() {
	// bytebase-action flags
	cmd.PersistentFlags().StringVar(&Config.Output, "output", "", "Output file location. The output file is a JSON file with the created resource names")
	cmd.PersistentFlags().StringVar(&Config.URL, "url", "https://demo.bytebase.com", "Bytebase URL")
	cmd.PersistentFlags().StringVar(&Config.ServiceAccount, "service-account", "api@service.bytebase.com", "Bytebase Service account")
	cmd.PersistentFlags().StringVar(&Config.ServiceAccountSecret, "service-account-secret", os.Getenv("BYTEBASE_SERVICE_ACCOUNT_SECRET"), "Bytebase Service account secret")
	cmd.PersistentFlags().StringVar(&Config.Project, "project", "projects/hr", "Bytebase project")
	cmd.PersistentFlags().StringSliceVar(&Config.Targets, "targets", []string{"instances/test-sample-instance/databases/hr_test", "instances/prod-sample-instance/databases/hr_prod"}, "Bytebase targets")
	cmd.PersistentFlags().StringVar(&Config.FilePattern, "file-pattern", "", "File pattern to glob migration files")

	// bytebase-action check flags
	cmdCheck := &cobra.Command{
		Use:               "check",
		Short:             "Check the release files",
		Args:              cobra.NoArgs,
		PersistentPreRunE: validateCheckFlags,
		RunE:              runCheck,
	}
	cmdCheck.Flags().StringVar(&Config.CheckRelease, "check-release", "SKIP", "Whether to fail on warning/error. Valid values: SKIP, FAIL_ON_WARNING, FAIL_ON_ERROR")

	cmd.AddCommand(cmdCheck)

	// bytebase-action rollout flags
	cmdRollout := &cobra.Command{
		Use:               "rollout",
		Short:             "Rollout the migrate files",
		Args:              cobra.NoArgs,
		PersistentPreRunE: validateRolloutFlags,
		RunE:              runRollout,
	}
	defaultTitle := time.Now().Format(time.RFC3339)
	cmdRollout.Flags().StringVar(&Config.ReleaseTitle, "release-title", defaultTitle, "The title of the release")
	cmdRollout.Flags().StringVar(&Config.RolloutTitle, "rollout-title", defaultTitle, "The title of the rollout")
	cmdRollout.Flags().StringVar(&Config.CheckPlan, "check-plan", "SKIP", "Whether to check the plan and fail on warning/error. Valid values: SKIP, FAIL_ON_WARNING, FAIL_ON_ERROR")
	cmdRollout.Flags().StringVar(&Config.TargetStage, "target-stage", "", "Rollout up to the target stage. Format: environments/{environment}.")
	cmdRollout.Flags().StringVar(&Config.Plan, "plan", "", "The plan to rollout. Format: projects/{project}/plans/{plan}. Shadows file-pattern and targets.")
	cmd.AddCommand(cmdRollout)
}

func validateSharedFlags(*cobra.Command, []string) error {
	if Config.ServiceAccount == "" {
		return errors.Errorf("service-account is required and cannot be empty")
	}
	if Config.ServiceAccountSecret == "" {
		return errors.Errorf("service-account-secret is required and cannot be empty")
	}

	// Validate URL format
	u, err := url.Parse(Config.URL)
	if err != nil {
		return errors.Wrapf(err, "invalid URL format: %s", Config.URL)
	}
	Config.URL = strings.TrimSuffix(u.String(), "/") // update the URL to the canonical form

	// Validate project format
	if !strings.HasPrefix(Config.Project, "projects/") {
		return errors.Errorf("invalid project format, must be projects/{project}")
	}

	// Validate targets format
	for _, target := range Config.Targets {
		if !strings.HasPrefix(target, "instances/") || !strings.Contains(target, "/databases/") {
			return errors.Errorf("invalid target format, must be instances/{instance}/databases/{database}: %s", target)
		}
	}

	return nil
}

func validateCheckFlags(*cobra.Command, []string) error {
	switch Config.CheckRelease {
	case "SKIP", "FAIL_ON_WARNING", "FAIL_ON_ERROR":
	default:
		return errors.Errorf("invalid check-release value: %s. Valid values: SKIP, FAIL_ON_WARNING, FAIL_ON_ERROR", Config.CheckRelease)
	}

	return nil
}

func validateRolloutFlags(*cobra.Command, []string) error {
	if Config.TargetStage == "" {
		return errors.Errorf("target-stage is required and cannot be empty")
	}

	switch Config.CheckPlan {
	case "SKIP", "FAIL_ON_WARNING", "FAIL_ON_ERROR":
	default:
		return errors.Errorf("invalid check-plan value: %s. Valid values: SKIP, FAIL_ON_WARNING, FAIL_ON_ERROR", Config.CheckPlan)
	}

	return nil
}

func writeOutputJSON(*cobra.Command, []string) error {
	if Config.Output == "" {
		return nil
	}

	slog.Info("writing output to file", "file", Config.Output)

	// Create parent directory if not exists
	if dir := filepath.Dir(Config.Output); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.Wrapf(err, "failed to create output directory: %s", dir)
		}
	}

	f, err := os.Create(Config.Output)
	if err != nil {
		return errors.Wrapf(err, "failed to create output file: %s", Config.Output)
	}
	defer f.Close()

	j, err := json.Marshal(outputMap)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal output map")
	}

	if _, err := f.Write(j); err != nil {
		return errors.Wrapf(err, "failed to write output file: %s", Config.Output)
	}
	return nil
}

func runCheck(*cobra.Command, []string) error {
	platform := getJobPlatform()
	slog.Info("running on platform", "platform", platform.String())
	client, err := NewClient(Config.URL, Config.ServiceAccount, Config.ServiceAccountSecret)
	if err != nil {
		return err
	}

	// Check version compatibility
	checkVersionCompatibility(client, args.Version)

	releaseFiles, err := getReleaseFiles(Config.FilePattern)
	if err != nil {
		return err
	}
	checkReleaseResponse, err := client.checkRelease(Config.Project, &v1pb.CheckReleaseRequest{
		Release: &v1pb.Release{Files: releaseFiles},
		Targets: Config.Targets,
	})
	if err != nil {
		return err
	}

	slog.Info("check release response", "resultCount", len(checkReleaseResponse.Results))

	// Generate platform-specific outputs
	switch platform {
	case GitHub:
		if err := github.CreateCommentAndAnnotation(checkReleaseResponse); err != nil {
			return err
		}
	case GitLab:
		if err := writeReleaseCheckToCodeQualityJSON(checkReleaseResponse); err != nil {
			return err
		}
	case AzureDevOps:
		if err := loggingReleaseChecks(checkReleaseResponse); err != nil {
			return err
		}
	case Bitbucket:
		if err := createBitbucketReport(checkReleaseResponse); err != nil {
			return err
		}
	}

	// Evaluate check results and return errors based on CheckRelease flag
	if Config.CheckRelease == "SKIP" {
		return nil
	}

	var errorCount, warningCount int
	for _, result := range checkReleaseResponse.Results {
		for _, advice := range result.Advices {
			switch advice.Status {
			case v1pb.Advice_ERROR:
				errorCount++
			case v1pb.Advice_WARNING:
				warningCount++
			}
		}
	}

	if errorCount > 0 {
		return errors.Errorf("found %d error(s) in release check. view on Bytebase", errorCount)
	}
	if warningCount > 0 && Config.CheckRelease == "FAIL_ON_WARNING" {
		return errors.Errorf("found %d warning(s) in release check. view on Bytebase", warningCount)
	}

	return nil
}

func runRollout(command *cobra.Command, _ []string) error {
	ctx := command.Context()
	client, err := NewClient(Config.URL, Config.ServiceAccount, Config.ServiceAccountSecret)
	if err != nil {
		return errors.Wrapf(err, "failed to create client")
	}

	// Check version compatibility
	checkVersionCompatibility(client, args.Version)

	var plan *v1pb.Plan
	if Config.Plan != "" {
		planP, err := client.getPlan(Config.Plan)
		if err != nil {
			return errors.Wrapf(err, "failed to get plan")
		}
		plan = planP
		slog.Info("use the provided plan", "url", fmt.Sprintf("%s/%s", client.url, plan.Name))
	} else {
		releaseFiles, err := getReleaseFiles(Config.FilePattern)
		if err != nil {
			return errors.Wrapf(err, "failed to get release files")
		}
		createReleaseResponse, err := client.createRelease(Config.Project, &v1pb.Release{
			Title:     Config.ReleaseTitle,
			Files:     releaseFiles,
			VcsSource: getVCSSource(),
		})
		if err != nil {
			return errors.Wrapf(err, "failed to create release")
		}
		outputMap["release"] = createReleaseResponse.Name

		planCreated, err := client.createPlan(Config.Project, &v1pb.Plan{
			Title: Config.ReleaseTitle,
			Specs: []*v1pb.Plan_Spec{
				{
					Id: uuid.New().String(),
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
							Targets: Config.Targets,
							Release: createReleaseResponse.Name,
						},
					},
				},
			},
		})
		if err != nil {
			return errors.Wrapf(err, "failed to create plan")
		}
		plan = planCreated
		slog.Info("plan created", "url", fmt.Sprintf("%s/%s", client.url, plan.Name))
	}
	outputMap["plan"] = plan.Name

	if err := runAndWaitForPlanChecks(ctx, client, plan.Name); err != nil {
		return errors.Wrapf(err, "failed to run and wait for plan checks")
	}

	if err := runAndWaitForRollout(ctx, client, plan.Name); err != nil {
		return errors.Wrapf(err, "failed to run and wait for rollout")
	}

	return nil
}

func runAndWaitForPlanChecks(ctx context.Context, client *Client, planName string) error {
	if Config.CheckPlan == "SKIP" {
		return nil
	}
	for {
		if ctx.Err() != nil {
			return errors.Wrapf(ctx.Err(), "context cancelled")
		}

		runs, err := client.listAllPlanCheckRuns(planName)
		if err != nil {
			return errors.Wrapf(err, "failed to list plan checks")
		}
		if len(runs.PlanCheckRuns) == 0 {
			slog.Info("running plan checks")
			_, err := client.runPlanChecks(&v1pb.RunPlanChecksRequest{
				Name: planName,
			})
			if err != nil {
				return errors.Wrapf(err, "failed to run plan checks")
			}
			continue
		}
		var failedCount, canceledCount, runningCount int
		var errorCount, warningCount int
		for _, run := range runs.PlanCheckRuns {
			switch run.Status {
			case v1pb.PlanCheckRun_FAILED:
				failedCount++
			case v1pb.PlanCheckRun_CANCELED:
				canceledCount++
			case v1pb.PlanCheckRun_RUNNING:
				runningCount++
			case v1pb.PlanCheckRun_DONE:
				for _, result := range run.Results {
					switch result.Status {
					case v1pb.PlanCheckRun_Result_ERROR:
						errorCount++
					case v1pb.PlanCheckRun_Result_WARNING:
						warningCount++
					}
				}
			}
		}
		if failedCount > 0 {
			return errors.Errorf("found failed plan checks. view on Bytebase")
		}
		if canceledCount > 0 {
			return errors.Errorf("found canceled plan checks. view on Bytebase")
		}
		if errorCount > 0 {
			return errors.Errorf("found error plan checks. view on Bytebase")
		}
		if warningCount > 0 && Config.CheckPlan == "FAIL_ON_WARNING" {
			return errors.Errorf("found warning plan checks. view on Bytebase")
		}
		if runningCount == 0 {
			break
		}
		slog.Info("waiting for plan checks to complete", "runningCount", runningCount)
		time.Sleep(5 * time.Second)
	}

	return nil
}

func runAndWaitForRollout(ctx context.Context, client *Client, planName string) error {
	// preview rollout with all pending stages
	rolloutPreview, err := client.createRollout(&v1pb.CreateRolloutRequest{
		Parent: Config.Project,
		Rollout: &v1pb.Rollout{
			Plan: planName,
		},
		Target:       nil, // all stages
		ValidateOnly: true,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create rollout")
	}

	pendingStages := []string{}
	for _, stage := range rolloutPreview.Stages {
		pendingStages = append(pendingStages, stage.Environment)
	}

	var stages []string
	var targetStageFound bool
	for _, stage := range rolloutPreview.GetStages() {
		stages = append(stages, stage.Environment)
		if stage.Environment == Config.TargetStage {
			targetStageFound = true
		}
	}
	if !targetStageFound {
		slog.Info("the target stage is not found in the rollout preview. exiting...", "targetStage", Config.TargetStage, "rolloutStages", stages, "hint", "make sure your target-stage input exists in the rollout stages")
		return nil
	}

	// create rollout with no stages to obtain the rollout name
	emptyTarget := ""
	rolloutEmpty, err := client.createRollout(&v1pb.CreateRolloutRequest{
		Parent: Config.Project,
		Rollout: &v1pb.Rollout{
			Plan:  planName,
			Title: Config.RolloutTitle,
		},
		Target:       &emptyTarget, // zero stage
		ValidateOnly: false,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to preview rollout")
	}
	outputMap["rollout"] = rolloutEmpty.Name

	slog.Info("rollout created", "url", fmt.Sprintf("%s/%s", client.url, rolloutEmpty.Name))

	return waitForRollout(ctx, client, pendingStages, rolloutEmpty.Name)
}

func waitForRollout(ctx context.Context, client *Client, pendingStages []string, rolloutName string) error {
	if len(pendingStages) == 0 {
		return nil
	}
	slog.Info("exit after the target stage is completed", "targetStage", Config.TargetStage)
	slog.Info("the rollout has the following pending stages", "stageCount", len(pendingStages), "stages", pendingStages)

	defer func() {
		if ctx.Err() == nil {
			return
		}
		slog.Info("context cancelled, canceling the rollout")
		// cancel rollout
		if err := func() error {
			taskRuns, err := client.listAllTaskRuns(rolloutName)
			if err != nil {
				return errors.Wrapf(err, "failed to list task runs")
			}
			taskRunsToCancelByStage := map[string][]string{}
			for _, taskRun := range taskRuns.TaskRuns {
				if taskRun.Status == v1pb.TaskRun_RUNNING || taskRun.Status == v1pb.TaskRun_PENDING {
					stage := strings.Split(taskRun.Name, "/tasks")[0]
					taskRunsToCancelByStage[stage] = append(taskRunsToCancelByStage[stage], taskRun.Name)
				}
			}
			var errs error
			for stage, taskRuns := range taskRunsToCancelByStage {
				_, err := client.batchCancelTaskRuns(&v1pb.BatchCancelTaskRunsRequest{
					Parent:   stage + "/tasks/-",
					TaskRuns: taskRuns,
				})
				if err != nil {
					err = errors.Wrapf(err, "failed to cancel task runs for stage %v", stage)
					errs = multierr.Append(errs, err)
				}
			}
			return errs
		}(); err != nil {
			slog.Error("failed to cancel rollout", "error", err)
		}
	}()
	i := 0
	for {
		if ctx.Err() != nil {
			return errors.Wrapf(ctx.Err(), "context cancelled")
		}
		// get rollout
		rollout, err := client.getRollout(rolloutName)
		if err != nil {
			return errors.Wrapf(err, "failed to get rollout")
		}
		if i >= len(rollout.GetStages()) {
			if len(pendingStages) == 0 {
				return errors.Errorf("rollout has no more stages")
			}
			// create a new target
			target := pendingStages[0]
			pendingStages = pendingStages[1:]

			rolloutAdvanced, err := client.createRollout(&v1pb.CreateRolloutRequest{
				Parent: Config.Project,
				Rollout: &v1pb.Rollout{
					Plan: rollout.GetPlan(),
				},
				Target: &target,
			})
			if err != nil {
				return errors.Wrapf(err, "failed to create rollout")
			}
			rollout = rolloutAdvanced
		}
		if i >= len(rollout.GetStages()) {
			return errors.Errorf("rollout has no more stages")
		}
		// check stage tasks
		stage := rollout.Stages[i]
		done := true
		var foundFailed, foundCanceled bool
		var notStartedTasks []string
		for _, task := range stage.GetTasks() {
			switch task.Status {
			case v1pb.Task_STATUS_UNSPECIFIED:
				done = false
			case v1pb.Task_NOT_STARTED:
				notStartedTasks = append(notStartedTasks, task.Name)
				done = false
			case v1pb.Task_PENDING:
				done = false
			case v1pb.Task_RUNNING:
				done = false
			case v1pb.Task_FAILED:
				foundFailed = true
				done = false
			case v1pb.Task_CANCELED:
				foundCanceled = true
				done = false
			case v1pb.Task_DONE:
			case v1pb.Task_SKIPPED:
			}
		}
		if foundFailed {
			return errors.Errorf("found failed tasks. view on Bytebase")
		}
		if foundCanceled {
			return errors.Errorf("found canceled tasks. view on Bytebase")
		}
		if done {
			slog.Info("stage completed", "stage", stage.Environment)
			if Config.TargetStage == stage.Environment {
				break
			}
			i++
			continue
		}

		// run stage tasks
		if len(notStartedTasks) > 0 {
			slog.Info("running stage tasks", "stage", stage.Environment, "taskCount", len(notStartedTasks))
			if _, err := client.batchRunTasks(&v1pb.BatchRunTasksRequest{
				Parent: stage.Name,
				Tasks:  notStartedTasks,
			}); err != nil {
				// ignore retryable error.
				if !strings.Contains(err.Error(), "cannot create pending task runs because there are pending/running/done task runs") {
					return errors.Wrapf(err, "failed to batch create tasks")
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

func checkVersionCompatibility(client *Client, cliVersion string) {
	if cliVersion == "unknown" {
		slog.Warn("CLI version unknown, unable to check compatibility")
		return
	}

	actuatorInfo, err := client.getActuatorInfo()
	if err != nil {
		slog.Warn("Unable to get server version for compatibility check", "error", err)
		return
	}

	serverVersion := actuatorInfo.Version
	if serverVersion == "" {
		slog.Warn("Server version is empty, unable to check compatibility")
		return
	}

	if cliVersion == "latest" {
		slog.Warn("Using 'latest' CLI version. It is recommended to use a specific version like bytebase-action:" + serverVersion + " to match your Bytebase server version " + serverVersion)
		return
	}

	if cliVersion != serverVersion {
		slog.Warn("CLI version mismatch", "cliVersion", cliVersion, "serverVersion", serverVersion, "recommendation", "use bytebase-action:"+serverVersion+" to match your Bytebase server")
	} else {
		slog.Info("CLI version matches server version", "version", cliVersion)
	}
}

func main() {
	slog.Info("bytebase-action version " + args.Version + " built at commit " + args.Gitcommit)
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	// Trigger graceful shutdown on SIGINT or SIGTERM.
	// The default signal sent by the `kill` command is SIGTERM,
	// which is taken as the graceful shutdown signal for many systems, eg., Kubernetes, Gunicorn.
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		slog.Info(fmt.Sprintf("%s received.", sig.String()))
		cancel()
	}()

	if err := cmd.ExecuteContext(ctx); err != nil {
		slog.Error("failed to execute command", log.BBError(err))
		os.Exit(1)
	}
}

func getVCSSource() *v1pb.Release_VCSSource {
	switch getJobPlatform() {
	case GitHub:
		return &v1pb.Release_VCSSource{
			VcsType: v1pb.VCSType_GITHUB,
			Url:     os.Getenv("GITHUB_SERVER_URL") + "/" + os.Getenv("GITHUB_REPOSITORY") + "/commit/" + os.Getenv("GITHUB_SHA"),
		}
	case GitLab:
		return &v1pb.Release_VCSSource{
			VcsType: v1pb.VCSType_GITLAB,
			Url:     os.Getenv("CI_PROJECT_URL") + "/-/commit/" + os.Getenv("CI_COMMIT_SHA"),
		}
	case Bitbucket:
		return &v1pb.Release_VCSSource{
			VcsType: v1pb.VCSType_BITBUCKET,
			Url:     os.Getenv("BITBUCKET_GIT_HTTP_ORIGIN") + "/commits/" + os.Getenv("BITBUCKET_COMMIT"),
		}
	case AzureDevOps:
		return &v1pb.Release_VCSSource{
			VcsType: v1pb.VCSType_AZURE_DEVOPS,
			Url:     os.Getenv("SYSTEM_COLLECTIONURI") + os.Getenv("SYSTEM_TEAMPROJECT") + "/_git/" + os.Getenv("BUILD_REPOSITORY_NAME") + "/commit/" + os.Getenv("BUILD_SOURCEVERSION"),
		}
	default:
		return nil
	}
}
