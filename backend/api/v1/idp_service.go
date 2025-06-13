package v1

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/enterprise"
	"github.com/bytebase/bytebase/backend/plugin/idp/ldap"
	"github.com/bytebase/bytebase/backend/plugin/idp/oauth2"
	"github.com/bytebase/bytebase/backend/plugin/idp/oidc"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// IdentityProviderService implements the identity provider service.
type IdentityProviderService struct {
	v1pb.UnimplementedIdentityProviderServiceServer
	store          *store.Store
	licenseService *enterprise.LicenseService
}

// NewIdentityProviderService creates a new IdentityProviderService.
func NewIdentityProviderService(store *store.Store, licenseService *enterprise.LicenseService) *IdentityProviderService {
	return &IdentityProviderService{
		store:          store,
		licenseService: licenseService,
	}
}

// GetIdentityProvider gets an identity provider.
func (s *IdentityProviderService) GetIdentityProvider(ctx context.Context, request *v1pb.GetIdentityProviderRequest) (*v1pb.IdentityProvider, error) {
	identityProviderMessage, err := s.getIdentityProviderMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	identityProvider, err := convertToIdentityProvider(identityProviderMessage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert identity provider: %v", err)
	}
	return identityProvider, nil
}

// ListIdentityProviders lists all identity providers.
func (s *IdentityProviderService) ListIdentityProviders(ctx context.Context, _ *v1pb.ListIdentityProvidersRequest) (*v1pb.ListIdentityProvidersResponse, error) {
	identityProviders, err := s.store.ListIdentityProviders(ctx, &store.FindIdentityProviderMessage{})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	response := &v1pb.ListIdentityProvidersResponse{}
	for _, identityProviderMessage := range identityProviders {
		identityProvider, err := convertToIdentityProvider(identityProviderMessage)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert identity provider: %v", err)
		}
		response.IdentityProviders = append(response.IdentityProviders, identityProvider)
	}
	return response, nil
}

// CreateIdentityProvider creates an identity provider.
func (s *IdentityProviderService) CreateIdentityProvider(ctx context.Context, request *v1pb.CreateIdentityProviderRequest) (*v1pb.IdentityProvider, error) {
	if err := s.checkFeatureAvailable(request.IdentityProvider); err != nil {
		return nil, err
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get workspace setting: %v", err)
	}
	if setting.ExternalUrl == "" {
		return nil, status.Errorf(codes.FailedPrecondition, setupExternalURLError)
	}

	if request.IdentityProvider == nil {
		return nil, status.Errorf(codes.InvalidArgument, "identity provider must be set")
	}

	if !isValidResourceID(request.IdentityProviderId) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid identity provider ID %v", request.IdentityProviderId)
	}
	if strings.ToLower(request.IdentityProvider.Domain) != request.IdentityProvider.Domain {
		return nil, status.Errorf(codes.InvalidArgument, "domain name must use lower-case")
	}
	if err := validIdentityProviderConfig(request.IdentityProvider.Type, request.IdentityProvider.Config); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	identityProviderMessage := &store.IdentityProviderMessage{
		ResourceID: request.IdentityProviderId,
		Title:      request.IdentityProvider.Title,
		Domain:     request.IdentityProvider.Domain,
		Type:       storepb.IdentityProviderType(request.IdentityProvider.Type),
		Config:     convertIdentityProviderConfigToStore(request.IdentityProvider.GetConfig()),
	}
	if request.ValidateOnly {
		identityProvider, err := convertToIdentityProvider(identityProviderMessage)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert identity provider: %v", err)
		}
		return identityProvider, nil
	}

	identityProviderMessage, err = s.store.CreateIdentityProvider(ctx, identityProviderMessage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create identity provider: %v", err)
	}
	identityProvider, err := convertToIdentityProvider(identityProviderMessage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert identity provider: %v", err)
	}
	return identityProvider, nil
}

