import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import {
  ArrowRight,
  Building,
  Database,
  GitBranch,
  GitFork,
  Globe,
  Info,
  Key,
  Plus,
  ShieldCheck,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { identityProviderServiceClientConnect } from "@/connect";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import {
  ResourceIdField,
  type ResourceIdFieldRef,
} from "@/react/components/ResourceIdField";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { WORKSPACE_ROUTE_IDENTITY_PROVIDER_DETAIL } from "@/router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useActuatorV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { useIdentityProviderStore } from "@/store/modules/idp";
import {
  getIdentityProviderResourceId,
  idpNamePrefix,
} from "@/store/modules/v1/common";
import type { OAuthWindowEventPayload } from "@/types";
import type {
  IdentityProvider,
  LDAPIdentityProviderConfig,
  OAuth2IdentityProviderConfig,
  OIDCIdentityProviderConfig,
  TestIdentityProviderResponse,
} from "@/types/proto-es/v1/idp_service_pb";
import {
  CreateIdentityProviderRequestSchema,
  FieldMappingSchema,
  IdentityProviderConfigSchema,
  IdentityProviderSchema,
  IdentityProviderType,
  LDAPIdentityProviderConfig_SecurityProtocol,
  LDAPIdentityProviderConfigSchema,
  OAuth2AuthStyle,
  OAuth2IdentityProviderConfigSchema,
  OIDCIdentityProviderConfigSchema,
  TestIdentityProviderRequestSchema,
} from "@/types/proto-es/v1/idp_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  hasWorkspacePermissionV2,
  identityProviderTypeToString,
  openWindowForSSO,
} from "@/utils";

// ============================================================
// Escape key stack
// ============================================================

const escapeStack: (() => void)[] = [];

function useEscapeKey(onEscape: () => void) {
  useEffect(() => {
    escapeStack.push(onEscape);
    const handler = (e: KeyboardEvent) => {
      if (
        e.key === "Escape" &&
        escapeStack[escapeStack.length - 1] === onEscape
      ) {
        onEscape();
      }
    };
    document.addEventListener("keydown", handler);
    return () => {
      document.removeEventListener("keydown", handler);
      const idx = escapeStack.lastIndexOf(onEscape);
      if (idx >= 0) escapeStack.splice(idx, 1);
    };
  }, [onEscape]);
}

// ============================================================
// Types
// ============================================================

interface OAuth2Template {
  title: string;
  domain: string;
  domainDisabled?: boolean;
  feature: PlanFeature;
  config: OAuth2IdentityProviderConfig;
}

interface FieldMappingState {
  identifier: string;
  displayName: string;
  phone: string;
  groups: string;
}

// ============================================================
// ExternalURLInfo
// ============================================================

function ExternalURLInfo({ type }: { type: IdentityProviderType }) {
  const { t } = useTranslation();
  const actuatorStore = useActuatorV1Store();
  const externalUrl = useVueState(
    () => actuatorStore.serverInfo?.externalUrl ?? ""
  );

  const redirectUrl = useMemo(() => {
    const url = externalUrl || window.origin;
    if (type === IdentityProviderType.OAUTH2) {
      return `${url}/oauth/callback`;
    }
    if (type === IdentityProviderType.OIDC) {
      return `${url}/oidc/callback`;
    }
    return "";
  }, [externalUrl, type]);

  if (!redirectUrl) return null;

  const handleCopy = () => {
    navigator.clipboard.writeText(redirectUrl);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.copied"),
    });
  };

  return (
    <div className="p-4 rounded-sm border border-gray-200 bg-gray-50">
      <div className="flex items-start gap-x-3">
        <Info className="w-5 h-5 text-blue-500 mt-0.5 shrink-0" />
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-gray-900 mb-2">
            {t("settings.sso.form.identity-provider-needed-information")}
          </p>
          <p className="text-sm text-gray-600 mb-3">
            {t("settings.sso.form.redirect-url-description")}
          </p>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              {t("settings.sso.form.redirect-url")}
            </label>
            <div className="flex items-center gap-x-2">
              <Input
                value={redirectUrl}
                readOnly
                className="flex-1 font-mono"
              />
              <Button variant="outline" size="sm" onClick={handleCopy}>
                {t("common.copy")}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ============================================================
// TestConnectionResultDialog
// ============================================================

function TestConnectionResultDialog({
  response,
  onClose,
}: {
  response: TestIdentityProviderResponse;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  useEscapeKey(onClose);

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/50"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="bg-white rounded-sm shadow-lg w-[32rem] max-h-[80vh] overflow-auto p-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-x-2">
            <div className="w-6 h-6 text-green-500">&#10003;</div>
            <h3 className="text-lg font-medium">
              {t("identity-provider.test-connection-success")}
            </h3>
          </div>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="h-4 w-4" />
          </Button>
        </div>

        <div className="flex flex-col gap-y-4">
          <p className="text-sm text-control-light">
            {t("identity-provider.userinfo-description")}
          </p>
          <div className="bg-gray-50 rounded-xs p-4">
            <div className="flex flex-col gap-y-1">
              {Object.entries(response.userInfo).map(([key, value]) => (
                <div
                  key={key}
                  className="grid grid-cols-3 gap-2 py-1 border-b border-gray-200 last:border-b-0"
                >
                  <div
                    className="text-sm font-medium text-control truncate"
                    title={key}
                  >
                    {key}
                  </div>
                  <div
                    className="col-span-2 text-sm text-main break-all"
                    title={value}
                  >
                    {value}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <p className="text-sm text-control-light">
            {t("identity-provider.claims-description")}
          </p>
          <div className="bg-gray-50 rounded-xs p-4">
            {Object.keys(response.claims).length === 0 ? (
              <div className="text-sm text-control-light italic">
                {t("identity-provider.no-claims")}
              </div>
            ) : (
              <div className="flex flex-col gap-y-1">
                {Object.entries(response.claims).map(([key, value]) => (
                  <div
                    key={key}
                    className="grid grid-cols-3 gap-2 py-1 border-b border-gray-200 last:border-b-0"
                  >
                    <div
                      className="text-sm font-medium text-control truncate"
                      title={key}
                    >
                      {key}
                    </div>
                    <div
                      className="col-span-2 text-sm text-main break-all"
                      title={value}
                    >
                      {value}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        <div className="flex justify-end mt-4">
          <Button onClick={onClose}>{t("common.close")}</Button>
        </div>
      </div>
    </div>
  );
}

// ============================================================
// TestConnectionButton
// ============================================================

function TestConnectionButton({
  idp,
  disabled,
  isCreating,
}: {
  idp: IdentityProvider;
  disabled: boolean;
  isCreating?: boolean;
}) {
  const { t } = useTranslation();
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] =
    useState<TestIdentityProviderResponse | null>(null);
  const currentEventNameRef = useRef("");
  const idpRef = useRef(idp);
  idpRef.current = idp;
  const testingRef = useRef(false);

  // Stable event handler that reads latest state via refs
  const handleOAuthEventRef = useRef(async (event: Event) => {
    if (testingRef.current) return;
    const payload = (event as CustomEvent).detail as OAuthWindowEventPayload;
    if (payload.error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Request error occurred",
        description: payload.error,
      });
      return;
    }

    try {
      testingRef.current = true;
      setTesting(true);
      const currentIdp = idpRef.current;
      const isOidc = currentIdp.type === IdentityProviderType.OIDC;
      const request = create(TestIdentityProviderRequestSchema, {
        identityProvider: currentIdp,
        context: isOidc
          ? { case: "oidcContext", value: { code: payload.code } }
          : { case: "oauth2Context", value: { code: payload.code } },
      });
      const response =
        await identityProviderServiceClientConnect.testIdentityProvider(
          request
        );
      setTestResult(response);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Request error occurred",
        description: (error as ConnectError).message,
      });
    } finally {
      testingRef.current = false;
      setTesting(false);
    }
  });

  useEffect(() => {
    return () => {
      if (currentEventNameRef.current) {
        window.removeEventListener(
          currentEventNameRef.current,
          handleOAuthEventRef.current as EventListener,
          false
        );
      }
    };
  }, []);

  const testConnection = async () => {
    if (testingRef.current) return;

    if (
      idp.type === IdentityProviderType.OAUTH2 ||
      idp.type === IdentityProviderType.OIDC
    ) {
      let idpForTesting: IdentityProvider = idp;
      if (isCreating && idp.type === IdentityProviderType.OIDC) {
        const request = create(CreateIdentityProviderRequestSchema, {
          identityProviderId: idp.name,
          identityProvider: idp,
          validateOnly: true,
        });
        const response =
          await identityProviderServiceClientConnect.createIdentityProvider(
            request
          );
        idpForTesting = response;
      }

      const eventName = `bb.oauth.signin.${idpForTesting.name}`;
      if (currentEventNameRef.current) {
        window.removeEventListener(
          currentEventNameRef.current,
          handleOAuthEventRef.current as EventListener,
          false
        );
      }
      window.addEventListener(
        eventName,
        handleOAuthEventRef.current as EventListener,
        false
      );
      currentEventNameRef.current = eventName;

      try {
        await openWindowForSSO(idpForTesting);
      } catch (error) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: "Request error occurred",
          description: (error as ConnectError).message,
        });
      }
    } else if (idp.type === IdentityProviderType.LDAP) {
      try {
        testingRef.current = true;
        setTesting(true);
        const request = create(TestIdentityProviderRequestSchema, {
          identityProvider: idp,
        });
        const response =
          await identityProviderServiceClientConnect.testIdentityProvider(
            request
          );
        setTestResult(response);
      } catch (error) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: "Request error occurred",
          description: (error as ConnectError).message,
        });
      } finally {
        testingRef.current = false;
        setTesting(false);
      }
    }
  };

  return (
    <>
      <Button
        variant="outline"
        disabled={disabled || testing}
        onClick={testConnection}
      >
        {t("identity-provider.test-connection")}
      </Button>
      {testResult && (
        <TestConnectionResultDialog
          response={testResult}
          onClose={() => setTestResult(null)}
        />
      )}
    </>
  );
}

