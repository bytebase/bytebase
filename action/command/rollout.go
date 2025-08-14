package command

import (
	"context"
	"fmt"
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
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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
	defaultTitle := w.CurrentTime.Format(time.RFC3339)
	cmdRollout.Flags().StringVar(&w.ReleaseTitle, "release-title", defaultTitle, "The title of the release")
	cmdRollout.Flags().StringVar(&w.CheckPlan, "check-plan", "SKIP", "Whether to check the plan and fail on warning/error. Valid values: SKIP, FAIL_ON_WARNING, FAIL_ON_ERROR")
	cmdRollout.Flags().StringVar(&w.TargetStage, "target-stage", "", "Rollout up to the target stage. Format: environments/{environment}.")
	cmdRollout.Flags().StringVar(&w.Plan, "plan", "", "The plan to rollout. Format: projects/{project}/plans/{plan}. Shadows file-pattern and targets.")
	cmdRollout.Flags().BoolVar(&w.Declarative, "declarative", false, "Whether to use declarative mode.")
	return cmdRollout
}

func validateRolloutFlags(w *world.World) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if p := cmd.Parent(); p != nil {
			if p.PersistentPreRunE != nil {
				if err := p.PersistentPreRunE(cmd, args); err != nil {
					return err
				}
			}
		}
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
		defer func() {
			writeOutputJSON(w)
		}()
		ctx := command.Context()
		client, err := NewClient(w.URL, w.ServiceAccount, w.ServiceAccountSecret)
		if err != nil {
			return errors.Wrapf(err, "failed to create client")
		}

		// Check version compatibility
		checkVersionCompatibility(w, client, args.Version)

		var plan *v1pb.Plan
		if w.Plan != "" {
			planP, err := client.GetPlan(ctx, w.Plan)
			if err != nil {
				return errors.Wrapf(err, "failed to get plan")
			}
			plan = planP
			w.Logger.Info("use the provided plan", "url", fmt.Sprintf("%s/%s", client.url, plan.Name))
		} else {
			var release string
			releaseFiles, releaseDigest, err := getReleaseFiles(w, w.FilePattern)
			if err != nil {
				return errors.Wrapf(err, "failed to get release files")
			}
			// Search release by digest so that we don't create duplicate releases.
			searchRelease, err := client.GetReleaseByDigest(ctx, w.Project, releaseDigest)
			if err != nil {
				return errors.Wrapf(err, "failed to get release by digest")
			}
			if searchRelease != nil {
				w.Logger.Info("found release by digest", "url", fmt.Sprintf("%s/%s", client.url, searchRelease.Name))
				release = searchRelease.Name
			} else {
				createReleaseResponse, err := client.CreateRelease(ctx, w.Project, &v1pb.Release{
					Title:     w.ReleaseTitle,
					Files:     releaseFiles,
					VcsSource: getVCSSource(w),
					Digest:    releaseDigest,
				})
				if err != nil {
					return errors.Wrapf(err, "failed to create release")
				}
				w.Logger.Info("release created", "url", fmt.Sprintf("%s/%s", client.url, createReleaseResponse.Name))
				release = createReleaseResponse.Name
			}
			w.OutputMap["release"] = release

			planCreated, err := client.CreatePlan(ctx, w.Project, &v1pb.Plan{
				Title: w.ReleaseTitle,
				Specs: []*v1pb.Plan_Spec{
					{
						Id: uuid.New().String(),
						Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
							ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
								Targets: w.Targets,
								Release: release,
							},
						},
					},
				},
			})
			if err != nil {
				return errors.Wrapf(err, "failed to create plan")
			}
			plan = planCreated
			w.Logger.Info("plan created", "url", fmt.Sprintf("%s/%s", client.url, plan.Name))
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

		runs, err := client.ListAllPlanCheckRuns(ctx, planName)
		if err != nil {
			return errors.Wrapf(err, "failed to list plan checks")
		}
		if len(runs.PlanCheckRuns) == 0 {
			w.Logger.Info("running plan checks")
			_, err := client.RunPlanChecks(ctx, &v1pb.RunPlanChecksRequest{
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
					default:
						// Other result statuses don't affect counts
					}
				}
			default:
				// Other run statuses don't affect counts
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
		w.Logger.Info("waiting for plan checks to complete", "runningCount", runningCount)
		time.Sleep(5 * time.Second)
	}

	return nil
}

