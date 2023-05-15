package v1

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"

	"embed"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/mail"
	"github.com/bytebase/bytebase/backend/runner/approval"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// SettingService implements the setting service.
type SettingService struct {
	v1pb.UnimplementedSettingServiceServer
	store          *store.Store
	profile        *config.Profile
	licenseService enterpriseAPI.LicenseService
	stateCfg       *state.State
}

// NewSettingService creates a new setting service.
func NewSettingService(
	store *store.Store,
	profile *config.Profile,
	licenseService enterpriseAPI.LicenseService,
	stateCfg *state.State,
) *SettingService {
	return &SettingService{
		store:          store,
		profile:        profile,
		licenseService: licenseService,
		stateCfg:       stateCfg,
	}
}

// Some settings contain secret info so we only return settings that are needed by the client.
var whitelistSettings = []api.SettingName{
	api.SettingBrandingLogo,
	api.SettingAppIM,
	api.SettingWatermark,
	api.SettingWorkspaceProfile,
	api.SettingPluginOpenAIKey,
	api.SettingPluginOpenAIEndpoint,
	api.SettingWorkspaceApproval,
	api.SettingWorkspaceMailDelivery,
}

var (
	//go:embed mail_templates/testmail/template.html
	//go:embed mail_templates/testmail/statics/logo-full.png
	//go:embed mail_templates/testmail/statics/banner.png
	testEmailFs embed.FS
)

// GetSetting gets the setting by name.
func (s *SettingService) GetSetting(ctx context.Context, request *v1pb.GetSettingRequest) (*v1pb.Setting, error) {
	settingName, err := getSettingName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "setting name is invalid: %v", err)
	}
	if settingName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "setting name is empty")
	}
	apiSettingName := api.SettingName(settingName)
	setting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{
		Name: &apiSettingName,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get setting: %v", err)
	}
	if setting == nil {
		return nil, status.Errorf(codes.NotFound, "setting %s not found", settingName)
	}
	// Only return whitelisted setting.
	for _, whitelist := range whitelistSettings {
		if setting.Name == whitelist {
			settingMessage, err := convertToSettingMessage(setting)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to convert setting message: %v", err)
			}
			strippedSettingMessage, err := stripSensitiveData(settingMessage)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to strip sensitive data: %v", err)
			}
			return strippedSettingMessage, nil
		}
	}

	return nil, status.Errorf(codes.InvalidArgument, "setting %s is not whitelisted", settingName)
}

// SetSetting set the setting by name.
func (s *SettingService) SetSetting(ctx context.Context, request *v1pb.SetSettingRequest) (*v1pb.Setting, error) {
	settingName, err := getSettingName(request.Setting.Name)
	if err != nil {
		return nil, err
	}
	if settingName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "setting name is empty")
	}
	if s.profile.IsFeatureUnavailable(settingName) {
		return nil, status.Errorf(codes.InvalidArgument, "feature %s is unavailable in current mode", settingName)
	}

	apiSettingName := api.SettingName(settingName)
	var storeSettingValue string
	switch apiSettingName {
	case api.SettingWorkspaceProfile:
		settingValue := request.Setting.Value.GetWorkspaceProfileSettingValue()
		if settingValue == nil {
			return nil, status.Errorf(codes.InvalidArgument, "setting value for %s is empty", apiSettingName)
		}
		if settingValue.ExternalUrl != "" {
			externalURL, err := common.NormalizeExternalURL(settingValue.ExternalUrl)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid external url: %v", err)
			}
			settingValue.ExternalUrl = externalURL
		}
		storeSettingValue = settingValue.String()
	case api.SettingWorkspaceApproval:
		if !s.licenseService.IsFeatureEnabled(api.FeatureCustomApproval) {
			return nil, status.Errorf(codes.PermissionDenied, api.FeatureCustomApproval.AccessErrorMessage())
		}
		settingValue := request.Setting.Value.GetWorkspaceApprovalSettingValue()
		if settingValue == nil {
			return nil, status.Errorf(codes.InvalidArgument, "setting value for %s is empty", apiSettingName)
		}
		e, err := cel.NewEnv(approval.ApprovalFactors...)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create cel env: %v", err)
		}
		for _, rule := range settingValue.Rules {
			if rule.Expression != nil && rule.Expression.Expr != nil {
				ast := cel.ParsedExprToAst(rule.Expression)
				_, issues := e.Check(ast)
				if issues != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid cel expression: %v, issues: %v", rule.Expression.String(), issues.Err())
				}
			}
			if err := validateApprovalTemplate(rule.Template); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid approval template: %v, err: %v", rule.Template, err)
			}
		}
		storeSettingValue = settingValue.String()
	case api.SettingWorkspaceMailDelivery:
		apiValue := request.Setting.Value.GetSmtpMailDeliverySettingValue()
		if apiValue == nil {
			return nil, status.Errorf(codes.InvalidArgument, "setting value for %s is empty", apiSettingName)
		}
		// We will fill the password read from the store if it is not set.
		if apiValue.Password == nil {
			oldStoreSetting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{
				Name: &apiSettingName,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to get setting %q: %v", apiSettingName, err)
			}
			if oldStoreSetting == nil {
				return nil, status.Errorf(codes.InvalidArgument, "should set the password for the first time")
			}
			oldValue := new(storepb.SMTPMailDeliverySetting)
			if err := protojson.Unmarshal([]byte(oldStoreSetting.Value), oldValue); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value: %v", err)
			}
			apiValue.Password = oldValue.Password
		}
		if request.ValidateOnly {
			if err := s.sendTestEmail(ctx, apiValue); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "failed to validate smtp setting: %v", err)
			}
			apiValue.Password = nil
			return &v1pb.Setting{
				Name: request.Setting.Name,
				Value: &v1pb.Value{
					Value: &v1pb.Value_SmtpMailDeliverySettingValue{
						SmtpMailDeliverySettingValue: apiValue,
					},
				},
			}, nil
		}
		storeMailDeliveryValue := &storepb.SMTPMailDeliverySetting{
			Server:         apiValue.Server,
			Port:           apiValue.Port,
			Encryption:     apiValue.Encryption,
			Authentication: apiValue.Authentication,
			Username:       apiValue.Username,
			Password:       apiValue.Password,
			From:           apiValue.From,
		}
		bytes, err := protojson.Marshal(storeMailDeliveryValue)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting value: %v", err)
		}
		storeSettingValue = string(bytes)
	// TODO: convert setting values and validate
	default:
		storeSettingValue = request.Setting.Value.GetStringValue()
	}
	setting, err := s.store.UpsertSettingV2(ctx, &store.SetSettingMessage{
		Name:  apiSettingName,
		Value: storeSettingValue,
	}, ctx.Value(common.PrincipalIDContextKey).(int))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set setting: %v", err)
	}

	settingMessage, err := convertToSettingMessage(setting)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert setting message: %v", err)
	}
	strippedSettingMessage, err := stripSensitiveData(settingMessage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to strip sensitive data: %v", err)
	}
	return strippedSettingMessage, nil
}