// ============================================================
// ProviderConfigForm
// ============================================================

function ProviderConfigForm({
  providerType,
  configForOAuth2,
  configForOIDC,
  configForLDAP,
  scopesString,
  onUpdateOAuth2,
  onUpdateOIDC,
  onUpdateLDAP,
  onUpdateScopes,
}: {
  providerType: IdentityProviderType;
  configForOAuth2: OAuth2IdentityProviderConfig;
  configForOIDC: OIDCIdentityProviderConfig;
  configForLDAP: LDAPIdentityProviderConfig;
  scopesString: string;
  onUpdateOAuth2: (config: OAuth2IdentityProviderConfig) => void;
  onUpdateOIDC: (config: OIDCIdentityProviderConfig) => void;
  onUpdateLDAP: (config: LDAPIdentityProviderConfig) => void;
  onUpdateScopes: (scopes: string) => void;
}) {
  const { t } = useTranslation();

  if (providerType === IdentityProviderType.OAUTH2) {
    return (
      <div className="flex flex-col gap-y-6">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <label className="block text-base font-semibold text-gray-800 mb-2">
              Client ID <span className="text-error">*</span>
            </label>
            <Input
              value={configForOAuth2.clientId}
              onChange={(e) =>
                onUpdateOAuth2({
                  ...configForOAuth2,
                  clientId: e.target.value,
                })
              }
              placeholder="e.g. 6655asd77895265aa110ac0d3"
            />
          </div>
          <div>
            <label className="block text-base font-semibold text-gray-800 mb-2">
              Client Secret <span className="text-error">*</span>
            </label>
            <Input
              type="password"
              value={configForOAuth2.clientSecret}
              onChange={(e) =>
                onUpdateOAuth2({
                  ...configForOAuth2,
                  clientSecret: e.target.value,
                })
              }
              placeholder="e.g. 5bbezxc3972ca304de70c5d70a6aa932asd8"
            />
          </div>
        </div>

        <div>
          <label className="block text-base font-semibold text-gray-800 mb-2">
            {t("settings.sso.form.auth-url")}{" "}
            <span className="text-error">*</span>
          </label>
          <Input
            value={configForOAuth2.authUrl}
            onChange={(e) =>
              onUpdateOAuth2({ ...configForOAuth2, authUrl: e.target.value })
            }
            placeholder={t("settings.sso.form.auth-url-placeholder")}
          />
          <p className="text-sm text-gray-600 mt-1">
            {t("settings.sso.form.auth-url-description")}
          </p>
        </div>

        <div>
          <label className="block text-base font-semibold text-gray-800 mb-2">
            {t("settings.sso.form.token-url")}{" "}
            <span className="text-error">*</span>
          </label>
          <Input
            value={configForOAuth2.tokenUrl}
            onChange={(e) =>
              onUpdateOAuth2({ ...configForOAuth2, tokenUrl: e.target.value })
            }
            placeholder={t("settings.sso.form.token-url-placeholder")}
          />
          <p className="text-sm text-gray-600 mt-1">
            {t("settings.sso.form.token-url-description")}
          </p>
        </div>

        <div>
          <label className="block text-base font-semibold text-gray-800 mb-2">
            {t("settings.sso.form.user-info-url")}{" "}
            <span className="text-error">*</span>
          </label>
          <Input
            value={configForOAuth2.userInfoUrl}
            onChange={(e) =>
              onUpdateOAuth2({
                ...configForOAuth2,
                userInfoUrl: e.target.value,
              })
            }
            placeholder={t("settings.sso.form.user-info-url-placeholder")}
          />
          <p className="text-sm text-gray-600 mt-1">
            {t("settings.sso.form.user-info-url-description")}
          </p>
        </div>

        <div>
          <label className="block text-base font-semibold text-gray-800 mb-2">
            {t("settings.sso.form.scopes")}{" "}
            <span className="text-error">*</span>
          </label>
          <Input
            value={scopesString}
            onChange={(e) => onUpdateScopes(e.target.value)}
            placeholder={t("settings.sso.form.scopes-placeholder")}
          />
          <p className="text-sm text-gray-600 mt-1">
            {t("settings.sso.form.scopes-description")}
          </p>
        </div>

        <div>
          <label className="block text-base font-semibold text-gray-800 mb-3">
            {t("settings.sso.form.authentication-style")}{" "}
            <span className="text-error">*</span>
          </label>
          <div className="flex flex-col gap-y-3">
            <label className="flex items-start gap-x-3 cursor-pointer">
              <input
                type="radio"
                name="oauth2-auth-style"
                checked={
                  configForOAuth2.authStyle === OAuth2AuthStyle.IN_PARAMS
                }
                onChange={() =>
                  onUpdateOAuth2({
                    ...configForOAuth2,
                    authStyle: OAuth2AuthStyle.IN_PARAMS,
                  })
                }
                className="mt-1"
              />
              <div>
                <div className="font-medium">
                  {t("settings.sso.form.in-parameters")}
                </div>
                <div className="text-sm text-gray-600">
                  {t("settings.sso.form.in-parameters-description")}
                </div>
              </div>
            </label>
            <label className="flex items-start gap-x-3 cursor-pointer">
              <input
                type="radio"
                name="oauth2-auth-style"
                checked={
                  configForOAuth2.authStyle === OAuth2AuthStyle.IN_HEADER
                }
                onChange={() =>
                  onUpdateOAuth2({
                    ...configForOAuth2,
                    authStyle: OAuth2AuthStyle.IN_HEADER,
                  })
                }
                className="mt-1"
              />
              <div>
                <div className="font-medium">
                  {t("settings.sso.form.in-header")}
                </div>
                <div className="text-sm text-gray-600">
                  {t("settings.sso.form.in-header-description")}
                </div>
              </div>
            </label>
          </div>
        </div>

        <div>
          <label className="block text-base font-semibold text-gray-800 mb-3">
            {t("settings.sso.form.security-options")}
          </label>
          <label className="flex items-center gap-x-2 cursor-pointer">
            <input
              type="checkbox"
              checked={configForOAuth2.skipTlsVerify}
              onChange={(e) =>
                onUpdateOAuth2({
                  ...configForOAuth2,
                  skipTlsVerify: e.target.checked,
                })
              }
            />
            <span>{t("settings.sso.form.skip-tls-verification")}</span>
          </label>
          <p className="text-sm text-gray-600 mt-1 ml-6">
            {t("settings.sso.form.skip-tls-warning")}
          </p>
        </div>
      </div>
    );
  }

  if (providerType === IdentityProviderType.OIDC) {
    return (
      <div className="flex flex-col gap-y-6">
        <div>
          <label className="block text-base font-semibold text-gray-800 mb-2">
            Issuer URL <span className="text-error">*</span>
          </label>
          <Input
            value={configForOIDC.issuer}
            onChange={(e) =>
              onUpdateOIDC({ ...configForOIDC, issuer: e.target.value })
            }
            placeholder={t("settings.sso.form.issuer-url-placeholder")}
          />
          <p className="text-sm text-gray-600 mt-1">
            {t("settings.sso.form.issuer-url-description")}
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <label className="block text-base font-semibold text-gray-800 mb-2">
              Client ID <span className="text-error">*</span>
            </label>
            <Input
              value={configForOIDC.clientId}
              onChange={(e) =>
                onUpdateOIDC({ ...configForOIDC, clientId: e.target.value })
              }
              placeholder="e.g. 6655asd77895265aa110ac0d3"
            />
          </div>
          <div>
            <label className="block text-base font-semibold text-gray-800 mb-2">
              Client Secret <span className="text-error">*</span>
            </label>
            <Input
              type="password"
              value={configForOIDC.clientSecret}
              onChange={(e) =>
                onUpdateOIDC({
                  ...configForOIDC,
                  clientSecret: e.target.value,
                })
              }
              placeholder="e.g. 5bbezxc3972ca304de70c5d70a6aa932asd8"
            />
          </div>
        </div>

        <div>
          <label className="block text-base font-semibold text-gray-800 mb-2">
            {t("settings.sso.form.scopes")}{" "}
            <span className="text-error">*</span>
          </label>
          <Input
            value={scopesString}
            onChange={(e) => onUpdateScopes(e.target.value)}
            placeholder={t("settings.sso.form.scopes-placeholder-oidc")}
          />
          <p className="text-sm text-gray-600 mt-1">
            {t("settings.sso.form.openid-scopes-description")}
          </p>
        </div>

        <div>
          <label className="block text-base font-semibold text-gray-800 mb-3">
            {t("settings.sso.form.authentication-style")}{" "}
            <span className="text-error">*</span>
          </label>
          <div className="flex flex-col gap-y-3">
            <label className="flex items-start gap-x-3 cursor-pointer">
              <input
                type="radio"
                name="oidc-auth-style"
                checked={configForOIDC.authStyle === OAuth2AuthStyle.IN_PARAMS}
                onChange={() =>
                  onUpdateOIDC({
                    ...configForOIDC,
                    authStyle: OAuth2AuthStyle.IN_PARAMS,
                  })
                }
                className="mt-1"
              />
              <div>
                <div className="font-medium">
                  {t("settings.sso.form.in-parameters")}
                </div>
                <div className="text-sm text-gray-600">
                  {t("settings.sso.form.in-parameters-description")}
                </div>
              </div>
            </label>
            <label className="flex items-start gap-x-3 cursor-pointer">
              <input
                type="radio"
                name="oidc-auth-style"
                checked={configForOIDC.authStyle === OAuth2AuthStyle.IN_HEADER}
                onChange={() =>
                  onUpdateOIDC({
                    ...configForOIDC,
                    authStyle: OAuth2AuthStyle.IN_HEADER,
                  })
                }
                className="mt-1"
              />
              <div>
                <div className="font-medium">
                  {t("settings.sso.form.in-header")}
                </div>
                <div className="text-sm text-gray-600">
                  {t("settings.sso.form.in-header-description")}
                </div>
              </div>
            </label>
          </div>
        </div>

        <div>
          <label className="block text-base font-semibold text-gray-800 mb-3">
            {t("settings.sso.form.security-options")}
          </label>
          <label className="flex items-center gap-x-2 cursor-pointer">
            <input
              type="checkbox"
              checked={configForOIDC.skipTlsVerify}
              onChange={(e) =>
                onUpdateOIDC({
                  ...configForOIDC,
                  skipTlsVerify: e.target.checked,
                })
              }
            />
            <span>{t("settings.sso.form.skip-tls-verification")}</span>
          </label>
          <p className="text-sm text-gray-600 mt-1 ml-6">
            {t("settings.sso.form.skip-tls-warning")}
          </p>
        </div>
      </div>
    );
  }

  if (providerType === IdentityProviderType.LDAP) {
    return (
      <div className="flex flex-col gap-y-6">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="md:col-span-2">
            <label className="block text-base font-semibold text-gray-800 mb-2">
              Host <span className="text-error">*</span>
            </label>
            <Input
              value={configForLDAP.host}
              onChange={(e) =>
                onUpdateLDAP({ ...configForLDAP, host: e.target.value })
              }
              placeholder={t("settings.sso.form.host-placeholder")}
            />
          </div>
          <div>
            <label className="block text-base font-semibold text-gray-800 mb-2">
              Port <span className="text-error">*</span>
            </label>
            <Input
              type="number"
              value={configForLDAP.port}
              onChange={(e) =>
                onUpdateLDAP({
                  ...configForLDAP,
                  port: parseInt(e.target.value, 10) || 0,
                })
              }
              min={1}
              max={65535}
              placeholder="389"
            />
          </div>
        </div>

        <div>
          <label className="block text-base font-semibold text-gray-800 mb-2">
            {t("settings.sso.form.bind-dn")}{" "}
            <span className="text-error">*</span>
          </label>
          <Input
            value={configForLDAP.bindDn}
            onChange={(e) =>
              onUpdateLDAP({ ...configForLDAP, bindDn: e.target.value })
            }
            placeholder={t("settings.sso.form.bind-dn-placeholder")}
          />
          <p className="text-sm text-gray-600 mt-1">
            {t("settings.sso.form.bind-dn-description")}
          </p>
        </div>

        <div>
          <label className="block text-base font-semibold text-gray-800 mb-2">
            {t("settings.sso.form.bind-password")}{" "}
            <span className="text-error">*</span>
          </label>
          <Input
            type="password"
            value={configForLDAP.bindPassword}
            onChange={(e) =>
              onUpdateLDAP({
                ...configForLDAP,
                bindPassword: e.target.value,
              })
            }
            placeholder="••••••••"
          />
        </div>

        <div>
          <label className="block text-base font-semibold text-gray-800 mb-2">
            {t("settings.sso.form.base-dn")}{" "}
            <span className="text-error">*</span>
          </label>
          <Input
            value={configForLDAP.baseDn}
            onChange={(e) =>
              onUpdateLDAP({ ...configForLDAP, baseDn: e.target.value })
            }
            placeholder={t("settings.sso.form.base-dn-placeholder")}
          />
          <p className="text-sm text-gray-600 mt-1">
            {t("settings.sso.form.base-dn-description")}
          </p>
        </div>

        <div>
          <label className="block text-base font-semibold text-gray-800 mb-2">
            {t("settings.sso.form.user-filter")}{" "}
            <span className="text-error">*</span>
          </label>
          <Input
            value={configForLDAP.userFilter}
            onChange={(e) =>
              onUpdateLDAP({ ...configForLDAP, userFilter: e.target.value })
            }
            placeholder={t("settings.sso.form.user-filter-placeholder")}
          />
          <p className="text-sm text-gray-600 mt-1">
            {t("settings.sso.form.user-filter-description")}
          </p>
        </div>

        <div>
          <label className="block text-base font-semibold text-gray-800 mb-3">
            {t("settings.sso.form.security-protocol")}{" "}
            <span className="text-error">*</span>
          </label>
          <div className="flex flex-col gap-y-3">
            <label className="flex items-start gap-x-3 cursor-pointer">
              <input
                type="radio"
                name="ldap-security"
                checked={
                  configForLDAP.securityProtocol ===
                  LDAPIdentityProviderConfig_SecurityProtocol.START_TLS
                }
                onChange={() =>
                  onUpdateLDAP({
                    ...configForLDAP,
                    securityProtocol:
                      LDAPIdentityProviderConfig_SecurityProtocol.START_TLS,
                  })
                }
                className="mt-1"
              />
              <div>
                <div className="font-medium">
                  {t("settings.sso.form.starttls")}
                </div>
                <div className="text-sm text-gray-600">
                  {t("settings.sso.form.starttls-description")}
                </div>
              </div>
            </label>
            <label className="flex items-start gap-x-3 cursor-pointer">
              <input
                type="radio"
                name="ldap-security"
                checked={
                  configForLDAP.securityProtocol ===
                  LDAPIdentityProviderConfig_SecurityProtocol.LDAPS
                }
                onChange={() =>
                  onUpdateLDAP({
                    ...configForLDAP,
                    securityProtocol:
                      LDAPIdentityProviderConfig_SecurityProtocol.LDAPS,
                  })
                }
                className="mt-1"
              />
              <div>
                <div className="font-medium">
                  {t("settings.sso.form.ldaps")}
                </div>
                <div className="text-sm text-gray-600">
                  {t("settings.sso.form.ldaps-description")}
                </div>
              </div>
            </label>
            <label className="flex items-start gap-x-3 cursor-pointer">
              <input
                type="radio"
                name="ldap-security"
                checked={
                  configForLDAP.securityProtocol ===
                  LDAPIdentityProviderConfig_SecurityProtocol.SECURITY_PROTOCOL_UNSPECIFIED
                }
                onChange={() =>
                  onUpdateLDAP({
                    ...configForLDAP,
                    securityProtocol:
                      LDAPIdentityProviderConfig_SecurityProtocol.SECURITY_PROTOCOL_UNSPECIFIED,
                  })
                }
                className="mt-1"
              />
              <div>
                <div className="font-medium">{t("settings.sso.form.none")}</div>
                <div className="text-sm text-gray-600">
                  {t("settings.sso.form.none-description")}
                </div>
              </div>
            </label>
          </div>
        </div>

        <div>
          <label className="block text-base font-semibold text-gray-800 mb-3">
            {t("settings.sso.form.security-options")}
          </label>
          <label className="flex items-center gap-x-2 cursor-pointer">
            <input
              type="checkbox"
              checked={configForLDAP.skipTlsVerify}
              onChange={(e) =>
                onUpdateLDAP({
                  ...configForLDAP,
                  skipTlsVerify: e.target.checked,
                })
              }
            />
            <span>{t("settings.sso.form.skip-tls-verification")}</span>
          </label>
          <p className="text-sm text-gray-600 mt-1 ml-6">
            {t("settings.sso.form.skip-tls-warning")}
          </p>
        </div>
      </div>
    );
  }

  return null;
}

