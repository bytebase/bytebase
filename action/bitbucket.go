package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func createBitbucketReport(checkResponse *v1pb.CheckReleaseResponse) error {
	repoOwner := os.Getenv("BITBUCKET_REPO_OWNER")
	repoSlug := os.Getenv("BITBUCKET_REPO_SLUG")
	commit := os.Getenv("BITBUCKET_COMMIT")
	if repoOwner == "" || repoSlug == "" || commit == "" {
		return errors.Errorf("BITBUCKET_REPO_OWNER, BITBUCKET_REPO_SLUG, and BITBUCKET_COMMIT environment variables must be set")
	}
	proxy, err := url.Parse("http://localhost:29418")
	if err != nil {
		return err
	}
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxy),
	}
	client := &http.Client{Transport: transport}

	targetURL := fmt.Sprintf("http://api.bitbucket.org/2.0/repositories/%s/%s/commit/%s/reports/bytebase", repoOwner, repoSlug, commit)
	var warningCount, errorCount int
	for _, result := range checkResponse.Results {
		for _, advice := range result.Advices {
			switch advice.Status {
			case v1pb.Advice_WARNING:
				warningCount++
			case v1pb.Advice_ERROR:
				errorCount++
			default:
				continue
			}
		}
	}
	result, details := "PASSED", "This pull request has passed all checks."
	if errorCount > 0 {
		result, details = "FAILED", fmt.Sprintf("This pull request introduces %d errors and %d warnings.", errorCount, warningCount)
	} else if warningCount > 0 {
		details = fmt.Sprintf("This pull request introduces %d warnings.", warningCount)
	}
	requestData := fmt.Sprintf(`{
		"title": "Bytebase SQL Review",
		"details": "%s",
		"report_type": "TEST",
		"reporter": "bytebase",
		"result": "%s",
		"data": []
	}`, details, result)
	req, err := http.NewRequest("PUT", targetURL, bytes.NewBufferString(requestData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrapf(err, "failed to create Bitbucket report, status code: %d, and failed to read response body: %v", resp.StatusCode, err)
		}
		return errors.Errorf("failed to create Bitbucket report, status code: %d, response body: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}
