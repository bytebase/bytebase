package command

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/bytebase/bytebase/action/args"
	"github.com/bytebase/bytebase/action/azure"
	"github.com/bytebase/bytebase/action/bitbucket"
	"github.com/bytebase/bytebase/action/command/output"
	"github.com/bytebase/bytebase/action/github"
	"github.com/bytebase/bytebase/action/gitlab"
	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func NewCheckCommand(w *world.World) *cobra.Command {
	// bytebase-action check flags
	cmdCheck := &cobra.Command{
		Use:               "check",
		Short:             "Check the release files",
		Args:              cobra.NoArgs,
		PersistentPreRunE: validateCheckFlags(w),
		RunE:              runCheck(w),
	}
	cmdCheck.Flags().StringVar(&w.CheckRelease, "check-release", "SKIP", "Whether to fail on warning/error. Valid values: SKIP, FAIL_ON_WARNING, FAIL_ON_ERROR")
	cmdCheck.Flags().StringVar(&w.CustomRules, "custom-rules", "", "Custom linting rules in natural language for AI-powered validation")
	return cmdCheck
}

func validateCheckFlags(w *world.World) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if p := cmd.Parent(); p != nil {
			if p.PersistentPreRunE != nil {
				if err := p.PersistentPreRunE(cmd, args); err != nil {
					return err
				}
			}
		}
		switch w.CheckRelease {
		case "SKIP", "FAIL_ON_WARNING", "FAIL_ON_ERROR":
		default:
			return errors.Errorf("invalid check-release value: %s. Valid values: SKIP, FAIL_ON_WARNING, FAIL_ON_ERROR", w.CheckRelease)
		}
		return nil
	}
}

func runCheck(w *world.World) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		defer func() {
			output.WriteOutput(w)
		}()
		platform := w.Platform
		w.Logger.Info("running on platform", "platform", platform.String())
		client, err := NewClient(w.URL, w.ServiceAccount, w.ServiceAccountSecret)
		if err != nil {
			return err
		}
		defer client.Close()

		// Check version compatibility
		CheckVersionCompatibility(w, client, args.Version)

		releaseFiles, _, err := getReleaseFiles(w)
		if err != nil {
			return err
		}
		releaseType := v1pb.Release_VERSIONED
		if w.Declarative {
			releaseType = v1pb.Release_DECLARATIVE
		}
		checkReleaseResponse, err := client.CheckRelease(cmd.Context(), &v1pb.CheckReleaseRequest{
			Parent: w.Project,
			Release: &v1pb.Release{
				Files: releaseFiles,
				Type:  releaseType,
			},
			Targets:     w.Targets,
			CustomRules: w.CustomRules,
		})
		if err != nil {
			return err
		}

		// Store check results in OutputMap for file output
		w.OutputMap.CheckResults = checkReleaseResponse

		w.Logger.Info("check release response", "resultCount", len(checkReleaseResponse.Results))

		// Generate platform-specific outputs
		switch platform {
		case world.GitHub:
			if err := github.CreateCommentAndAnnotation(checkReleaseResponse); err != nil {
				return err
			}
		case world.GitLab:
			if err := gitlab.WriteReleaseCheckToCodeQualityJSON(checkReleaseResponse); err != nil {
				return err
			}
		case world.AzureDevOps:
			if err := azure.LoggingReleaseChecks(checkReleaseResponse); err != nil {
				return err
			}
		case world.Bitbucket:
			if err := bitbucket.CreateBitbucketReport(checkReleaseResponse); err != nil {
				return err
			}
		default:
			// Unknown platform, no specific output handling
		}

		// Evaluate check results and return errors based on CheckRelease flag
		if w.CheckRelease == "SKIP" {
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
				default:
					// Other advice statuses don't affect counts
				}
			}
		}

		if errorCount > 0 {
			return errors.Errorf("found %d error(s) in release check. view on Bytebase", errorCount)
		}
		if warningCount > 0 && w.CheckRelease == "FAIL_ON_WARNING" {
			return errors.Errorf("found %d warning(s) in release check. view on Bytebase", warningCount)
		}

		return nil
	}
}