// ============================================================
// FieldMappingForm
// ============================================================

function FieldMappingForm({
  providerType,
  fieldMapping,
  onChange,
}: {
  providerType: IdentityProviderType;
  fieldMapping: FieldMappingState;
  onChange: (mapping: FieldMappingState) => void;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-y-6">
      <div className="grid grid-cols-[256px_1fr] gap-4 items-center">
        <Input
          value={fieldMapping.identifier}
          onChange={(e) =>
            onChange({ ...fieldMapping, identifier: e.target.value })
          }
          placeholder={t("settings.sso.form.identifier-placeholder")}
        />
        <div className="flex items-center text-base">
          <ArrowRight className="mx-2 h-5 w-5 text-gray-400" />
          <p className="flex items-center font-semibold text-gray-800">
            {t("settings.sso.form.identifier")}
            <span className="ml-0.5 text-error">*</span>
            <span
              className="ml-1"
              title={t("settings.sso.form.identifier-tips")}
            >
              <Info className="w-4 h-4 text-blue-500" />
            </span>
          </p>
        </div>
      </div>

      <div className="grid grid-cols-[256px_1fr] gap-4 items-center">
        <Input
          value={fieldMapping.displayName}
          onChange={(e) =>
            onChange({ ...fieldMapping, displayName: e.target.value })
          }
          placeholder={t("settings.sso.form.display-name-placeholder")}
        />
        <div className="flex items-center text-base">
          <ArrowRight className="mx-2 h-5 w-5 text-gray-400" />
          <p className="font-semibold text-gray-800">
            {t("settings.sso.form.display-name")}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-[256px_1fr] gap-4 items-center">
        <Input
          value={fieldMapping.phone}
          onChange={(e) => onChange({ ...fieldMapping, phone: e.target.value })}
          placeholder={t("settings.sso.form.phone-placeholder")}
        />
        <div className="flex items-center text-base">
          <ArrowRight className="mx-2 h-5 w-5 text-gray-400" />
          <p className="font-semibold text-gray-800">
            {t("settings.sso.form.phone")}
          </p>
        </div>
      </div>

      {providerType === IdentityProviderType.OIDC && (
        <>
          <div className="grid grid-cols-[256px_1fr] gap-4 items-center">
            <Input
              value={fieldMapping.groups}
              onChange={(e) =>
                onChange({ ...fieldMapping, groups: e.target.value })
              }
              placeholder={t("settings.sso.form.groups-placeholder")}
            />
            <div className="flex items-center text-base">
              <ArrowRight className="mx-2 h-5 w-5 text-gray-400" />
              <p className="font-semibold text-gray-800">
                {t("settings.sso.form.groups")}
              </p>
            </div>
          </div>
          <p className="text-sm text-gray-600">
            {t("settings.sso.form.groups-description")}
          </p>
        </>
      )}
    </div>
  );
}

