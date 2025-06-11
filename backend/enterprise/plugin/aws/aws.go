package aws

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/enterprise/plugin"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/licensemanager"
	"github.com/aws/aws-sdk-go-v2/service/licensemanager/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var _ plugin.LicenseProvider = (*Provider)(nil)

// Provider is the AWS license provider.
type Provider struct {
	projectID      string
	licenseManager *licensemanager.Client
	identity       *sts.GetCallerIdentityOutput
}

// NewProvider will create a new AWS license provider.
// To use the aws provider in local development, you need to follow the guide https://catalog.workshops.aws/mpseller/en-US/container/integrate-contract#task-1:-create-a-test-license
// to subscribe our test product in the marketplace,
// and expose AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION and PRODUCT_ID in the environment.
// And the AWS account must have the permission to access the AWS license manager.
func NewProvider(_ *plugin.ProviderConfig) (plugin.LicenseProvider, error) {
	projectID := os.Getenv("AWS_PRODUCT_ID")
	if projectID == "" {
		return nil, errors.Errorf("cannot find aws project id")
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := sts.NewFromConfig(cfg)
	identity, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}

	return &Provider{
		projectID:      projectID,
		identity:       identity,
		licenseManager: licensemanager.NewFromConfig(cfg),
	}, nil
}

// StoreLicense will store the aws license.
func (*Provider) StoreLicense(_ context.Context, _ string) error {
	return nil
}

// LoadSubscription will load the aws subscription.
func (p *Provider) LoadSubscription(ctx context.Context) *v1pb.Subscription {
	license, err := p.checkoutLicense(ctx)
	if err != nil {
		slog.Error("failed to checkout license",
			log.BBError(err),
		)
		return nil
	}

	if license.Status != types.LicenseStatusAvailable {
		return nil
	}

	subscription := &v1pb.Subscription{
		Plan:    v1pb.PlanType_ENTERPRISE,
		OrgName: aws.ToString(p.identity.Account),
	}

	if v := license.Validity; v != nil {
		end, err := time.Parse(time.RFC3339, aws.ToString(v.End))
		if err != nil {
			slog.Error("failed to parse subscription end time",
				slog.String("end", *v.Begin),
				log.BBError(err),
			)
			return nil
		}
		subscription.ExpiresTime = timestamppb.New(end.UTC())
	}

	for _, entitlement := range license.Entitlements {
		name := entitlement.Name
		if name == nil {
			continue
		}
		switch *name {
		case "instance":
			subscription.InstanceCount = int32(aws.ToInt64(entitlement.MaxCount))
		case "seat":
			subscription.SeatCount = int32(aws.ToInt64(entitlement.MaxCount))
		}
	}

	return subscription
}

func (p *Provider) checkoutLicense(ctx context.Context) (*types.GrantedLicense, error) {
	productSKUField := "ProductSKU"
	issuerNameField := "IssuerName"
	var maxResults int32 = 1

	res, err := p.licenseManager.ListReceivedLicenses(ctx, &licensemanager.ListReceivedLicensesInput{
		Filters: []types.Filter{
			{
				Name:   &productSKUField,
				Values: []string{p.projectID},
			},
			{
				Name:   &issuerNameField,
				Values: []string{"AWS/Marketplace"},
			},
		},
		MaxResults: &maxResults,
	})
	if err != nil {
		return nil, err
	}

	if len(res.Licenses) != 1 {
		return nil, errors.Errorf("failed to list aws license")
	}

	return &res.Licenses[0], nil
}