// UpdateIdentityProvider updates an identity provider.
func (s *IdentityProviderService) UpdateIdentityProvider(ctx context.Context, request *v1pb.UpdateIdentityProviderRequest) (*v1pb.IdentityProvider, error) {
	if request.IdentityProvider == nil {
		return nil, status.Errorf(codes.InvalidArgument, "identity provider must be set")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}
	if err := s.checkFeatureAvailable(request.IdentityProvider); err != nil {
		return nil, err
	}

	identityProviderMessage, err := s.getIdentityProviderMessage(ctx, request.IdentityProvider.Name)
	if err != nil {
		return nil, err
	}

	patch := &store.UpdateIdentityProviderMessage{
		ResourceID: identityProviderMessage.ResourceID,
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Title = &request.IdentityProvider.Title
		case "domain":
			if strings.ToLower(request.IdentityProvider.Domain) != request.IdentityProvider.Domain {
				return nil, status.Errorf(codes.InvalidArgument, "domain name must use lower-case")
			}
			patch.Domain = &request.IdentityProvider.Domain
		case "config":
			patch.Config = convertIdentityProviderConfigToStore(request.IdentityProvider.Config)
		}
	}
	if patch.Config != nil {
		if err := validIdentityProviderConfig(v1pb.IdentityProviderType(identityProviderMessage.Type), request.IdentityProvider.Config); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		// Don't update client secret if it's empty string.
		switch identityProviderMessage.Type {
		case storepb.IdentityProviderType_OAUTH2:
			if request.IdentityProvider.Config.GetOauth2Config().ClientSecret == "" {
				patch.Config.GetOauth2Config().ClientSecret = identityProviderMessage.Config.GetOauth2Config().ClientSecret
			}
		case storepb.IdentityProviderType_OIDC:
			if request.IdentityProvider.Config.GetOidcConfig().ClientSecret == "" {
				patch.Config.GetOidcConfig().ClientSecret = identityProviderMessage.Config.GetOidcConfig().ClientSecret
			}
		case storepb.IdentityProviderType_LDAP:
			if request.IdentityProvider.Config.GetLdapConfig().BindPassword == "" {
				patch.Config.GetLdapConfig().BindPassword = identityProviderMessage.Config.GetLdapConfig().BindPassword
			}
		}
	}

	identityProviderMessage, err = s.store.UpdateIdentityProvider(ctx, patch)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	identityProvider, err := convertToIdentityProvider(identityProviderMessage)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert identity provider: %v", err))
	}
	return identityProvider, nil
}