// ============================================================
// CreateWizardDrawer
// ============================================================

function getTemplateList(hasEnterpriseSSOFeature: boolean): OAuth2Template[] {
  return [
    {
      title: "Google",
      domain: "google.com",
      domainDisabled: !hasEnterpriseSSOFeature,
      feature: PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO,
      config: create(OAuth2IdentityProviderConfigSchema, {
        clientId: "",
        clientSecret: "",
        authUrl: "https://accounts.google.com/o/oauth2/v2/auth",
        tokenUrl: "https://oauth2.googleapis.com/token",
        userInfoUrl: "https://www.googleapis.com/oauth2/v2/userinfo",
        scopes: [
          "https://www.googleapis.com/auth/userinfo.email",
          "https://www.googleapis.com/auth/userinfo.profile",
        ],
        skipTlsVerify: false,
        authStyle: OAuth2AuthStyle.IN_PARAMS,
        fieldMapping: create(FieldMappingSchema, {
          identifier: "email",
          displayName: "name",
          phone: "",
          groups: "",
        }),
      }),
    },
    {
      title: "GitHub",
      domain: "github.com",
      domainDisabled: !hasEnterpriseSSOFeature,
      feature: PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO,
      config: create(OAuth2IdentityProviderConfigSchema, {
        clientId: "",
        clientSecret: "",
        authUrl: "https://github.com/login/oauth/authorize",
        tokenUrl: "https://github.com/login/oauth/access_token",
        userInfoUrl: "https://api.github.com/user",
        scopes: ["user", "user:email"],
        skipTlsVerify: false,
        authStyle: OAuth2AuthStyle.IN_PARAMS,
        fieldMapping: create(FieldMappingSchema, {
          identifier: "email",
          displayName: "name",
          phone: "",
          groups: "",
        }),
      }),
    },
    {
      title: "GitLab",
      domain: "gitlab.com",
      feature: PlanFeature.FEATURE_ENTERPRISE_SSO,
      config: create(OAuth2IdentityProviderConfigSchema, {
        clientId: "",
        clientSecret: "",
        authUrl: "https://gitlab.com/oauth/authorize",
        tokenUrl: "https://gitlab.com/oauth/token",
        userInfoUrl: "https://gitlab.com/api/v4/user",
        scopes: ["read_user"],
        skipTlsVerify: false,
        authStyle: OAuth2AuthStyle.IN_PARAMS,
        fieldMapping: create(FieldMappingSchema, {
          identifier: "email",
          displayName: "name",
          phone: "",
          groups: "",
        }),
      }),
    },
    {
      title: "Microsoft Entra",
      domain: "",
      feature: PlanFeature.FEATURE_ENTERPRISE_SSO,
      config: create(OAuth2IdentityProviderConfigSchema, {
        clientId: "",
        clientSecret: "",
        authUrl:
          "https://login.microsoftonline.com/{uuid}/oauth2/v2.0/authorize",
        tokenUrl: "https://login.microsoftonline.com/{uuid}/oauth2/v2.0/token",
        userInfoUrl: "https://graph.microsoft.com/v1.0/me",
        scopes: ["user.read"],
        skipTlsVerify: false,
        authStyle: OAuth2AuthStyle.IN_PARAMS,
        fieldMapping: create(FieldMappingSchema, {
          identifier: "userPrincipalName",
          displayName: "displayName",
          phone: "",
          groups: "",
        }),
      }),
    },
    {
      title: "Custom",
      domain: "",
      feature: PlanFeature.FEATURE_ENTERPRISE_SSO,
      config: create(OAuth2IdentityProviderConfigSchema, {
        clientId: "",
        clientSecret: "",
        authUrl: "",
        tokenUrl: "",
        userInfoUrl: "",
        scopes: [],
        skipTlsVerify: false,
        authStyle: OAuth2AuthStyle.IN_PARAMS,
        fieldMapping: create(FieldMappingSchema, {
          identifier: "",
          displayName: "",
          phone: "",
          groups: "",
        }),
      }),
    },
  ];
}

