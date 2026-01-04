package v1

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/idp/ldap"
	"github.com/bytebase/bytebase/backend/plugin/idp/oauth2"
	"github.com/bytebase/bytebase/backend/plugin/idp/oidc"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// IdentityProviderService implements the identity provider service.
type IdentityProviderService struct {
	v1connect.UnimplementedIdentityProviderServiceHandler
	store          *store.Store
	licenseService *enterprise.LicenseService
	profile        *config.Profile
}

// NewIdentityProviderService creates a new IdentityProviderService.
func NewIdentityProviderService(store *store.Store, licenseService *enterprise.LicenseService, profile *config.Profile) *IdentityProviderService {
	return &IdentityProviderService{
		store:          store,
		licenseService: licenseService,
		profile:        profile,
	}
}

// GetIdentityProvider gets an identity provider.
func (s *IdentityProviderService) GetIdentityProvider(ctx context.Context, req *connect.Request[v1pb.GetIdentityProviderRequest]) (*connect.Response[v1pb.IdentityProvider], error) {
	identityProviderMessage, err := s.getIdentityProviderMessage(ctx, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	identityProvider := convertToIdentityProvider(identityProviderMessage)
	return connect.NewResponse(identityProvider), nil
}

// ListIdentityProviders lists all identity providers.
func (s *IdentityProviderService) ListIdentityProviders(ctx context.Context, _ *connect.Request[v1pb.ListIdentityProvidersRequest]) (*connect.Response[v1pb.ListIdentityProvidersResponse], error) {
	identityProviders, err := s.store.ListIdentityProviders(ctx, &store.FindIdentityProviderMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &v1pb.ListIdentityProvidersResponse{}
	for _, identityProviderMessage := range identityProviders {
		identityProvider := convertToIdentityProvider(identityProviderMessage)
		response.IdentityProviders = append(response.IdentityProviders, identityProvider)
	}
	return connect.NewResponse(response), nil
}

// CreateIdentityProvider creates an identity provider.
func (s *IdentityProviderService) CreateIdentityProvider(ctx context.Context, req *connect.Request[v1pb.CreateIdentityProviderRequest]) (*connect.Response[v1pb.IdentityProvider], error) {
	if err := s.checkFeatureAvailable(req.Msg.IdentityProvider); err != nil {
		return nil, err
	}

	if _, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile); err != nil {
		return nil, err
	}

	if req.Msg.IdentityProvider == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("identity provider must be set"))
	}

	if !isValidResourceID(req.Msg.IdentityProviderId) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid identity provider ID %v", req.Msg.IdentityProviderId))
	}
	if strings.ToLower(req.Msg.IdentityProvider.Domain) != req.Msg.IdentityProvider.Domain {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("domain name must use lower-case"))
	}
	if err := validIdentityProviderConfig(req.Msg.IdentityProvider.Type, req.Msg.IdentityProvider.Config); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	identityProviderMessage := &store.IdentityProviderMessage{
		ResourceID: req.Msg.IdentityProviderId,
		Title:      req.Msg.IdentityProvider.Title,
		Domain:     req.Msg.IdentityProvider.Domain,
		Type:       storepb.IdentityProviderType(req.Msg.IdentityProvider.Type),
		Config:     convertIdentityProviderConfigToStore(req.Msg.IdentityProvider.GetConfig()),
	}
	if req.Msg.ValidateOnly {
		identityProvider := convertToIdentityProvider(identityProviderMessage)
		return connect.NewResponse(identityProvider), nil
	}

	identityProviderMessage, err := s.store.CreateIdentityProvider(ctx, identityProviderMessage)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create identity provider"))
	}
	identityProvider := convertToIdentityProvider(identityProviderMessage)
	return connect.NewResponse(identityProvider), nil
}

