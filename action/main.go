package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func run(platform JobPlatform) error {
	url, serviceAccount, serviceAccountSecret := os.Getenv("BYTEBASE_URL"), os.Getenv("BYTEBASE_SERVICE_ACCOUNT"), os.Getenv("BYTEBASE_SERVICE_ACCOUNT_SECRET")
	if url == "" {
		return errors.Errorf("environment BYTEBASE_URL is not set")
	}
	if serviceAccount == "" {
		return errors.Errorf("environment BYTEBASE_SERVICE_ACCOUNT is not set")
	}
	if serviceAccountSecret == "" {
		return errors.Errorf("environment BYTEBASE_SERVICE_ACCOUNT_SECRET is not set")
	}
	project, bytebaseTargets, filePattern := os.Getenv("BYTEBASE_PROJECT"), os.Getenv("BYTEBASE_TARGETS"), os.Getenv("FILE_PATTERN")
	if project == "" {
		return errors.Errorf("environment BYTEBASE_PROJECT is not set")
	}
	if bytebaseTargets == "" {
		return errors.Errorf("environment BYTEBASE_TARGETS is not set")
	}
	if filePattern == "" {
		return errors.Errorf("environment FILE_PATTERN is not set")
	}
	targets := strings.Split(bytebaseTargets, ",")

	client, err := NewClient(url, serviceAccount, serviceAccountSecret)
	if err != nil {
		return err
	}

	releaseFiles, err := getReleaseFiles(filePattern)
	if err != nil {
		return err
	}
	checkReleaseResponse, err := client.checkRelease(project, &v1pb.CheckReleaseRequest{
		Release: &v1pb.Release{Files: releaseFiles},
		Targets: targets,
	})
	if err != nil {
		return err
	}
	switch platform {
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
	platform := getJobPlatform()
	fmt.Printf("Hello, World - %s!\n", platform.String())
	if err := run(platform); err != nil {
		panic(err)
	}
}
