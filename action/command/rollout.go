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
	"github.com/bytebase/bytebase/action/command/output"
	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func NewRolloutCommand(w *world.World) *cobra.Command {
	// bytebase-action rollout flags
	cmdRollout := &cobra.Command{
		Use:               "rollout",
		Short:             "Rollout the migrate files",
		Args:              cobra.NoArgs,
		PersistentPreRunE: rolloutPreRun(w),
		RunE:              runRollout(w),
	}
	cmdRollout.Flags().StringVar(&w.ReleaseTitle, "release-title", "", "The title of the release. Generated from project and current timestamp if not provided.")
	cmdRollout.Flags().StringVar(&w.TargetStage, "target-stage", "", "Rollout up to the target stage. Format: environments/{environment}.")
	cmdRollout.Flags().StringVar(&w.Plan, "plan", "", "The plan to rollout. Format: projects/{project}/plans/{plan}. Shadows file-pattern and targets.")
	return cmdRollout
}

func rolloutPreRun(w *world.World) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if p := cmd.Parent(); p != nil {
			if p.PersistentPreRunE != nil {
				if err := p.PersistentPreRunE(cmd, args); err != nil {
					return err
				}
			}
		}
		if w.ReleaseTitle == "" {
			w.ReleaseTitle = fmt.Sprintf("[%s] %s", strings.TrimPrefix(w.Project, "projects/"), time.Now().UTC().Format(time.RFC3339))
		}
		return nil
	}
}

func runRollout(w *world.World) func(command *cobra.Command, _ []string) error {
	return func(command *cobra.Command, _ []string) error {
		defer func() {
			output.WriteOutput(w)
		}()
		w.IsRollout = true
		ctx := command.Context()
		client, err := NewClient(w.URL, w.ServiceAccount, w.ServiceAccountSecret)
		if err != nil {
			return errors.Wrapf(err, "failed to create client")
		}
		defer client.Close()

		// Check version compatibility
		CheckVersionCompatibility(w, client, args.Version)

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
			releaseFiles, releaseDigest, err := getReleaseFiles(w)
			if err != nil {
				return errors.Wrapf(err, "failed to get release files")
			}
			releaseType := v1pb.Release_VERSIONED
			if w.Declarative {
				releaseType = v1pb.Release_DECLARATIVE
			}
			createReleaseResponse, err := client.CreateRelease(ctx, w.Project, &v1pb.Release{
				Title:     w.ReleaseTitle,
				Files:     releaseFiles,
				VcsSource: getVCSSource(w),
				Digest:    releaseDigest,
				Type:      releaseType,
			})
			if err != nil {
				return errors.Wrapf(err, "failed to create release")
			}
			w.Logger.Info("release created", "url", fmt.Sprintf("%s/%s", client.url, createReleaseResponse.Name))
			release = createReleaseResponse.Name
			w.OutputMap.Release = release

			planCreated, err := client.CreatePlan(ctx, w.Project, &v1pb.Plan{
				Title: "Release " + w.ReleaseTitle,
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
		w.OutputMap.Plan = plan.Name

		if err := runAndWaitForRollout(ctx, w, client, plan.Name); err != nil {
			return errors.Wrapf(err, "failed to run and wait for rollout")
		}

		return nil
	}
}

func runAndWaitForRollout(ctx context.Context, w *world.World, client *Client, planName string) error {
	// create rollout with all stages created
	rollout, err := client.CreateRollout(ctx, &v1pb.CreateRolloutRequest{
		Parent: planName,
		Target: nil, // all stages
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create rollout")
	}
	w.OutputMap.Rollout = rollout.Name
	w.Rollout = rollout

	w.Logger.Info("rollout created", "url", fmt.Sprintf("%s/%s", client.url, rollout.Name))

	return waitForRollout(ctx, w, client, rollout.Name)
}

func waitForRollout(ctx context.Context, w *world.World, client *Client, rolloutName string) error {
	if w.TargetStage == "" {
		w.Logger.Info("target stage is not specified, exiting...")
		return nil
	}

	defer func() {
		if ctx.Err() == nil {
			return
		}
		w.Logger.Info("context cancelled, canceling the rollout")
		if err := cancelRollout(context.Background(), client, rolloutName); err != nil {
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
		}))
	if len(rollout.Stages) == 0 {
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
	if !targetStageFound {
		w.Logger.Info("the target stage is not found in the rollout. exiting...", "targetStage", w.TargetStage)
		return nil
	}

	for i := range rollout.Stages {
		// Polling loop for the current stage
		for {
			if ctx.Err() != nil {
				return errors.Wrapf(ctx.Err(), "context cancelled")
			}
			// get rollout to refresh status
			rollout, err := client.GetRollout(ctx, rolloutName)
			if err != nil {
				return errors.Wrapf(err, "failed to get rollout")
			}
			w.Rollout = rollout
			if i >= len(rollout.Stages) {
				return errors.Errorf("rollout stage index out of bounds")
			}
			stage := rollout.Stages[i]

			// check stage tasks
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
					return nil
				}
				// Break inner loop to proceed to next stage
				break
			}

			// run stage tasks
			if len(notStartedTasks) > 0 {
				w.Logger.Info("running stage tasks", "stage", stage.Environment, "taskCount", len(notStartedTasks))
				if _, err := client.BatchRunTasks(ctx, &v1pb.BatchRunTasksRequest{
					Parent: stage.Name,
					Tasks:  notStartedTasks,
				}); err != nil {
					return errors.Wrapf(err, "failed to batch create tasks")
				}
			}
			time.Sleep(5 * time.Second)
		}
	}

	return nil
}

func cancelRollout(ctx context.Context, client *Client, rolloutName string) error {
	taskRuns, err := client.ListAllTaskRuns(ctx, rolloutName)
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
		_, err := client.BatchCancelTaskRuns(ctx, &v1pb.BatchCancelTaskRunsRequest{
			Parent:   stage + "/tasks/-",
			TaskRuns: taskRuns,
		})
		if err != nil {
			err = errors.Wrapf(err, "failed to cancel task runs for stage %v", stage)
			errs = multierr.Append(errs, err)
		}
	}
	return errs
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
