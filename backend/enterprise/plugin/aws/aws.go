package aws

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/enterprise/plugin"
	api "github.com/bytebase/bytebase/backend/legacyapi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	lm "github.com/aws/aws-sdk-go-v2/service/licensemanager"
	"github.com/aws/aws-sdk-go-v2/service/licensemanager/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var _ plugin.LicenseProvider = (*Provider)(nil)

// Provider is the AWS license provider.
type Provider struct {
	projectID      string
	licenseManager *lm.Client
	identity       *sts.GetCallerIdentityOutput
}

// NewProvider will create a new AWS license provider.
func NewProvider(providerConfig *plugin.ProviderConfig) (plugin.LicenseProvider, error) {
	projectID := os.Getenv("PRODUCT_ID")
	if projectID == "" {
		return nil, errors.Errorf("cannot find aws project id")
	}

	ctx := context.TODO()
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
		licenseManager: lm.NewFromConfig(cfg),
	}, nil
}

// StoreLicense will store the hub license.
func (*Provider) StoreLicense(_ context.Context, _ *enterprise.SubscriptionPatch) error {
	return nil
}

// LoadSubscription will load the hub subscription.
func (p *Provider) LoadSubscription(ctx context.Context) *enterprise.Subscription {
	subscription := &enterprise.Subscription{
		InstanceCount: 0,
		Plan:          api.FREE,
		OrgID:         aws.ToString(p.identity.Account),
		OrgName:       aws.ToString(p.identity.Arn),
	}

	license, err := p.checkoutLicense(ctx)
	if err != nil {
		slog.Debug("failed to checkout license",
			log.BBError(err),
		)
		return subscription
	}

	if license.Status != types.LicenseStatusAvailable {
		return subscription
	}

	subscription.Plan = api.TEAM

	if v := license.Validity; v != nil {
		begin, err := time.Parse(time.RFC3339, aws.ToString(v.Begin))
		if err != nil {
			slog.Debug("failed to parse subscription begin time",
				slog.String("begin", *v.Begin),
				log.BBError(err),
			)
		} else {
			subscription.StartedTs = begin.UTC().Unix()
		}

		end, err := time.Parse(time.RFC3339, aws.ToString(v.End))
		if err != nil {
			slog.Debug("failed to parse subscription end time",
				slog.String("end", *v.Begin),
				log.BBError(err),
			)
		} else {
			subscription.ExpiresTs = end.UTC().Unix()
		}
	}

	for _, entitlement := range license.Entitlements {
		if v := entitlement.Name; v != nil && *v == "instance" {
			subscription.InstanceCount = int(aws.ToInt64(entitlement.MaxCount))
			break
		}
	}

	return subscription
}

func (p *Provider) checkoutLicense(ctx context.Context) (*types.GrantedLicense, error) {
	productSKUField := "ProductSKU"
	var maxResults int32 = 1

	res, err := p.licenseManager.ListReceivedLicenses(ctx, &lm.ListReceivedLicensesInput{
		Filters: []types.Filter{
			{
				Name:   &productSKUField,
				Values: []string{p.projectID},
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
