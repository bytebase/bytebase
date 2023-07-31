package v1

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
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
	licenseService enterpriseAPI.LicenseService
}

// NewIdentityProviderService creates a new IdentityProviderService.
func NewIdentityProviderService(store *store.Store, licenseService enterpriseAPI.LicenseService) *IdentityProviderService {
	return &IdentityProviderService{
		store:          store,
		licenseService: licenseService,
	}
}

// GetIdentityProvider gets an identity provider.
func (s *IdentityProviderService) GetIdentityProvider(ctx context.Context, request *v1pb.GetIdentityProviderRequest) (*v1pb.IdentityProvider, error) {
	identityProvider, err := s.getIdentityProviderMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	return convertToIdentityProvider(identityProvider), nil
}

// ListIdentityProviders lists all identity providers.
func (s *IdentityProviderService) ListIdentityProviders(ctx context.Context, request *v1pb.ListIdentityProvidersRequest) (*v1pb.ListIdentityProvidersResponse, error) {
	identityProviders, err := s.store.ListIdentityProviders(ctx, &store.FindIdentityProviderMessage{ShowDeleted: request.ShowDeleted})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.ListIdentityProvidersResponse{}
	for _, identityProvider := range identityProviders {
		response.IdentityProviders = append(response.IdentityProviders, convertToIdentityProvider(identityProvider))
	}
	return response, nil
}