// DeleteIdentityProvider deletes an identity provider.
func (s *IdentityProviderService) DeleteIdentityProvider(ctx context.Context, request *v1pb.DeleteIdentityProviderRequest) (*emptypb.Empty, error) {
	identityProvider, err := s.getIdentityProviderMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}

	if err := s.store.DeleteIdentityProvider(ctx, identityProvider.ResourceID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

var googleGitHubDomains = map[string]bool{
	"google.com": true,
	"github.com": true,
}

func (s *IdentityProviderService) checkFeatureAvailable(idp *v1pb.IdentityProvider) error {
	featurePlan := v1pb.PlanFeature_FEATURE_ENTERPRISE_SSO
	if idp.Type == v1pb.IdentityProviderType_OAUTH2 && googleGitHubDomains[idp.Domain] {
		featurePlan = v1pb.PlanFeature_FEATURE_GOOGLE_AND_GITHUB_SSO
	}
	if err := s.licenseService.IsFeatureEnabled(featurePlan); err != nil {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	return nil
}

// TestIdentityProvider tests an identity provider connection.
func (s *IdentityProviderService) TestIdentityProvider(ctx context.Context, request *v1pb.TestIdentityProviderRequest) (*v1pb.TestIdentityProviderResponse, error) {
	identityProvider := request.IdentityProvider
	if identityProvider == nil {
		return nil, status.Errorf(codes.NotFound, "identity provider not found")
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get workspace setting: %v", err)
	}
	if setting.ExternalUrl == "" {
		return nil, status.Errorf(codes.FailedPrecondition, setupExternalURLError)
	}

	switch identityProvider.Type {
	case v1pb.IdentityProviderType_OAUTH2:
		// Find client secret for those existed identity providers.
		if identityProvider.Config.GetOauth2Config().ClientSecret == "" {
			storedIdentityProvider, err := s.getIdentityProviderMessage(ctx, identityProvider.Name)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to find identity provider, error: %s", err.Error())
			}
			if storedIdentityProvider == nil {
				return nil, status.Errorf(codes.Internal, "identity provider %s not found", identityProvider.Name)
			}
			identityProvider.Config.GetOauth2Config().ClientSecret = storedIdentityProvider.Config.GetOauth2Config().ClientSecret
		}
		oauth2Context := request.GetOauth2Context()
		if oauth2Context == nil {
			return nil, status.Errorf(codes.InvalidArgument, "missing OAuth2 context")
		}
		identityProviderConfig := convertIdentityProviderConfigToStore(identityProvider.Config)
		oauth2IdentityProvider, err := oauth2.NewIdentityProvider(identityProviderConfig.GetOauth2Config())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to new oauth2 identity provider")
		}

		redirectURL := fmt.Sprintf("%s/oauth/callback", setting.ExternalUrl)
		token, err := oauth2IdentityProvider.ExchangeToken(ctx, redirectURL, oauth2Context.Code)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to exchange access token, error: %s", err.Error())
		}
		if _, err := oauth2IdentityProvider.UserInfo(token); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to get user info, error: %s", err.Error())
		}
	case v1pb.IdentityProviderType_OIDC:
		// Find client secret for those existed identity providers.
		if identityProvider.Config.GetOidcConfig().ClientSecret == "" {
			storedIdentityProvider, err := s.getIdentityProviderMessage(ctx, identityProvider.Name)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to find identity provider, error: %s", err.Error())
			}
			if storedIdentityProvider == nil {
				return nil, status.Errorf(codes.Internal, "identity provider %s not found", identityProvider.Name)
			}
			identityProvider.Config.GetOidcConfig().ClientSecret = storedIdentityProvider.Config.GetOidcConfig().ClientSecret
		}
		oauth2Context := request.GetOauth2Context()
		if oauth2Context == nil {
			return nil, status.Errorf(codes.InvalidArgument, "missing OAuth2 context")
		}
		identityProviderConfig := convertIdentityProviderConfigToStore(identityProvider.Config)
		oidcIdentityProvider, err := oidc.NewIdentityProvider(ctx, identityProviderConfig.GetOidcConfig())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create new OIDC identity provider: %v", err)
		}

		redirectURL := fmt.Sprintf("%s/oidc/callback", setting.ExternalUrl)
		token, err := oidcIdentityProvider.ExchangeToken(ctx, redirectURL, oauth2Context.Code)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to exchange access token, error: %s", err.Error())
		}
		if _, err := oidcIdentityProvider.UserInfo(ctx, token, ""); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to get user info, error: %s", err.Error())
		}
	case v1pb.IdentityProviderType_LDAP:
		// Retrieve bind password from stored identity provider if not provided.
		if identityProvider.Config.GetLdapConfig().BindPassword == "" {
			storedIdentityProvider, err := s.getIdentityProviderMessage(ctx, identityProvider.Name)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to find identity provider, error: %s", err.Error())
			}
			if storedIdentityProvider == nil {
				return nil, status.Errorf(codes.Internal, "identity provider %s not found", identityProvider.Name)
			}
			identityProvider.Config.GetLdapConfig().BindPassword = storedIdentityProvider.Config.GetLdapConfig().BindPassword
		}
		identityProviderConfig := convertIdentityProviderConfigToStore(identityProvider.Config).GetLdapConfig()
		ldapIdentityProvider, err := ldap.NewIdentityProvider(
			ldap.IdentityProviderConfig{
				Host:             identityProviderConfig.Host,
				Port:             int(identityProviderConfig.Port),
				SkipTLSVerify:    identityProviderConfig.SkipTlsVerify,
				BindDN:           identityProviderConfig.BindDn,
				BindPassword:     identityProviderConfig.BindPassword,
				BaseDN:           identityProviderConfig.BaseDn,
				UserFilter:       identityProviderConfig.UserFilter,
				SecurityProtocol: identityProviderConfig.SecurityProtocol,
				FieldMapping:     identityProviderConfig.FieldMapping,
			},
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create new LDAP identity provider: %v", err)
		}

		conn, err := ldapIdentityProvider.Connect()
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to test connection, error: %s", err.Error())
		}
		_ = conn.Close()
	default:
		return nil, status.Errorf(codes.InvalidArgument, "identity provider type %s not supported", identityProvider.Type.String())
	}
	return &v1pb.TestIdentityProviderResponse{}, nil
}