const PROVIDER_TYPE_LIST = [
  {
    type: IdentityProviderType.OAUTH2,
    feature: PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO,
  },
  {
    type: IdentityProviderType.OIDC,
    feature: PlanFeature.FEATURE_ENTERPRISE_SSO,
  },
  {
    type: IdentityProviderType.LDAP,
    feature: PlanFeature.FEATURE_ENTERPRISE_SSO,
  },
];

function getProviderIcon(type: IdentityProviderType) {
  switch (type) {
    case IdentityProviderType.OAUTH2:
      return Key;
    case IdentityProviderType.OIDC:
      return ShieldCheck;
    case IdentityProviderType.LDAP:
      return Database;
    default:
      return Key;
  }
}

function getTemplateIcon(title: string) {
  switch (title.toLowerCase()) {
    case "google":
      return Globe;
    case "github":
      return GitBranch;
    case "gitlab":
      return GitFork;
    case "microsoft entra":
      return Building;
    default:
      return Key;
  }
}

function CreateWizardDrawer({
  onClose,
  onCreated,
}: {
  onClose: () => void;
  onCreated: (provider: IdentityProvider) => void;
}) {
  const { t } = useTranslation();
  const identityProviderStore = useIdentityProviderStore();
  const subscriptionStore = useSubscriptionV1Store();
  useEscapeKey(onClose);

  const hasEnterpriseSSOFeature = useVueState(() =>
    subscriptionStore.hasFeature(PlanFeature.FEATURE_ENTERPRISE_SSO)
  );

  const [currentStep, setCurrentStep] = useState(1);
  const [selectedType, setSelectedType] = useState<IdentityProviderType>(
    IdentityProviderType.OAUTH2
  );
  const [selectedTemplate, setSelectedTemplate] =
    useState<OAuth2Template | null>(null);
  const [isCreating, setIsCreating] = useState(false);

  const [title, setTitle] = useState("");
  const [domain, setDomain] = useState("");
  const [resourceId, setResourceId] = useState("");
  const [resourceIdValid, setResourceIdValid] = useState(true);
  const resourceIdFieldRef = useRef<ResourceIdFieldRef>(null);

  const [configForOAuth2, setConfigForOAuth2] =
    useState<OAuth2IdentityProviderConfig>(
      create(OAuth2IdentityProviderConfigSchema, {
        authStyle: OAuth2AuthStyle.IN_PARAMS,
      })
    );
  const [configForOIDC, setConfigForOIDC] =
    useState<OIDCIdentityProviderConfig>(
      create(OIDCIdentityProviderConfigSchema, {
        authStyle: OAuth2AuthStyle.IN_PARAMS,
      })
    );
  const [configForLDAP, setConfigForLDAP] =
    useState<LDAPIdentityProviderConfig>(
      create(LDAPIdentityProviderConfigSchema, { port: 389 })
    );
  const [scopesString, setScopesString] = useState("");
  const [fieldMapping, setFieldMapping] = useState<FieldMappingState>({
    identifier: "",
    displayName: "",
    phone: "",
    groups: "",
  });

  const templateList = useMemo(
    () => getTemplateList(hasEnterpriseSSOFeature),
    [hasEnterpriseSSOFeature]
  );

  // Initialize with first template on mount
  useEffect(() => {
    const first = templateList[0];
    if (first) {
      applyTemplate(first);
    }
  }, []);

  const applyTemplate = useCallback((template: OAuth2Template) => {
    setSelectedTemplate(template);
    setTitle(template.title);
    setDomain(template.domain);
    setConfigForOAuth2(
      create(OAuth2IdentityProviderConfigSchema, {
        clientId: template.config.clientId || "",
        clientSecret: template.config.clientSecret || "",
        authUrl: template.config.authUrl || "",
        tokenUrl: template.config.tokenUrl || "",
        userInfoUrl: template.config.userInfoUrl || "",
        scopes: template.config.scopes || [],
        skipTlsVerify: template.config.skipTlsVerify || false,
        authStyle: template.config.authStyle || OAuth2AuthStyle.IN_PARAMS,
      })
    );
    setScopesString((template.config.scopes || []).join(" "));
    const fm = template.config.fieldMapping;
    setFieldMapping({
      identifier: fm?.identifier || "",
      displayName: fm?.displayName || "",
      phone: fm?.phone || "",
      groups: fm?.groups || "",
    });
  }, []);

  const maxSteps = selectedType === IdentityProviderType.OAUTH2 ? 5 : 4;
  const isLastStep = currentStep === maxSteps;

  const isConfigurationValid = useMemo(() => {
    if (selectedType === IdentityProviderType.OAUTH2) {
      return !!(
        configForOAuth2.clientId &&
        configForOAuth2.clientSecret &&
        configForOAuth2.authUrl &&
        configForOAuth2.tokenUrl &&
        configForOAuth2.userInfoUrl &&
        scopesString
      );
    }
    if (selectedType === IdentityProviderType.OIDC) {
      return !!(
        configForOIDC.clientId &&
        configForOIDC.clientSecret &&
        configForOIDC.issuer &&
        scopesString
      );
    }
    if (selectedType === IdentityProviderType.LDAP) {
      return !!(
        configForLDAP.host &&
        configForLDAP.port &&
        configForLDAP.bindDn &&
        configForLDAP.bindPassword &&
        configForLDAP.baseDn &&
        configForLDAP.userFilter
      );
    }
    return false;
  }, [
    selectedType,
    configForOAuth2,
    configForOIDC,
    configForLDAP,
    scopesString,
  ]);

  const canProceed = useMemo(() => {
    const isOAuth2 = selectedType === IdentityProviderType.OAUTH2;
    switch (currentStep) {
      case 1:
        return !!selectedType;
      case 2:
        if (isOAuth2) return !!selectedTemplate;
        return !!(title && resourceId && resourceIdValid);
      case 3:
        if (isOAuth2) return !!(title && resourceId && resourceIdValid);
        return isConfigurationValid;
      case 4:
        if (isOAuth2) return isConfigurationValid;
        return !!fieldMapping.identifier;
      case 5:
        return !!fieldMapping.identifier;
      default:
        return false;
    }
  }, [
    currentStep,
    selectedType,
    selectedTemplate,
    title,
    resourceId,
    resourceIdValid,
    isConfigurationValid,
    fieldMapping.identifier,
  ]);

  const canCreate = !!(
    title &&
    resourceId &&
    isConfigurationValid &&
    fieldMapping.identifier
  );

  const allowTestConnection = isConfigurationValid && !!fieldMapping.identifier;

  const buildIdpToCreate = useCallback((): IdentityProvider => {
    const base = create(IdentityProviderSchema, {
      name: resourceId,
      title,
      domain,
      type: selectedType,
    });

    if (selectedType === IdentityProviderType.OAUTH2) {
      base.config = create(IdentityProviderConfigSchema, {
        config: {
          case: "oauth2Config",
          value: create(OAuth2IdentityProviderConfigSchema, {
            ...configForOAuth2,
            scopes: scopesString.split(" ").filter(Boolean),
            fieldMapping: create(FieldMappingSchema, fieldMapping),
          }),
        },
      });
    } else if (selectedType === IdentityProviderType.OIDC) {
      base.config = create(IdentityProviderConfigSchema, {
        config: {
          case: "oidcConfig",
          value: create(OIDCIdentityProviderConfigSchema, {
            ...configForOIDC,
            scopes: scopesString.split(" ").filter(Boolean),
            fieldMapping: create(FieldMappingSchema, fieldMapping),
          }),
        },
      });
    } else if (selectedType === IdentityProviderType.LDAP) {
      base.config = create(IdentityProviderConfigSchema, {
        config: {
          case: "ldapConfig",
          value: create(LDAPIdentityProviderConfigSchema, {
            ...configForLDAP,
            fieldMapping: create(FieldMappingSchema, fieldMapping),
          }),
        },
      });
    }

    return base;
  }, [
    resourceId,
    title,
    domain,
    selectedType,
    configForOAuth2,
    configForOIDC,
    configForLDAP,
    scopesString,
    fieldMapping,
  ]);

  const idpToCreate = useMemo(() => buildIdpToCreate(), [buildIdpToCreate]);

  const handleTypeChange = (type: IdentityProviderType) => {
    setSelectedType(type);
    if (type === IdentityProviderType.OAUTH2) {
      const first = templateList[0];
      if (first) applyTemplate(first);
    } else {
      setTitle("");
      setDomain("");
      setScopesString("");
      setFieldMapping({
        identifier: "",
        displayName: "",
        phone: "",
        groups: "",
      });
    }
  };

  const handlePrev = () => {
    if (currentStep > 1) setCurrentStep(currentStep - 1);
  };

  const handleNext = () => {
    if (currentStep < maxSteps && canProceed) {
      setCurrentStep(currentStep + 1);
    }
  };

  const handleCreate = async () => {
    if (!canCreate) return;
    setIsCreating(true);
    try {
      const createdProvider =
        await identityProviderStore.createIdentityProvider(idpToCreate);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("identity-provider.identity-provider-created"),
      });
      onCreated(createdProvider);
    } catch (error) {
      if (error instanceof ConnectError && error.code === Code.AlreadyExists) {
        resourceIdFieldRef.current?.addValidationError(
          (error as ConnectError).message
        );
      } else {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("identity-provider.identity-provider-create-failed"),
        });
      }
    } finally {
      setIsCreating(false);
    }
  };

  const validateResourceId = useCallback(
    async (val: string) => {
      try {
        await identityProviderStore.getOrFetchIdentityProviderByName(
          `${idpNamePrefix}${val}`,
          true
        );
        return [
          {
            type: "error" as const,
            message: t("resource-id.validation.duplicated", {
              resource: "SSO",
            }),
          },
        ];
      } catch {
        return [];
      }
    },
    [identityProviderStore, t]
  );

  // Step labels
  const stepLabels = useMemo(() => {
    const labels = [t("settings.sso.form.type")];
    if (selectedType === IdentityProviderType.OAUTH2) {
      labels.push(t("settings.sso.form.use-template"));
    }
    labels.push(t("common.general"));
    labels.push(t("settings.sso.form.configuration"));
    labels.push(t("settings.sso.form.user-information-mapping"));
    return labels;
  }, [selectedType, t]);

  const getProviderDescription = (type: IdentityProviderType) => {
    switch (type) {
      case IdentityProviderType.OAUTH2:
        return t("settings.sso.form.oauth2-description");
      case IdentityProviderType.OIDC:
        return t("settings.sso.form.oidc-description");
      case IdentityProviderType.LDAP:
        return t("settings.sso.form.ldap-description");
      default:
        return "";
    }
  };

  const getTemplateDescription = (templateTitle: string) => {
    switch (templateTitle.toLowerCase()) {
      case "google":
        return t("settings.sso.form.google-template-description");
      case "github":
        return t("settings.sso.form.github-template-description");
      case "gitlab":
        return t("settings.sso.form.gitlab-template-description");
      case "microsoft entra":
        return t("settings.sso.form.microsoft-entra-template-description");
      case "custom":
        return t("settings.sso.form.custom-template-description");
      default:
        return "";
    }
  };

  // Determine which step content to render
  const isOAuth2 = selectedType === IdentityProviderType.OAUTH2;
  const basicInfoStep = isOAuth2 ? 3 : 2;
  const configStep = isOAuth2 ? 4 : 3;
  const mappingStep = isOAuth2 ? 5 : 4;

  return (
    <>
      <div className="fixed inset-0 z-40 bg-black/30" onClick={onClose} />
      <div className="fixed inset-y-0 right-0 z-50 w-[64rem] max-w-[100vw] bg-white shadow-xl flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 className="text-lg font-medium">{t("settings.sso.create")}</h2>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="h-5 w-5" />
          </Button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto px-6 py-6">
          <div className="flex flex-col gap-y-6">
            {/* Step indicators */}
            <div className="flex items-center gap-x-2">
              {stepLabels.map((label, i) => {
                const stepNum = i + 1;
                const isActive = stepNum === currentStep;
                const isComplete = stepNum < currentStep;
                return (
                  <div key={i} className="flex items-center gap-x-2">
                    {i > 0 && (
                      <div
                        className={`w-8 h-px ${isComplete ? "bg-accent" : "bg-gray-300"}`}
                      />
                    )}
                    <div className="flex items-center gap-x-1.5">
                      <div
                        className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-medium ${
                          isActive
                            ? "bg-accent text-white"
                            : isComplete
                              ? "bg-accent/20 text-accent"
                              : "bg-gray-200 text-gray-500"
                        }`}
                      >
                        {stepNum}
                      </div>
                      <span
                        className={`text-sm ${isActive ? "font-medium text-main" : "text-gray-500"}`}
                      >
                        {label}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>

            {/* Step content */}
            <div className="bg-white rounded-sm border border-gray-200 px-6 pt-6 pb-10">
              {/* Step 1: Select provider type */}
              {currentStep === 1 && (
                <div className="flex flex-col gap-y-6">
                  <div className="text-center flex flex-col gap-y-2">
                    <h2 className="text-2xl font-bold text-gray-900">
                      {t("settings.sso.form.type")}
                    </h2>
                    <p className="text-gray-600">
                      {t("settings.sso.form.type-description")}
                    </p>
                  </div>
                  <div className="max-w-2xl mx-auto w-full">
                    {PROVIDER_TYPE_LIST.map((item) => {
                      const Icon = getProviderIcon(item.type);
                      const hasFeature = subscriptionStore.hasFeature(
                        item.feature
                      );
                      return (
                        <label
                          key={item.type}
                          className={`block border rounded-sm mb-4 p-4 transition-colors cursor-pointer ${
                            selectedType === item.type
                              ? "border-blue-500 bg-blue-50"
                              : "border-gray-200 hover:border-gray-300"
                          } ${!hasFeature ? "opacity-50 cursor-not-allowed" : ""}`}
                        >
                          <div className="flex items-start gap-x-3">
                            <input
                              type="radio"
                              name="provider-type"
                              checked={selectedType === item.type}
                              disabled={!hasFeature}
                              onChange={() => handleTypeChange(item.type)}
                              className="mt-1.5"
                            />
                            <Icon
                              className="w-6 h-6 mt-1 shrink-0"
                              strokeWidth={1.5}
                            />
                            <div className="flex-1">
                              <div className="flex items-center gap-x-2">
                                <span className="text-lg font-medium text-gray-900">
                                  {identityProviderTypeToString(item.type)}
                                </span>
                              </div>
                              <p className="text-sm text-gray-600 mt-1">
                                {getProviderDescription(item.type)}
                              </p>
                            </div>
                          </div>
                        </label>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Step 2 for OAuth2: Select template */}
              {currentStep === 2 && isOAuth2 && (
                <div className="flex flex-col gap-y-6">
                  <div className="text-center flex flex-col gap-y-2">
                    <h2 className="text-2xl font-bold text-gray-900">
                      {t("settings.sso.form.use-template")}
                    </h2>
                    <p className="text-gray-600">
                      {t("settings.sso.form.template-description")}
                    </p>
                  </div>
                  <div className="max-w-3xl mx-auto grid grid-cols-1 sm:grid-cols-2 gap-4">
                    {templateList.map((tmpl) => {
                      const Icon = getTemplateIcon(tmpl.title);
                      const hasFeature = subscriptionStore.hasFeature(
                        tmpl.feature
                      );
                      return (
                        <label
                          key={tmpl.title}
                          className={`block border rounded-sm p-4 transition-colors cursor-pointer ${
                            selectedTemplate?.title === tmpl.title
                              ? "border-blue-500 bg-blue-50"
                              : "border-gray-200 hover:border-gray-300"
                          } ${!hasFeature ? "opacity-50 cursor-not-allowed" : ""}`}
                        >
                          <div className="flex items-center gap-x-3">
                            <input
                              type="radio"
                              name="template"
                              checked={selectedTemplate?.title === tmpl.title}
                              disabled={!hasFeature}
                              onChange={() => applyTemplate(tmpl)}
                              className="shrink-0"
                            />
                            <Icon
                              className="w-8 h-8 shrink-0"
                              strokeWidth={1}
                            />
                            <div className="flex-1">
                              <span className="text-base font-medium text-gray-900">
                                {tmpl.title}
                              </span>
                              <p className="text-sm text-gray-600">
                                {getTemplateDescription(tmpl.title)}
                              </p>
                            </div>
                          </div>
                        </label>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Basic information step */}
              {currentStep === basicInfoStep && (
                <div className="flex flex-col gap-y-6">
                  <div className="text-center flex flex-col gap-y-2">
                    <h2 className="text-2xl font-bold text-gray-900">
                      {t("common.general")}
                    </h2>
                    <p className="text-gray-600">
                      {t("settings.sso.form.general-setting-description")}
                    </p>
                  </div>
                  <div className="max-w-2xl mx-auto flex flex-col gap-y-6 w-full">
                    <div>
                      <label className="block text-base font-semibold text-gray-800 mb-2">
                        {t("settings.sso.form.name")}{" "}
                        <span className="text-error">*</span>
                      </label>
                      <Input
                        value={title}
                        onChange={(e) => setTitle(e.target.value)}
                        placeholder={t("settings.sso.form.name-description")}
                        className="mb-2"
                      />
                      <ResourceIdField
                        ref={resourceIdFieldRef}
                        value={resourceId}
                        resourceType="idp"
                        resourceName="SSO"
                        resourceTitle={title}
                        suffix
                        validate={validateResourceId}
                        onChange={setResourceId}
                        onValidationChange={setResourceIdValid}
                      />
                    </div>
                    <div>
                      <label className="block text-base font-semibold text-gray-800 mb-2">
                        {t("settings.sso.form.domain")}
                      </label>
                      <Input
                        value={domain}
                        onChange={(e) => setDomain(e.target.value)}
                        disabled={selectedTemplate?.domainDisabled}
                        placeholder={t("settings.sso.form.domain-description")}
                      />
                      <p className="text-sm text-gray-600 mt-1">
                        {t("settings.sso.form.domain-optional-hint")}
                      </p>
                    </div>
                  </div>
                </div>
              )}

              {/* Configuration step */}
              {currentStep === configStep && (
                <div className="flex flex-col gap-y-6">
                  <div className="text-center flex flex-col gap-y-2">
                    <h2 className="text-2xl font-bold text-gray-900">
                      {t("settings.sso.form.configuration")}
                    </h2>
                    <p className="text-gray-600">
                      {t("settings.sso.form.configuration-description")}
                    </p>
                  </div>
                  <div className="max-w-3xl mx-auto w-full">
                    {(selectedType === IdentityProviderType.OAUTH2 ||
                      selectedType === IdentityProviderType.OIDC) && (
                      <div className="mb-6">
                        <ExternalURLInfo type={selectedType} />
                      </div>
                    )}
                    <ProviderConfigForm
                      providerType={selectedType}
                      configForOAuth2={configForOAuth2}
                      configForOIDC={configForOIDC}
                      configForLDAP={configForLDAP}
                      scopesString={scopesString}
                      onUpdateOAuth2={setConfigForOAuth2}
                      onUpdateOIDC={setConfigForOIDC}
                      onUpdateLDAP={setConfigForLDAP}
                      onUpdateScopes={setScopesString}
                    />
                  </div>
                </div>
              )}

              {/* User information mapping step */}
              {currentStep === mappingStep && (
                <div className="flex flex-col gap-y-6">
                  <div className="text-center flex flex-col gap-y-2">
                    <h2 className="text-2xl font-bold text-gray-900">
                      {t("settings.sso.form.user-information-mapping")}
                    </h2>
                    <p className="text-gray-600">
                      {t(
                        "settings.sso.form.user-information-mapping-description"
                      )}{" "}
                      <a
                        href="https://docs.bytebase.com/administration/sso/oauth2#user-information-field-mapping?source=console"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-accent hover:underline ml-1"
                      >
                        {t("common.learn-more")}
                      </a>
                    </p>
                  </div>
                  <div className="max-w-2xl mx-auto flex flex-col gap-y-6 w-full">
                    <FieldMappingForm
                      providerType={selectedType}
                      fieldMapping={fieldMapping}
                      onChange={setFieldMapping}
                    />
                    <div>
                      <TestConnectionButton
                        disabled={!allowTestConnection}
                        isCreating
                        idp={idpToCreate}
                      />
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-x-2 px-6 py-4 border-t">
          {currentStep === 1 ? (
            <Button variant="outline" onClick={onClose}>
              {t("common.cancel")}
            </Button>
          ) : (
            <Button variant="outline" onClick={handlePrev}>
              {t("common.back")}
            </Button>
          )}
          {!isLastStep ? (
            <Button disabled={!canProceed} onClick={handleNext}>
              {t("common.next")}
            </Button>
          ) : (
            <Button disabled={!canCreate || isCreating} onClick={handleCreate}>
              {t("common.create")}
            </Button>
          )}
        </div>

        {isCreating && (
          <div className="absolute inset-0 z-10 bg-white/50 flex items-center justify-center">
            <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
          </div>
        )}
      </div>
    </>
  );
}

// ============================================================
// IDPsPage (main)
// ============================================================

export function IDPsPage() {
  const { t } = useTranslation();
  const identityProviderStore = useIdentityProviderStore();

  const subscriptionStore = useSubscriptionV1Store();

  const [ready, setReady] = useState(false);
  const [showCreateDrawer, setShowCreateDrawer] = useState(false);

  const hasSSOFeature = useVueState(() =>
    subscriptionStore.hasFeature(PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO)
  );
  const canCreate = hasWorkspacePermissionV2("bb.identityProviders.create");

  const identityProviderList = useVueState(() => [
    ...identityProviderStore.identityProviderList,
  ]);

  useEffect(() => {
    identityProviderStore
      .fetchIdentityProviderList()
      .finally(() => setReady(true));
  }, []);

  const handleCreateSSO = () => {
    if (!hasSSOFeature) return;
    setShowCreateDrawer(true);
  };

  const handleProviderCreated = (provider: IdentityProvider) => {
    setShowCreateDrawer(false);
    router.replace({
      name: WORKSPACE_ROUTE_IDENTITY_PROVIDER_DETAIL,
      params: {
        idpId: getIdentityProviderResourceId(provider.name),
      },
    });
  };

  const handleRowClick = (idp: IdentityProvider) => {
    router.push({
      name: WORKSPACE_ROUTE_IDENTITY_PROVIDER_DETAIL,
      params: {
        idpId: getIdentityProviderResourceId(idp.name),
      },
    });
  };

  return (
    <div className="w-full px-4 py-4 flex flex-col gap-y-4">
      <FeatureAttention feature={PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO} />

      <div className="textinfolabel">
        {t("settings.sso.description")}{" "}
        <a
          href="https://docs.bytebase.com/administration/sso/overview?source=console"
          target="_blank"
          rel="noopener noreferrer"
          className="text-accent hover:underline"
        >
          {t("common.learn-more")}
        </a>
      </div>

      <div className="w-full flex justify-end">
        <PermissionGuard permissions={["bb.identityProviders.create"]}>
          <Button disabled={!canCreate} onClick={handleCreateSSO}>
            <FeatureBadge
              feature={PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO}
              clickable={false}
              className="mr-1 text-white inline-flex"
            />
            <Plus className="h-4 w-4 mr-1" />
            {t("settings.sso.create")}
          </Button>
        </PermissionGuard>
      </div>

      {ready ? (
        <div className="border rounded-sm overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-control-bg">
                <th className="px-4 py-2 text-left font-medium w-40">
                  {t("common.id")}
                </th>
                <th className="px-4 py-2 text-left font-medium">
                  {t("common.name")}
                </th>
                <th className="px-4 py-2 text-left font-medium w-32">
                  {t("common.type")}
                </th>
                <th className="px-4 py-2 text-left font-medium w-48">
                  {t("settings.sso.form.domain")}
                </th>
              </tr>
            </thead>
            <tbody>
              {identityProviderList.length === 0 ? (
                <tr>
                  <td
                    colSpan={4}
                    className="px-4 py-8 text-center text-control-light"
                  >
                    {t("common.no-data")}
                  </td>
                </tr>
              ) : (
                identityProviderList.map((idp, i) => (
                  <tr
                    key={idp.name}
                    className={`border-b last:border-b-0 cursor-pointer hover:bg-gray-50 ${
                      i % 2 === 1 ? "bg-gray-50/50" : ""
                    }`}
                    onClick={() => handleRowClick(idp)}
                  >
                    <td className="px-4 py-2">
                      {getIdentityProviderResourceId(idp.name)}
                    </td>
                    <td className="px-4 py-2">{idp.title}</td>
                    <td className="px-4 py-2">
                      {identityProviderTypeToString(idp.type)}
                    </td>
                    <td className="px-4 py-2">{idp.domain || "-"}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="flex items-center justify-center h-32">
          <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
        </div>
      )}

      {showCreateDrawer && (
        <CreateWizardDrawer
          onClose={() => setShowCreateDrawer(false)}
          onCreated={handleProviderCreated}
        />
      )}
    </div>
  );
}
