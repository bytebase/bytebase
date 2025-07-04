package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/bytebase/bytebase/action/args"
	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func NewRolloutCommand(w *world.World) *cobra.Command {
	// bytebase-action rollout flags
	cmdRollout := &cobra.Command{
		Use:               "rollout",
		Short:             "Rollout the migrate files",
		Args:              cobra.NoArgs,
		PersistentPreRunE: validateRolloutFlags(w),
		RunE:              runRollout(w),
	}
	defaultTitle := time.Now().Format(time.RFC3339)
	cmdRollout.Flags().StringVar(&w.ReleaseTitle, "release-title", defaultTitle, "The title of the release")
	cmdRollout.Flags().StringVar(&w.RolloutTitle, "rollout-title", defaultTitle, "The title of the rollout")
	cmdRollout.Flags().StringVar(&w.CheckPlan, "check-plan", "SKIP", "Whether to check the plan and fail on warning/error. Valid values: SKIP, FAIL_ON_WARNING, FAIL_ON_ERROR")
	cmdRollout.Flags().StringVar(&w.TargetStage, "target-stage", "", "Rollout up to the target stage. Format: environments/{environment}.")
	cmdRollout.Flags().StringVar(&w.Plan, "plan", "", "The plan to rollout. Format: projects/{project}/plans/{plan}. Shadows file-pattern and targets.")
	return cmdRollout
}

func validateRolloutFlags(w *world.World) func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
		switch w.CheckPlan {
		case "SKIP", "FAIL_ON_WARNING", "FAIL_ON_ERROR":
		default:
			return errors.Errorf("invalid check-plan value: %s. Valid values: SKIP, FAIL_ON_WARNING, FAIL_ON_ERROR", w.CheckPlan)
		}
		return nil
	}
}

func runRollout(w *world.World) func(command *cobra.Command, _ []string) error {
	return func(command *cobra.Command, _ []string) error {
		ctx := command.Context()
		client, err := NewClient(w.URL, w.ServiceAccount, w.ServiceAccountSecret)
		if err != nil {
			return errors.Wrapf(err, "failed to create client")
		}

		// Check version compatibility
		checkVersionCompatibility(client, args.Version)

		var plan *v1pb.Plan
		if w.Plan != "" {
			planP, err := client.getPlan(w.Plan)
			if err != nil {
				return errors.Wrapf(err, "failed to get plan")
			}
			plan = planP
			slog.Info("use the provided plan", "url", fmt.Sprintf("%s/%s", client.url, plan.Name))
		} else {
			releaseFiles, err := getReleaseFiles(w.FilePattern)
			if err != nil {
				return errors.Wrapf(err, "failed to get release files")
			}
			createReleaseResponse, err := client.createRelease(w.Project, &v1pb.Release{
				Title:     w.ReleaseTitle,
				Files:     releaseFiles,
				VcsSource: getVCSSource(),
			})
			if err != nil {
				return errors.Wrapf(err, "failed to create release")
			}
			w.OutputMap["release"] = createReleaseResponse.Name

			planCreated, err := client.createPlan(w.Project, &v1pb.Plan{
				Title: w.ReleaseTitle,
				Specs: []*v1pb.Plan_Spec{
					{
						Id: uuid.New().String(),
						Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
							ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
								Targets: w.Targets,
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
		w.OutputMap["plan"] = plan.Name

		if err := runAndWaitForPlanChecks(ctx, w, client, plan.Name); err != nil {
			return errors.Wrapf(err, "failed to run and wait for plan checks")
		}

		if err := runAndWaitForRollout(ctx, w, client, plan.Name); err != nil {
			return errors.Wrapf(err, "failed to run and wait for rollout")
		}

		return nil
	}
}
func runAndWaitForPlanChecks(ctx context.Context, w *world.World, client *Client, planName string) error {
	if w.CheckPlan == "SKIP" {
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
		if warningCount > 0 && w.CheckPlan == "FAIL_ON_WARNING" {
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

func runAndWaitForRollout(ctx context.Context, w *world.World, client *Client, planName string) error {
	// preview rollout with all pending stages
	rolloutPreview, err := client.createRollout(&v1pb.CreateRolloutRequest{
		Parent: w.Project,
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

	// create rollout with no stages to obtain the rollout name
	emptyTarget := ""
	rolloutEmpty, err := client.createRollout(&v1pb.CreateRolloutRequest{
		Parent: w.Project,
		Rollout: &v1pb.Rollout{
			Plan:  planName,
			Title: w.RolloutTitle,
		},
		Target:       &emptyTarget, // zero stage
		ValidateOnly: false,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to preview rollout")
	}
	w.OutputMap["rollout"] = rolloutEmpty.Name

	slog.Info("rollout created", "url", fmt.Sprintf("%s/%s", client.url, rolloutEmpty.Name))

	return waitForRollout(ctx, w, client, pendingStages, rolloutEmpty.Name)
}

func waitForRollout(ctx context.Context, w *world.World, client *Client, pendingStages []string, rolloutName string) error {
	if w.TargetStage == "" {
		slog.Info("target stage is not specified, exiting...")
		return nil
	}

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

	rollout, err := client.getRollout(rolloutName)
	if err != nil {
		return errors.Wrapf(err, "failed to get rollout")
	}

	slog.Info("exit after the target stage is completed", "targetStage", w.TargetStage)
	slog.Info("the rollout has the following stages",
		"createdStages", slices.Collect(func(yield func(string) bool) {
			for _, stage := range rollout.GetStages() {
				if !yield(stage.Environment) {
					return
				}
			}
		}),
		"pendingStages", pendingStages)

	targetStageFound := false
	for _, stage := range rollout.GetStages() {
		if stage.Environment == w.TargetStage {
			targetStageFound = true
			break
		}
	}
	for _, stage := range pendingStages {
		if stage == w.TargetStage {
			targetStageFound = true
			break
		}
	}
	if !targetStageFound {
		slog.Info("the target stage is not found in the rollout. exiting...", "targetStage", w.TargetStage)
		return nil
	}

	// To make it more robust,
	// - remove pending stage that already in the rollout stage and call CreateRollout with latest rollout stage once so no new tasks will be missed
	if len(rollout.Stages) > 0 {
		latestStage := rollout.Stages[len(rollout.Stages)-1].Environment
		index := slices.Index(pendingStages, latestStage)
		if index != -1 {
			slog.Info("removing pending stage that already in the rollout stage", "latestStage", latestStage)
			pendingStages = pendingStages[index+1:]
			_, err := client.createRollout(&v1pb.CreateRolloutRequest{
				Parent: w.Project,
				Rollout: &v1pb.Rollout{
					Plan: rollout.GetPlan(),
				},
				Target: &latestStage,
			})
			if err != nil {
				return errors.Wrapf(err, "failed to create rollout")
			}
		}
	}

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
				Parent: w.Project,
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
			if w.TargetStage == stage.Environment {
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
