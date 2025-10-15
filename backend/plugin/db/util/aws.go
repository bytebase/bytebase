//nolint:revive
package util

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// AssumeRoleIfNeeded checks if role assumption is configured and updates the AWS config with assumed role credentials.
// This is a shared utility used by multiple AWS service drivers (Elasticsearch, RDS, etc.).
// Returns an error if role assumption fails.
func AssumeRoleIfNeeded(ctx context.Context, awsCfg *aws.Config, connectionCtx db.ConnectionContext, awsCredential *storepb.DataSource_AWSCredential) error {
	// If no role ARN is provided, no assumption needed
	if awsCredential == nil || awsCredential.RoleArn == "" {
		return nil
	}

	roleArn := awsCredential.RoleArn

	// Create STS client with base credentials
	stsClient := sts.NewFromConfig(*awsCfg)

	// Generate descriptive session name for CloudTrail auditing
	sessionName := generateSessionName(connectionCtx.InstanceID)

	// Configure assume role provider
	assumeRoleProvider := stscreds.NewAssumeRoleProvider(stsClient, roleArn,
		func(o *stscreds.AssumeRoleOptions) {
			o.RoleSessionName = sessionName
			o.Duration = 1 * time.Hour // Temporary credentials valid for 1 hour

			// Add external ID if provided for additional security
			if externalID := awsCredential.ExternalId; externalID != "" {
				o.ExternalID = &externalID
			}
		})

	// Update config with assumed role credentials
	awsCfg.Credentials = assumeRoleProvider

	// Test credentials retrieval and provide context-specific error messages
	_, err := awsCfg.Credentials.Retrieve(ctx)
	if err != nil {
		return handleAssumeRoleError(err, roleArn, awsCredential.ExternalId)
	}

	return nil
}

// generateSessionName creates a descriptive session name for AWS CloudTrail auditing.
// Format: bytebase-{instance-id}-{timestamp}
// AWS constraints: 2-64 characters, matching pattern [\w+=,.@-]*
func generateSessionName(instanceID string) string {
	// Sanitize instance ID to ensure valid session name (alphanumeric, =,.@-)
	sanitizedID := "unknown"
	if instanceID != "" {
		sanitizedID = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
				return r
			}
			return '-'
		}, instanceID)
	}

	// Generate timestamp
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	// Calculate max length for instance ID to stay within 64 char limit
	// Format: "bytebase-" (9) + instanceID + "-" (1) + timestamp (10) = 20 overhead
	// So instanceID can be at most 44 characters (64 - 20)
	maxInstanceIDLength := 44
	if len(sanitizedID) > maxInstanceIDLength {
		// Truncate but keep the end part which is usually more unique
		sanitizedID = sanitizedID[len(sanitizedID)-maxInstanceIDLength:]
	}

	sessionName := fmt.Sprintf("bytebase-%s-%s", sanitizedID, timestamp)

	// Final safety check (should never happen with our math above)
	if len(sessionName) > 64 {
		sessionName = sessionName[:64]
	}

	return sessionName
}

// handleAssumeRoleError provides context-specific error messages for role assumption failures.
func handleAssumeRoleError(err error, roleArn string, externalID string) error {
	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "AccessDenied"):
		if externalID != "" {
			return errors.Errorf("failed to assume role %s: access denied (external ID configured: %s)", roleArn, externalID)
		}
		return errors.Errorf("failed to assume role %s: access denied", roleArn)

	case strings.Contains(errStr, "InvalidParameterValue") && strings.Contains(errStr, "ExternalId"):
		return errors.Errorf("failed to assume role %s: external ID mismatch", roleArn)

	case strings.Contains(errStr, "NoSuchEntity") || strings.Contains(errStr, "not found"):
		return errors.Errorf("failed to assume role %s: role not found", roleArn)

	case strings.Contains(errStr, "ExpiredToken") || strings.Contains(errStr, "TokenRefreshRequired"):
		return errors.Errorf("failed to assume role %s: base AWS credentials expired", roleArn)

	case strings.Contains(errStr, "throttling") || strings.Contains(errStr, "TooManyRequests"):
		return errors.Errorf("failed to assume role %s: AWS API rate limit exceeded", roleArn)

	default:
		return errors.Wrapf(err, "failed to assume role %s", roleArn)
	}
}
