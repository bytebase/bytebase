package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/bytebase/bytebase/action/github"
	"github.com/bytebase/bytebase/backend/common/log"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	Config struct {
		// bytebase-action flags
		URL                  string
		ServiceAccount       string
		ServiceAccountSecret string
		Project              string // projects/{project}
		Targets              []string
		FilePattern          string

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
	}
	cmd = &cobra.Command{
		Use:   "bytebase-action",
		Short: "Bytebase action",
	}
)

func init() {
	// bytebase-action flags
	cmd.PersistentFlags().StringVar(&Config.URL, "url", "https://demo.bytebase.com", "Bytebase URL")
	cmd.PersistentFlags().StringVar(&Config.ServiceAccount, "service-account", "ci@service.bytebase.com", "Bytebase Service account")
	cmd.PersistentFlags().StringVar(&Config.ServiceAccountSecret, "service-account-secret", os.Getenv("BYTEBASE_SERVICE_ACCOUNT_SECRET"), "Bytebase Service account secret")
	// cmd.MarkPersistentFlagRequired("service-account-secret")
	cmd.PersistentFlags().StringVar(&Config.Project, "project", "projects/project-sample", "Bytebase project")
	cmd.PersistentFlags().StringSliceVar(&Config.Targets, "targets", []string{"instances/test-sample-instance/databases/hr_test", "instances/prod-sample-instance/databases/hr_prod"}, "Bytebase targets")
	cmd.PersistentFlags().StringVar(&Config.FilePattern, "file-pattern", "", "File pattern to glob migration files")

	// bytebase-action check flags
	cmdCheck := &cobra.Command{
		Use:   "check",
		Short: "Check the release files",
		Args:  cobra.NoArgs,
		RunE:  runCI,
	}
	cmd.AddCommand(cmdCheck)

	cmdRollout := &cobra.Command{
		Use:   "rollout",
		Short: "Rollout the migrate files",
		Args:  cobra.NoArgs,
		RunE:  runRollout,
	}
	defaultTitle := time.Now().Format(time.RFC3339)
	cmdRollout.Flags().StringVar(&Config.ReleaseTitle, "release-title", defaultTitle, "The title of the release")
	cmdRollout.Flags().StringVar(&Config.RolloutTitle, "rollout-title", defaultTitle, "The title of the rollout")
	cmdRollout.Flags().StringVar(&Config.CheckPlan, "check-plan", "SKIP", "Whether to check the plan and fail on warning/error. Valid values: SKIP, FAIL_ON_WARNING, FAIL_ON_ERROR")
	cmdRollout.Flags().StringVar(&Config.TargetStage, "target-stage", "", "Rollout up to the target stage. Format: environments/{environment}.")
	cmd.AddCommand(cmdRollout)
}

func Execute() error {
	ctx := context.Background()
	return cmd.ExecuteContext(ctx)
}

func runCI(*cobra.Command, []string) error {
	platform := getJobPlatform()
	client, err := NewClient(Config.URL, Config.ServiceAccount, Config.ServiceAccountSecret)
	if err != nil {
		return err
	}

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
	return nil
}

func runRollout(command *cobra.Command, _ []string) error {
	ctx := command.Context()
	client, err := NewClient(Config.URL, Config.ServiceAccount, Config.ServiceAccountSecret)
	if err != nil {
		return errors.Wrapf(err, "failed to create client")
	}
	releaseFiles, err := getReleaseFiles(Config.FilePattern)
	if err != nil {
		return errors.Wrapf(err, "failed to get release files")
	}
	createReleaseResponse, err := client.createRelease(Config.Project, &v1pb.Release{
		Title:     Config.ReleaseTitle,
		Files:     releaseFiles,
		VcsSource: nil, // TODO(p0ny): impl
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create release")
	}

	planPreview, err := client.previewPlan(Config.Project, &v1pb.PreviewPlanRequest{
		Release:         createReleaseResponse.Name,
		Targets:         Config.Targets,
		AllowOutOfOrder: true,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to preview plan")
	}

	planCreated, err := client.createPlan(Config.Project, planPreview.Plan)
	if err != nil {
		return errors.Wrapf(err, "failed to create plan")
	}

	if err := runAndWaitForPlanChecks(ctx, client, planCreated.Name); err != nil {
		return errors.Wrapf(err, "failed to run and wait for plan checks")
	}

	rolloutCreated, err := client.createRollout(Config.Project, &v1pb.CreateRolloutRequest{
		Rollout: &v1pb.Rollout{
			Plan: planCreated.Name,
		},
		Target: nil, // TODO(p0ny): impl
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create rollout")
	}

	_ = rolloutCreated
	// TODO(p0ny): wait for rollout to complete the target stage

	return nil
}

func runAndWaitForPlanChecks(ctx context.Context, client *Client, planName string) error {
	if Config.CheckPlan == "SKIP" {
		return nil
	}

	_, err := client.runPlanChecks(&v1pb.RunPlanChecksRequest{
		Name: planName,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to run plan checks")
	}
	for {
		if ctx.Err() != nil {
			return errors.Wrapf(ctx.Err(), "context cancelled")
		}

		runs, err := client.listAllPlanCheckRuns(planName)
		if err != nil {
			return errors.Wrapf(err, "failed to list plan checks")
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
			return errors.Errorf("found failed plan checks. View on Bytebase.")
		}
		if canceledCount > 0 {
			return errors.Errorf("found canceled plan checks. View on Bytebase.")
		}
		if errorCount > 0 {
			return errors.Errorf("found error plan checks. View on Bytebase.")
		}
		if warningCount > 0 && Config.CheckPlan == "FAIL_ON_WARNING" {
			return errors.Errorf("found warning plan checks. View on Bytebase.")
		}
		if runningCount == 0 {
			break
		}
		time.Sleep(5 * time.Second)
	}

	return nil
}

func main() {
	if err := cmd.Execute(); err != nil {
		slog.Error("failed to execute command", log.BBError(err))
		os.Exit(1)
	}
}