func convertToSettingMessage(setting *store.SettingMessage) (*v1pb.Setting, error) {
	switch setting.Name {
	case api.SettingWorkspaceMailDelivery:
		storeValue := new(storepb.SMTPMailDeliverySetting)
		if err := protojson.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value: %v", err)
		}
		return &v1pb.Setting{
			Name: settingNamePrefix + string(setting.Name),
			Value: &v1pb.Value{
				Value: &v1pb.Value_SmtpMailDeliverySettingValue{
					SmtpMailDeliverySettingValue: &storepb.SMTPMailDeliverySetting{
						Server:         storeValue.Server,
						Port:           storeValue.Port,
						Encryption:     storeValue.Encryption,
						Authentication: storeValue.Authentication,
						Ca:             storeValue.Ca,
						Key:            storeValue.Key,
						Cert:           storeValue.Cert,
						Username:       storeValue.Username,
						Password:       storeValue.Password,
						From:           storeValue.From,
					},
				},
			},
		}, nil
	// TODO: convert setting values
	default:
		return &v1pb.Setting{
			Name: settingNamePrefix + string(setting.Name),
			Value: &v1pb.Value{
				Value: &v1pb.Value_StringValue{
					StringValue: setting.Value,
				},
			},
		}, nil
	}
}

func validateApprovalTemplate(template *storepb.ApprovalTemplate) error {
	if template.Flow == nil {
		return errors.Errorf("approval template cannot be nil")
	}
	if len(template.Flow.Steps) == 0 {
		return errors.Errorf("approval template cannot have 0 step")
	}
	for _, step := range template.Flow.Steps {
		if step.Type != storepb.ApprovalStep_ANY {
			return errors.Errorf("invalid approval step type: %v", step.Type)
		}
		if len(step.Nodes) != 1 {
			return errors.Errorf("expect 1 node in approval step, got: %v", len(step.Nodes))
		}
	}
	return nil
}