// UpdateIdentityProvider updates an identity provider.
func (s *IdentityProviderService) UpdateIdentityProvider(ctx context.Context, req *connect.Request[v1pb.UpdateIdentityProviderRequest]) (*connect.Response[v1pb.IdentityProvider], error) {
	if req.Msg.IdentityProvider == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("identity provider must be set"))
	}
	if req.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update_mask must be set"))
	}
	if err := s.checkFeatureAvailable(req.Msg.IdentityProvider); err != nil {
		return nil, err
	}

	identityProviderID, err := common.GetIdentityProviderID(req.Msg.IdentityProvider.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	identityProviderMessage, err := s.store.GetIdentityProvider(ctx, &store.FindIdentityProviderMessage{
		ResourceID: &identityProviderID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if identityProviderMessage == nil {
		if req.Msg.AllowMissing {
			return s.CreateIdentityProvider(ctx, connect.NewRequest(&v1pb.CreateIdentityProviderRequest{
				IdentityProviderId: identityProviderID,
				IdentityProvider:   req.Msg.IdentityProvider,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("identity provider %q not found", req.Msg.IdentityProvider.Name))
	}

	patch := &store.UpdateIdentityProviderMessage{
		ResourceID: identityProviderMessage.ResourceID,
	}
	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Title = &req.Msg.IdentityProvider.Title
		case "domain":
			if strings.ToLower(req.Msg.IdentityProvider.Domain) != req.Msg.IdentityProvider.Domain {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("domain name must use lower-case"))
			}
			patch.Domain = &req.Msg.IdentityProvider.Domain
		case "config":
			patch.Config = convertIdentityProviderConfigToStore(req.Msg.IdentityProvider.Config)
		default:
		}
	}
	if patch.Config != nil {
		if err := validIdentityProviderConfig(v1pb.IdentityProviderType(identityProviderMessage.Type), req.Msg.IdentityProvider.Config); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		// Don't update client secret if it's empty string.
		switch identityProviderMessage.Type {
		case storepb.IdentityProviderType_OAUTH2:
			if req.Msg.IdentityProvider.Config.GetOauth2Config().ClientSecret == "" {
				patch.Config.GetOauth2Config().ClientSecret = identityProviderMessage.Config.GetOauth2Config().ClientSecret
			}
		case storepb.IdentityProviderType_OIDC:
			if req.Msg.IdentityProvider.Config.GetOidcConfig().ClientSecret == "" {
				patch.Config.GetOidcConfig().ClientSecret = identityProviderMessage.Config.GetOidcConfig().ClientSecret
			}
		case storepb.IdentityProviderType_LDAP:
			if req.Msg.IdentityProvider.Config.GetLdapConfig().BindPassword == "" {
				patch.Config.GetLdapConfig().BindPassword = identityProviderMessage.Config.GetLdapConfig().BindPassword
			}
		default:
		}
	}

	identityProviderMessage, err = s.store.UpdateIdentityProvider(ctx, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	identityProvider := convertToIdentityProvider(identityProviderMessage)
	return connect.NewResponse(identityProvider), nil
}

// DeleteIdentityProvider deletes an identity provider.
func (s *IdentityProviderService) DeleteIdentityProvider(ctx context.Context, req *connect.Request[v1pb.DeleteIdentityProviderRequest]) (*connect.Response[emptypb.Empty], error) {
	identityProvider, err := s.getIdentityProviderMessage(ctx, req.Msg.Name)
	if err != nil {
		return nil, err
	}

	if err := s.store.DeleteIdentityProvider(ctx, identityProvider.ResourceID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
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
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return nil
}

// TestIdentityProvider tests an identity provider connection.
func (s *IdentityProviderService) TestIdentityProvider(ctx context.Context, req *connect.Request[v1pb.TestIdentityProviderRequest]) (*connect.Response[v1pb.TestIdentityProviderResponse], error) {
	identityProvider := req.Msg.IdentityProvider
	if identityProvider == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("identity provider not found"))
	}

	externalURL, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile)
	if err != nil {
		return nil, err
	}

	switch identityProvider.Type {
	case v1pb.IdentityProviderType_OAUTH2:
		// Find client secret for those existed identity providers.
		if identityProvider.Config.GetOauth2Config().ClientSecret == "" {
			storedIdentityProvider, err := s.getIdentityProviderMessage(ctx, identityProvider.Name)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find identity provider, error: %s", err.Error()))
			}
			if storedIdentityProvider == nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("identity provider %s not found", identityProvider.Name))
			}
			identityProvider.Config.GetOauth2Config().ClientSecret = storedIdentityProvider.Config.GetOauth2Config().ClientSecret
		}
		oauth2Context := req.Msg.GetOauth2Context()
		if oauth2Context == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("missing OAuth2 context"))
		}
		identityProviderConfig := convertIdentityProviderConfigToStore(identityProvider.Config)
		oauth2IdentityProvider, err := oauth2.NewIdentityProvider(identityProviderConfig.GetOauth2Config())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to new oauth2 identity provider"))
		}

		redirectURL := fmt.Sprintf("%s/oauth/callback", externalURL)
		token, err := oauth2IdentityProvider.ExchangeToken(ctx, redirectURL, oauth2Context.Code)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to exchange access token, error: %s", err.Error()))
		}
		userInfo, claims, err := oauth2IdentityProvider.UserInfo(token)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to get user info, error: %s", err.Error()))
		}

		claimsMap := make(map[string]string)
		for key, value := range claims {
			claimsMap[key] = fmt.Sprintf("%v", value)
		}
		userInfoMap := make(map[string]string)
		if userInfo != nil {
			if userInfo.Identifier != "" {
				userInfoMap["email"] = userInfo.Identifier
			}
			if userInfo.DisplayName != "" {
				userInfoMap["title"] = userInfo.DisplayName
			}
			if userInfo.Phone != "" {
				userInfoMap["phone"] = userInfo.Phone
			}
		}
		return connect.NewResponse(&v1pb.TestIdentityProviderResponse{Claims: claimsMap, UserInfo: userInfoMap}), nil
	case v1pb.IdentityProviderType_OIDC:
		// Find client secret for those existed identity providers.
		if identityProvider.Config.GetOidcConfig().ClientSecret == "" {
			storedIdentityProvider, err := s.getIdentityProviderMessage(ctx, identityProvider.Name)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find identity provider, error: %s", err.Error()))
			}
			if storedIdentityProvider == nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("identity provider %s not found", identityProvider.Name))
			}
			identityProvider.Config.GetOidcConfig().ClientSecret = storedIdentityProvider.Config.GetOidcConfig().ClientSecret
		}

		oidcContext := req.Msg.GetOidcContext()
		if oidcContext == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("missing OIDC context"))
		}

		identityProviderConfig := convertIdentityProviderConfigToStore(identityProvider.Config)
		oidcIdentityProvider, err := oidc.NewIdentityProvider(ctx, identityProviderConfig.GetOidcConfig())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create new OIDC identity provider"))
		}

		redirectURL := fmt.Sprintf("%s/oidc/callback", externalURL)
		token, err := oidcIdentityProvider.ExchangeToken(ctx, redirectURL, oidcContext.Code)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to exchange access token, error: %s", err.Error()))
		}
		userInfo, claims, err := oidcIdentityProvider.UserInfo(ctx, token, "")
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to get user info, error: %s", err.Error()))
		}

		claimsMap := make(map[string]string)
		for key, value := range claims {
			claimsMap[key] = fmt.Sprintf("%v", value)
		}
		userInfoMap := make(map[string]string)
		if userInfo != nil {
			if userInfo.Identifier != "" {
				userInfoMap["email"] = userInfo.Identifier
			}
			if userInfo.DisplayName != "" {
				userInfoMap["title"] = userInfo.DisplayName
			}
			if userInfo.Phone != "" {
				userInfoMap["phone"] = userInfo.Phone
			}
			if userInfo.Groups != nil {
				userInfoMap["groups"] = fmt.Sprintf("%v", userInfo.Groups)
			}
		}
		return connect.NewResponse(&v1pb.TestIdentityProviderResponse{Claims: claimsMap, UserInfo: userInfoMap}), nil
	case v1pb.IdentityProviderType_LDAP:
		// Retrieve bind password from stored identity provider if not provided.
		if identityProvider.Config.GetLdapConfig().BindPassword == "" {
			storedIdentityProvider, err := s.getIdentityProviderMessage(ctx, identityProvider.Name)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find identity provider, error: %s", err.Error()))
			}
			if storedIdentityProvider == nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("identity provider %s not found", identityProvider.Name))
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
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create new LDAP identity provider"))
		}

		conn, err := ldapIdentityProvider.Connect()
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to test connection, error: %s", err.Error()))
		}
		_ = conn.Close()

		// LDAP cannot return claims without username and password so we return an empty claims map.
		return connect.NewResponse(&v1pb.TestIdentityProviderResponse{Claims: make(map[string]string)}), nil
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("identity provider type %s not supported", identityProvider.Type.String()))
	}
}