// CreateIdentityProvider creates an identity provider.
func (s *IdentityProviderService) CreateIdentityProvider(ctx context.Context, request *v1pb.CreateIdentityProviderRequest) (*v1pb.IdentityProvider, error) {
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
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	identityProviderMessage := store.IdentityProviderMessage{
		ResourceID: request.IdentityProviderId,
		Title:      request.IdentityProvider.Title,
		Domain:     request.IdentityProvider.Domain,
		Type:       storepb.IdentityProviderType(request.IdentityProvider.Type),
		Config:     convertIdentityProviderConfigToStore(request.IdentityProvider.GetConfig()),
	}
	identityProvider, err := s.store.CreateIdentityProvider(ctx, &identityProviderMessage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToIdentityProvider(identityProvider), nil
}

// UpdateIdentityProvider updates an identity provider.
func (s *IdentityProviderService) UpdateIdentityProvider(ctx context.Context, request *v1pb.UpdateIdentityProviderRequest) (*v1pb.IdentityProvider, error) {
	if request.IdentityProvider == nil {
		return nil, status.Errorf(codes.InvalidArgument, "identity provider must be set")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	identityProvider, err := s.getIdentityProviderMessage(ctx, request.IdentityProvider.Name)
	if err != nil {
		return nil, err
	}
	if identityProvider.Deleted {
		return nil, status.Errorf(codes.NotFound, "identity provider %q has been deleted", request.IdentityProvider.Name)
	}

	patch := &store.UpdateIdentityProviderMessage{
		ResourceID: identityProvider.ResourceID,
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
		if err := validIdentityProviderConfig(v1pb.IdentityProviderType(identityProvider.Type), request.IdentityProvider.Config); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		// Don't update client secret if it's empty string.
		if identityProvider.Type == storepb.IdentityProviderType_OAUTH2 {
			if request.IdentityProvider.Config.GetOauth2Config().ClientSecret == "" {
				patch.Config.GetOauth2Config().ClientSecret = identityProvider.Config.GetOauth2Config().ClientSecret
			}
		} else if identityProvider.Type == storepb.IdentityProviderType_OIDC {
			if request.IdentityProvider.Config.GetOidcConfig().ClientSecret == "" {
				patch.Config.GetOidcConfig().ClientSecret = identityProvider.Config.GetOidcConfig().ClientSecret
			}
		}
	}

	identityProvider, err = s.store.UpdateIdentityProvider(ctx, patch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToIdentityProvider(identityProvider), nil
}

// DeleteIdentityProvider deletes an identity provider.
func (s *IdentityProviderService) DeleteIdentityProvider(ctx context.Context, request *v1pb.DeleteIdentityProviderRequest) (*emptypb.Empty, error) {
	identityProvider, err := s.getIdentityProviderMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	if identityProvider.Deleted {
		return nil, status.Errorf(codes.NotFound, "identity provider %q has been deleted", request.Name)
	}

	patch := &store.UpdateIdentityProviderMessage{
		ResourceID: identityProvider.ResourceID,
		Delete:     &deletePatch,
	}
	if _, err := s.store.UpdateIdentityProvider(ctx, patch); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

// UndeleteIdentityProvider undeletes an identity provider.
func (s *IdentityProviderService) UndeleteIdentityProvider(ctx context.Context, request *v1pb.UndeleteIdentityProviderRequest) (*v1pb.IdentityProvider, error) {
	identityProvider, err := s.getIdentityProviderMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	if !identityProvider.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "identity provider %q is active", request.Name)
	}

	patch := &store.UpdateIdentityProviderMessage{
		ResourceID: identityProvider.ResourceID,
		Delete:     &undeletePatch,
	}
	identityProvider, err = s.store.UpdateIdentityProvider(ctx, patch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToIdentityProvider(identityProvider), nil
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

	if identityProvider.Type == v1pb.IdentityProviderType_OAUTH2 {
		// Find client secret for those existed identity providers.
		if request.IdentityProvider.Config.GetOauth2Config().ClientSecret == "" {
			storedIdentityProvider, err := s.getIdentityProviderMessage(ctx, request.IdentityProvider.Name)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to find identity provider, error: %s", err.Error())
			}
			if storedIdentityProvider == nil {
				return nil, status.Errorf(codes.Internal, "identity provider %s not found", request.IdentityProvider.Name)
			}
			request.IdentityProvider.Config.GetOauth2Config().ClientSecret = storedIdentityProvider.Config.GetOauth2Config().ClientSecret
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
	} else if identityProvider.Type == v1pb.IdentityProviderType_OIDC {
		// Find client secret for those existed identity providers.
		if request.IdentityProvider.Config.GetOidcConfig().ClientSecret == "" {
			storedIdentityProvider, err := s.getIdentityProviderMessage(ctx, request.IdentityProvider.Name)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to find identity provider, error: %s", err.Error())
			}
			if storedIdentityProvider == nil {
				return nil, status.Errorf(codes.Internal, "identity provider %s not found", request.IdentityProvider.Name)
			}
			request.IdentityProvider.Config.GetOidcConfig().ClientSecret = storedIdentityProvider.Config.GetOidcConfig().ClientSecret
		}
		oauth2Context := request.GetOauth2Context()
		if oauth2Context == nil {
			return nil, status.Errorf(codes.InvalidArgument, "missing OAuth2 context")
		}
		identityProviderConfig := convertIdentityProviderConfigToStore(identityProvider.Config)
		oidcIdentityProvider, err := oidc.NewIdentityProvider(
			ctx,
			oidc.IdentityProviderConfig{
				Issuer:        identityProviderConfig.GetOidcConfig().Issuer,
				ClientID:      identityProviderConfig.GetOidcConfig().ClientId,
				ClientSecret:  identityProviderConfig.GetOidcConfig().ClientSecret,
				FieldMapping:  identityProviderConfig.GetOidcConfig().FieldMapping,
				SkipTLSVerify: identityProviderConfig.GetOidcConfig().SkipTlsVerify,
				AuthStyle:     identityProviderConfig.GetOidcConfig().GetAuthStyle(),
			})
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
	} else {
		return nil, status.Errorf(codes.InvalidArgument, "identity provider type %s not supported", identityProvider.Type.String())
	}
	return &v1pb.TestIdentityProviderResponse{}, nil
}

func (s *IdentityProviderService) getIdentityProviderMessage(ctx context.Context, name string) (*store.IdentityProviderMessage, error) {
	identityProviderID, err := common.GetIdentityProviderID(name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	identityProvider, err := s.store.GetIdentityProvider(ctx, &store.FindIdentityProviderMessage{
		ResourceID: &identityProviderID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if identityProvider == nil {
		return nil, status.Errorf(codes.NotFound, "identity provider %q not found", name)
	}

	return identityProvider, nil
}

func convertToIdentityProvider(identityProvider *store.IdentityProviderMessage) *v1pb.IdentityProvider {
	identityProviderType := v1pb.IdentityProviderType(identityProvider.Type)
	config := convertIdentityProviderConfigFromStore(identityProvider.Config)
	return &v1pb.IdentityProvider{
		Name:   fmt.Sprintf("%s%s", common.IdentityProviderNamePrefix, identityProvider.ResourceID),
		Uid:    fmt.Sprintf("%d", identityProvider.UID),
		State:  convertDeletedToState(identityProvider.Deleted),
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
			Email:       v.FieldMapping.Email,
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
			Email:       v.FieldMapping.Email,
		}
		return &v1pb.IdentityProviderConfig{
			Config: &v1pb.IdentityProviderConfig_OidcConfig{
				OidcConfig: &v1pb.OIDCIdentityProviderConfig{
					Issuer:        v.Issuer,
					ClientId:      v.ClientId,
					ClientSecret:  "", // SECURITY: We do not expose the client secret
					Scopes:        oidc.DefaultScopes,
					FieldMapping:  &fieldMapping,
					SkipTlsVerify: v.SkipTlsVerify,
					AuthStyle:     v1pb.OAuth2AuthStyle(v.AuthStyle),
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
			Email:       v.FieldMapping.Email,
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
			Email:       v.FieldMapping.Email,
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
				},
			},
		}
	} else {
		return nil
	}
}

// validIdentityProviderConfig validates the identity provider's config is a valid JSON.
func validIdentityProviderConfig(identityProviderType v1pb.IdentityProviderType, identityProviderConfig *v1pb.IdentityProviderConfig) error {
	if identityProviderType == v1pb.IdentityProviderType_OAUTH2 {
		if identityProviderConfig.GetOauth2Config() == nil {
			return errors.Errorf("unexpected provider config value")
		}
	} else if identityProviderType == v1pb.IdentityProviderType_OIDC {
		if identityProviderConfig.GetOidcConfig() == nil {
			return errors.Errorf("unexpected provider config value")
		}
	} else {
		return errors.Errorf("unexpected provider type %s", identityProviderType)
	}
	return nil
}