func (s *SettingService) sendTestEmail(ctx context.Context, value *storepb.SMTPMailDeliverySetting) error {
	if value.Password == nil {
		return status.Errorf(codes.InvalidArgument, "password is required when sending test email")
	}
	if value.To == nil {
		return status.Errorf(codes.InvalidArgument, "to is required when sending test email")
	}
	if value.From == "" {
		return status.Errorf(codes.InvalidArgument, "from is required when sending test email")
	}

	consoleRedirectURL := "www.bytebase.com"
	workspaceProfileSettingName := api.SettingWorkspaceProfile
	setting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{Name: &workspaceProfileSettingName})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get workspace profile setting: %v", err)
	}
	if setting != nil {
		settingValue := new(storepb.WorkspaceProfileSetting)
		if err := protojson.Unmarshal([]byte(setting.Value), settingValue); err != nil {
			return status.Errorf(codes.Internal, "failed to unmarshal setting value: %v", err)
		}
		if settingValue.ExternalUrl != "" {
			consoleRedirectURL = settingValue.ExternalUrl
		}
	}

	email := mail.NewEmailMsg()

	logoFull, err := testEmailFs.ReadFile("mail_templates/testmail/statics/logo-full.png")
	if err != nil {
		return status.Errorf(codes.Internal, "failed to read logo-full.png: %v", err)
	}
	banner, err := testEmailFs.ReadFile("mail_templates/testmail/statics/banner.png")
	if err != nil {
		return status.Errorf(codes.Internal, "failed to read banner.png: %v", err)
	}
	mailHTMLBody, err := testEmailFs.ReadFile("mail_templates/testmail/template.html")
	if err != nil {
		return status.Errorf(codes.Internal, "failed to read template.html: %v", err)
	}
	logoFullReader := bytes.NewReader(logoFull)
	bannerReader := bytes.NewReader(banner)
	logoFullFileName, err := email.Attach(logoFullReader, "logo-full.png", "image/png")
	if err != nil {
		return status.Errorf(codes.Internal, "failed to attach logo-full.png: %v", err)
	}
	bannerFileName, err := email.Attach(bannerReader, "banner.png", "image/png")
	if err != nil {
		return status.Errorf(codes.Internal, "failed to attach banner.png: %v", err)
	}

	polishHTMLBody := prepareTestMailContent(string(mailHTMLBody), consoleRedirectURL, logoFullFileName, bannerFileName)
	email.SetFrom(fmt.Sprintf("Bytebase <%s>", value.From)).AddTo(*value.To).SetSubject("Bytebase mail server test").SetBody(polishHTMLBody)
	client := mail.NewSMTPClient(value.Server, int(value.Port))
	client.SetAuthType(convertToMailSMTPAuthType(value.Authentication)).
		SetAuthCredentials(value.Username, *value.Password).
		SetEncryptionType(convertToMailSMTPEncryptionType(value.Encryption))

	if err := client.SendMail(email); err != nil {
		return status.Errorf(codes.Internal, "failed to send test email: %v", err)
	}
	return nil
}

func prepareTestMailContent(htmlTemplate, consoleRedirectURL, logoContentID, bannerContentID string) string {
	testEmailContent := strings.ReplaceAll(htmlTemplate, "{{BYTEBASE_LOGO_URL}}", fmt.Sprintf("cid:%s", logoContentID))
	testEmailContent = strings.ReplaceAll(testEmailContent, "{{BYTEBASE_BANNER_URL}}", fmt.Sprintf("cid:%s", bannerContentID))
	testEmailContent = strings.ReplaceAll(testEmailContent, "{{BYTEBASE_CONSOLE_REDIRECT_URL}}", consoleRedirectURL)
	return testEmailContent
}

func convertToMailSMTPAuthType(authType storepb.SMTPMailDeliverySetting_Authentication) mail.SMTPAuthType {
	switch authType {
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_NONE:
		return mail.SMTPAuthTypeNone
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_PLAIN:
		return mail.SMTPAuthTypePlain
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_LOGIN:
		return mail.SMTPAuthTypeLogin
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_CRAM_MD5:
		return mail.SMTPAuthTypeCRAMMD5
	}
	return mail.SMTPAuthTypeNone
}

func convertToMailSMTPEncryptionType(encryptionType storepb.SMTPMailDeliverySetting_Encryption) mail.SMTPEncryptionType {
	switch encryptionType {
	case storepb.SMTPMailDeliverySetting_ENCRYPTION_NONE:
		return mail.SMTPEncryptionTypeNone
	case storepb.SMTPMailDeliverySetting_ENCRYPTION_STARTTLS:
		return mail.SMTPEncryptionTypeSTARTTLS
	case storepb.SMTPMailDeliverySetting_ENCRYPTION_SSL_TLS:
		return mail.SMTPEncryptionTypeSSLTLS
	}
	return mail.SMTPEncryptionTypeNone
}

// stripSensitiveData strips the sensitive data like password from the setting.value.
func stripSensitiveData(setting *v1pb.Setting) (*v1pb.Setting, error) {
	settingName, err := getSettingName(setting.Name)
	if err != nil {
		return nil, err
	}
	apiSettingName := api.SettingName(settingName)
	switch apiSettingName {
	case api.SettingWorkspaceMailDelivery:
		mailDeliveryValue, ok := setting.Value.Value.(*v1pb.Value_SmtpMailDeliverySettingValue)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "invalid setting value type: %T", setting.Value.Value)
		}
		mailDeliveryValue.SmtpMailDeliverySettingValue.Password = nil
		mailDeliveryValue.SmtpMailDeliverySettingValue.Ca = nil
		mailDeliveryValue.SmtpMailDeliverySettingValue.Cert = nil
		mailDeliveryValue.SmtpMailDeliverySettingValue.Key = nil
		setting.Value.Value = mailDeliveryValue
	default:
	}
	return setting, nil
}