func (s *IdentityProviderService) getIdentityProviderMessage(ctx context.Context, name string) (*store.IdentityProviderMessage, error) {
	identityProviderID, err := common.GetIdentityProviderID(name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	identityProvider, err := s.store.GetIdentityProvider(ctx, &store.FindIdentityProviderMessage{
		ResourceID: &identityProviderID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if identityProvider == nil {
		return nil, status.Errorf(codes.NotFound, "identity provider %q not found", name)
	}

	return identityProvider, nil
}

func convertToIdentityProvider(identityProvider *store.IdentityProviderMessage) (*v1pb.IdentityProvider, error) {
	identityProviderType := v1pb.IdentityProviderType(identityProvider.Type)
	config, err := convertIdentityProviderConfigFromStore(identityProvider.Config)
	if err != nil {
		return nil, err
	}
	return &v1pb.IdentityProvider{
		Name:   fmt.Sprintf("%s%s", common.IdentityProviderNamePrefix, identityProvider.ResourceID),
		Title:  identityProvider.Title,
		Domain: identityProvider.Domain,
		Type:   identityProviderType,
		Config: config,
	}, nil
}

func convertIdentityProviderConfigFromStore(identityProviderConfig *storepb.IdentityProviderConfig) (*v1pb.IdentityProviderConfig, error) {
	if v := identityProviderConfig.GetOauth2Config(); v != nil {
		fieldMapping := v1pb.FieldMapping{
			Identifier:  v.FieldMapping.Identifier,
			DisplayName: v.FieldMapping.DisplayName,
			Phone:       v.FieldMapping.Phone,
			Groups:      v.FieldMapping.Groups,
		}
		return &v1pb.IdentityProviderConfig{
			Config: &v1pb.IdentityProviderConfig_Oauth2Config{
				Oauth2Config: &v1pb.OAuth2IdentityProviderConfig{
					AuthUrl:       v.AuthUrl,
					TokenUrl:      v.TokenUrl,
					UserInfoUrl:   v.UserInfoUrl,
					ClientId:      v.ClientId,
					ClientSecret:  "", // SECURITY: We do not expose the client secret
					Scopes:        v.Scopes,
					FieldMapping:  &fieldMapping,
					SkipTlsVerify: v.SkipTlsVerify,
					AuthStyle:     v1pb.OAuth2AuthStyle(v.AuthStyle),
				},
			},
		}, nil
	} else if v := identityProviderConfig.GetOidcConfig(); v != nil {
		fieldMapping := v1pb.FieldMapping{
			Identifier:  v.FieldMapping.Identifier,
			DisplayName: v.FieldMapping.DisplayName,
			Phone:       v.FieldMapping.Phone,
			Groups:      v.FieldMapping.Groups,
		}
		oidcConfig := &v1pb.OIDCIdentityProviderConfig{
			Issuer:        v.Issuer,
			ClientId:      v.ClientId,
			ClientSecret:  "", // SECURITY: We do not expose the client secret
			FieldMapping:  &fieldMapping,
			SkipTlsVerify: v.SkipTlsVerify,
			AuthStyle:     v1pb.OAuth2AuthStyle(v.AuthStyle),
			Scopes:        v.Scopes,
			AuthEndpoint:  "", // Leave it empty as it's not stored in the database.
		}

		// Fetch openid configuration to get the auth endpoint.
		openidConfiguration, err := oidc.GetOpenIDConfiguration(v.Issuer)
		if err != nil {
			// Log the error but continue as it's not critical.
			slog.Warn("failed to fetch openid configuration", slog.String("issuer", v.Issuer), log.BBError(err))
		}
		if openidConfiguration != nil {
			// Update the auth endpoint if it's available.
			oidcConfig.AuthEndpoint = openidConfiguration.AuthorizationEndpoint
		}
		return &v1pb.IdentityProviderConfig{
			Config: &v1pb.IdentityProviderConfig_OidcConfig{
				OidcConfig: oidcConfig,
			},
		}, nil
	} else if v := identityProviderConfig.GetLdapConfig(); v != nil {
		fieldMapping := v1pb.FieldMapping{
			Identifier:  v.FieldMapping.Identifier,
			DisplayName: v.FieldMapping.DisplayName,
			Phone:       v.FieldMapping.Phone,
			Groups:      v.FieldMapping.Groups,
		}
		return &v1pb.IdentityProviderConfig{
			Config: &v1pb.IdentityProviderConfig_LdapConfig{
				LdapConfig: &v1pb.LDAPIdentityProviderConfig{
					Host:             v.Host,
					Port:             v.Port,
					SkipTlsVerify:    v.SkipTlsVerify,
					BindDn:           v.BindDn,
					BindPassword:     "", // SECURITY: We do not expose the bind password
					BaseDn:           v.BaseDn,
					UserFilter:       v.UserFilter,
					SecurityProtocol: v1pb.LDAPIdentityProviderConfig_SecurityProtocol(v.SecurityProtocol),
					FieldMapping:     &fieldMapping,
				},
			},
		}, nil
	}
	return nil, nil
}

func convertIdentityProviderConfigToStore(identityProviderConfig *v1pb.IdentityProviderConfig) *storepb.IdentityProviderConfig {
	if v := identityProviderConfig.GetOauth2Config(); v != nil {
		fieldMapping := storepb.FieldMapping{
			Identifier:  v.FieldMapping.Identifier,
			DisplayName: v.FieldMapping.DisplayName,
			Phone:       v.FieldMapping.Phone,
			Groups:      v.FieldMapping.Groups,
		}
		return &storepb.IdentityProviderConfig{
			Config: &storepb.IdentityProviderConfig_Oauth2Config{
				Oauth2Config: &storepb.OAuth2IdentityProviderConfig{
					AuthUrl:       v.AuthUrl,
					TokenUrl:      v.TokenUrl,
					UserInfoUrl:   v.UserInfoUrl,
					ClientId:      v.ClientId,
					ClientSecret:  v.ClientSecret,
					Scopes:        v.Scopes,
					FieldMapping:  &fieldMapping,
					SkipTlsVerify: v.SkipTlsVerify,
					AuthStyle:     storepb.OAuth2AuthStyle(v.AuthStyle),
				},
			},
		}
	} else if v := identityProviderConfig.GetOidcConfig(); v != nil {
		fieldMapping := storepb.FieldMapping{
			Identifier:  v.FieldMapping.Identifier,
			DisplayName: v.FieldMapping.DisplayName,
			Phone:       v.FieldMapping.Phone,
			Groups:      v.FieldMapping.Groups,
		}
		return &storepb.IdentityProviderConfig{
			Config: &storepb.IdentityProviderConfig_OidcConfig{
				OidcConfig: &storepb.OIDCIdentityProviderConfig{
					Issuer:        v.Issuer,
					ClientId:      v.ClientId,
					ClientSecret:  v.ClientSecret,
					FieldMapping:  &fieldMapping,
					SkipTlsVerify: v.SkipTlsVerify,
					AuthStyle:     storepb.OAuth2AuthStyle(v.AuthStyle),
					Scopes:        v.Scopes,
				},
			},
		}
	} else if v := identityProviderConfig.GetLdapConfig(); v != nil {
		fieldMapping := storepb.FieldMapping{
			Identifier:  v.FieldMapping.Identifier,
			DisplayName: v.FieldMapping.DisplayName,
			Phone:       v.FieldMapping.Phone,
			Groups:      v.FieldMapping.Groups,
		}
		return &storepb.IdentityProviderConfig{
			Config: &storepb.IdentityProviderConfig_LdapConfig{
				LdapConfig: &storepb.LDAPIdentityProviderConfig{
					Host:             v.Host,
					Port:             v.Port,
					SkipTlsVerify:    v.SkipTlsVerify,
					BindDn:           v.BindDn,
					BindPassword:     v.BindPassword,
					BaseDn:           v.BaseDn,
					UserFilter:       v.UserFilter,
					SecurityProtocol: storepb.LDAPIdentityProviderConfig_SecurityProtocol(v.SecurityProtocol),
					FieldMapping:     &fieldMapping,
				},
			},
		}
	}
	return nil
}

// validIdentityProviderConfig validates the identity provider's config is a valid JSON.
func validIdentityProviderConfig(identityProviderType v1pb.IdentityProviderType, identityProviderConfig *v1pb.IdentityProviderConfig) error {
	switch identityProviderType {
	case v1pb.IdentityProviderType_OAUTH2:
		if identityProviderConfig.GetOauth2Config() == nil {
			return errors.Errorf("unexpected provider config value")
		}
	case v1pb.IdentityProviderType_OIDC:
		if identityProviderConfig.GetOidcConfig() == nil {
			return errors.Errorf("unexpected provider config value")
		}
	case v1pb.IdentityProviderType_LDAP:
		if identityProviderConfig.GetLdapConfig() == nil {
			return errors.Errorf("unexpected provider config value")
		}
	default:
		return errors.Errorf("unexpected provider type %s", identityProviderType)
	}
	return nil
}
