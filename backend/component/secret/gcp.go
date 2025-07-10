package secret

import (
	"context"
	"fmt"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func getGCPSecretManagerClient(ctx context.Context) (*secretmanager.Client, error) {
	// will find default credentials in GKE, fallback to GOOGLE_APPLICATION_CREDENTIALS envionment.
	// https://pkg.go.dev/golang.org/x/oauth2/google#FindDefaultCredentialsWithParams
	creds, err := google.FindDefaultCredentials(ctx, secretmanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get GCP credentials")
	}
	client, err := secretmanager.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get GCP secret manager client")
	}
	return client, nil
}

func getSecretFromGCP(ctx context.Context, externalSecret *storepb.DataSourceExternalSecret) (string, error) {
	client, err := getGCPSecretManagerClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		// The secret name should like
		// projects/{project}/secrets/{secret name}/versions/{version}
		// we will use latest for te version
		Name: fmt.Sprintf("%s/versions/latest", externalSecret.SecretName),
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			return "", errors.Wrapf(err, "cannot found secret %s", externalSecret.SecretName)
		}
		return "", errors.Wrapf(err, "failed to get GCP secret %s", externalSecret.SecretName)
	}

	return string(result.Payload.Data), nil
}