func runAndWaitForRollout(ctx context.Context, w *world.World, client *Client, planName string) error {
	// preview rollout with all pending stages
	rolloutPreview, err := client.CreateRollout(ctx, &v1pb.CreateRolloutRequest{
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
	rolloutEmpty, err := client.CreateRollout(ctx, &v1pb.CreateRolloutRequest{
		Parent: w.Project,
		Rollout: &v1pb.Rollout{
			Plan: planName,
		},
		Target:       &emptyTarget, // zero stage
		ValidateOnly: false,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to preview rollout")
	}
	w.OutputMap["rollout"] = rolloutEmpty.Name

	w.Logger.Info("rollout created", "url", fmt.Sprintf("%s/%s", client.url, rolloutEmpty.Name))

	return waitForRollout(ctx, w, client, pendingStages, rolloutEmpty.Name)
}

func waitForRollout(ctx context.Context, w *world.World, client *Client, pendingStages []string, rolloutName string) error {
	if w.TargetStage == "" {
		w.Logger.Info("target stage is not specified, exiting...")
		return nil
	}

	defer func() {
		if ctx.Err() == nil {
			return
		}
		w.Logger.Info("context cancelled, canceling the rollout")
		// cancel rollout
		if err := func() error {
			taskRuns, err := client.ListAllTaskRuns(context.Background(), rolloutName)
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
				_, err := client.BatchCancelTaskRuns(context.Background(), &v1pb.BatchCancelTaskRunsRequest{
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
			w.Logger.Error("failed to cancel rollout", "error", err)
		}
	}()

	rollout, err := client.GetRollout(ctx, rolloutName)
	if err != nil {
		return errors.Wrapf(err, "failed to get rollout")
	}

	w.Logger.Info("exit after the target stage is completed", "targetStage", w.TargetStage)
	w.Logger.Info("the rollout has the following stages",
		"createdStages", slices.Collect(func(yield func(string) bool) {
			for _, stage := range rollout.GetStages() {
				if !yield(stage.Environment) {
					return
				}
			}
		}),
		"pendingStages", pendingStages)
	if len(pendingStages) == 0 && len(rollout.Stages) == 0 {
		w.Logger.Info("no stages in the rollout. exiting...")
		return nil
	}

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
		w.Logger.Info("the target stage is not found in the rollout. exiting...", "targetStage", w.TargetStage)
		return nil
	}

	// To make it more robust,
	// - remove pending stage that already in the rollout stage and call CreateRollout with latest rollout stage once so no new tasks will be missed
	if len(rollout.Stages) > 0 {
		latestStage := rollout.Stages[len(rollout.Stages)-1].Environment
		index := slices.Index(pendingStages, latestStage)
		if index != -1 {
			w.Logger.Info("removing pending stage that already in the rollout stage", "latestStage", latestStage)
			pendingStages = pendingStages[index+1:]
			_, err := client.CreateRollout(ctx, &v1pb.CreateRolloutRequest{
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
		rollout, err := client.GetRollout(ctx, rolloutName)
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

			rolloutAdvanced, err := client.CreateRollout(ctx, &v1pb.CreateRolloutRequest{
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
			default:
				// Treat unknown task status as not done to be safe
				done = false
			}
		}
		if foundFailed {
			return errors.Errorf("found failed tasks. view on Bytebase")
		}
		if foundCanceled {
			return errors.Errorf("found canceled tasks. view on Bytebase")
		}
		if done {
			w.Logger.Info("stage completed", "stage", stage.Environment)
			if w.TargetStage == stage.Environment {
				break
			}
			i++
			continue
		}

		// run stage tasks
		if len(notStartedTasks) > 0 {
			w.Logger.Info("running stage tasks", "stage", stage.Environment, "taskCount", len(notStartedTasks))
			if _, err := client.BatchRunTasks(ctx, &v1pb.BatchRunTasksRequest{
				Parent: stage.Name,
				Tasks:  notStartedTasks,
			}); err != nil {
				// Check for specific error indicating task runs already exist (retryable)
				if !strings.Contains(err.Error(), "cannot create pending task runs because there are pending/running/done task runs") {
					return errors.Wrapf(err, "failed to batch create tasks")
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

func getVCSSource(w *world.World) *v1pb.Release_VCSSource {
	switch w.Platform {
	case world.GitHub:
		return &v1pb.Release_VCSSource{
			VcsType: v1pb.VCSType_GITHUB,
			Url:     os.Getenv("GITHUB_SERVER_URL") + "/" + os.Getenv("GITHUB_REPOSITORY") + "/commit/" + os.Getenv("GITHUB_SHA"),
		}
	case world.GitLab:
		return &v1pb.Release_VCSSource{
			VcsType: v1pb.VCSType_GITLAB,
			Url:     os.Getenv("CI_PROJECT_URL") + "/-/commit/" + os.Getenv("CI_COMMIT_SHA"),
		}
	case world.Bitbucket:
		return &v1pb.Release_VCSSource{
			VcsType: v1pb.VCSType_BITBUCKET,
			Url:     os.Getenv("BITBUCKET_GIT_HTTP_ORIGIN") + "/commits/" + os.Getenv("BITBUCKET_COMMIT"),
		}
	case world.AzureDevOps:
		return &v1pb.Release_VCSSource{
			VcsType: v1pb.VCSType_AZURE_DEVOPS,
			Url:     os.Getenv("SYSTEM_COLLECTIONURI") + os.Getenv("SYSTEM_TEAMPROJECT") + "/_git/" + os.Getenv("BUILD_REPOSITORY_NAME") + "/commit/" + os.Getenv("BUILD_SOURCEVERSION"),
		}
	default:
		return nil
	}
}