func (s *IdentityProviderService) getIdentityProviderMessage(ctx context.Context, name string) (*store.IdentityProviderMessage, error) {
	identityProviderID, err := common.GetIdentityProviderID(name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	identityProvider, err := s.store.GetIdentityProvider(ctx, &store.FindIdentityProviderMessage{
		ResourceID: &identityProviderID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if identityProvider == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("identity provider %q not found", name))
	}

	return identityProvider, nil
}

func convertToIdentityProvider(identityProvider *store.IdentityProviderMessage) *v1pb.IdentityProvider {
	identityProviderType := v1pb.IdentityProviderType(identityProvider.Type)
	config := convertIdentityProviderConfigFromStore(identityProvider.Config)
	return &v1pb.IdentityProvider{
		Name:   fmt.Sprintf("%s%s", common.IdentityProviderNamePrefix, identityProvider.ResourceID),
		Title:  identityProvider.Title,
		Domain: identityProvider.Domain,
		Type:   identityProviderType,
		Config: config,
	}
}

func convertIdentityProviderConfigFromStore(identityProviderConfig *storepb.IdentityProviderConfig) *v1pb.IdentityProviderConfig {
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
		}
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
		openidConfiguration, err := oidc.GetOpenIDConfiguration(v.Issuer, v.SkipTlsVerify)
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
		}
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
		}
	}
	return nil
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
