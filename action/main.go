package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/bytebase/bytebase/action/github"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	Config struct {
		// bytebase-action flags
		URL                  string
		ServiceAccount       string
		ServiceAccountSecret string
		Project              string
		Targets              []string

		// bytebase-action check flags
		FilePattern string
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

	// bytebase-action check flags
	cmdCheck := &cobra.Command{
		Use:   "check",
		Short: "Check the release files",
		Args:  cobra.NoArgs,
		RunE:  runCI,
	}
	cmdCheck.PersistentFlags().StringVar(&Config.FilePattern, "file-pattern", "", "File pattern to glob migration files")

	cmd.AddCommand(cmdCheck)
}

func Execute() error {
	return cmd.Execute()
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

func main() {
	cmd.Execute()
}
