import { create } from "@bufbuild/protobuf";
import type { ConnectError } from "@connectrpc/connect";
import { cloneDeep, isEqual } from "lodash-es";
import { ArrowRight, Database, Info, Key, ShieldCheck, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { identityProviderServiceClientConnect } from "@/connect";
import { ResourceIdField } from "@/react/components/ResourceIdField";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { WORKSPACE_ROUTE_IDENTITY_PROVIDERS } from "@/router/dashboard/workspaceRoutes";
import { pushNotification } from "@/store";
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
import {
  hasWorkspacePermissionV2,
  identityProviderTypeToString,
  openWindowForSSO,
} from "@/utils";

// ============================================================
// Types
// ============================================================

interface FieldMappingState {
  identifier: string;
  displayName: string;
  phone: string;
  groups: string;
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

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [onClose]);

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/50"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="bg-white rounded-md shadow-lg w-[32rem] max-h-[80vh] overflow-auto p-6">
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
          <div className="bg-gray-50 rounded-md p-4">
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
          <div className="bg-gray-50 rounded-md p-4">
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
}: {
  idp: IdentityProvider;
  disabled: boolean;
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
      const eventName = `bb.oauth.signin.${idp.name}`;
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
        await openWindowForSSO(idp);
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
// DeleteConfirmDialog
// ============================================================

function DeleteConfirmDialog({
  onConfirm,
  onCancel,
}: {
  onConfirm: () => void;
  onCancel: () => void;
}) {
  const { t } = useTranslation();

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onCancel();
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [onCancel]);

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/50"
      onClick={(e) => {
        if (e.target === e.currentTarget) onCancel();
      }}
    >
      <div className="bg-white rounded-md shadow-lg w-96 p-6">
        <h3 className="text-lg font-medium mb-2">{t("settings.sso.delete")}</h3>
        <p className="text-sm text-gray-600 mb-6">
          {t("identity-provider.delete-warning")}
        </p>
        <div className="flex justify-end gap-x-2">
          <Button variant="outline" onClick={onCancel}>
            {t("common.cancel")}
          </Button>
          <Button variant="destructive" onClick={onConfirm}>
            {t("common.delete")}
          </Button>
        </div>
      </div>
    </div>
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
  isEditMode,
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
  isEditMode: boolean;
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
              Client Secret{" "}
              {!isEditMode && <span className="text-error">*</span>}
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
              placeholder={
                isEditMode && !configForOAuth2.clientSecret
                  ? "Leave empty to keep existing secret"
                  : "e.g. 5bbezxc3972ca304de70c5d70a6aa932asd8"
              }
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
              Client Secret{" "}
              {!isEditMode && <span className="text-error">*</span>}
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
              placeholder={
                isEditMode && !configForOIDC.clientSecret
                  ? "Leave empty to keep existing secret"
                  : "e.g. 5bbezxc3972ca304de70c5d70a6aa932asd8"
              }
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
            <input
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
              className="flex h-9 w-full rounded-md border border-control-border bg-transparent px-3 py-1 text-sm"
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
            {!isEditMode && <span className="text-error">*</span>}
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
            placeholder={
              isEditMode && !configForLDAP.bindPassword
                ? "Leave empty to keep existing password"
                : "••••••••"
            }
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
// Helper functions
// ============================================================

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

function extractConfigState(idp: IdentityProvider) {
  let configForOAuth2 = create(OAuth2IdentityProviderConfigSchema, {});
  let configForOIDC = create(OIDCIdentityProviderConfigSchema, {});
  let configForLDAP = create(LDAPIdentityProviderConfigSchema, {});
  let scopesString = "";
  const fieldMapping: FieldMappingState = {
    identifier: "",
    displayName: "",
    phone: "",
    groups: "",
  };

  if (idp.config?.config?.case === "oauth2Config") {
    const cfg = idp.config.config.value;
    configForOAuth2 = create(OAuth2IdentityProviderConfigSchema, cfg);
    scopesString = (cfg.scopes || []).join(" ");
    if (cfg.fieldMapping) {
      fieldMapping.identifier = cfg.fieldMapping.identifier;
      fieldMapping.displayName = cfg.fieldMapping.displayName;
      fieldMapping.phone = cfg.fieldMapping.phone;
      fieldMapping.groups = cfg.fieldMapping.groups;
    }
  } else if (idp.config?.config?.case === "oidcConfig") {
    const cfg = idp.config.config.value;
    configForOIDC = create(OIDCIdentityProviderConfigSchema, cfg);
    scopesString = (cfg.scopes || []).join(" ");
    if (cfg.fieldMapping) {
      fieldMapping.identifier = cfg.fieldMapping.identifier;
      fieldMapping.displayName = cfg.fieldMapping.displayName;
      fieldMapping.phone = cfg.fieldMapping.phone;
      fieldMapping.groups = cfg.fieldMapping.groups;
    }
  } else if (idp.config?.config?.case === "ldapConfig") {
    const cfg = idp.config.config.value;
    configForLDAP = create(LDAPIdentityProviderConfigSchema, cfg);
    if (cfg.fieldMapping) {
      fieldMapping.identifier = cfg.fieldMapping.identifier;
      fieldMapping.displayName = cfg.fieldMapping.displayName;
      fieldMapping.phone = cfg.fieldMapping.phone;
      fieldMapping.groups = cfg.fieldMapping.groups;
    }
  }

  return {
    configForOAuth2,
    configForOIDC,
    configForLDAP,
    scopesString,
    fieldMapping,
  };
}

// ============================================================
// IDPDetailPage
// ============================================================

export function IDPDetailPage() {
  const { t } = useTranslation();
  const identityProviderStore = useIdentityProviderStore();

  // Reactively read idpId from Vue Router's current route params
  const idpId = useVueState(
    () => router.currentRoute.value.params.idpId as string | undefined
  );

  const [isLoading, setIsLoading] = useState(true);
  const [isUpdating, setIsUpdating] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  // The original IDP from the store
  const [originalIdp, setOriginalIdp] = useState<IdentityProvider | null>(null);

  // Local editable state
  const [localIdp, setLocalIdp] = useState<IdentityProvider | null>(null);
  const [configForOAuth2, setConfigForOAuth2] =
    useState<OAuth2IdentityProviderConfig>(
      create(OAuth2IdentityProviderConfigSchema, {})
    );
  const [configForOIDC, setConfigForOIDC] =
    useState<OIDCIdentityProviderConfig>(
      create(OIDCIdentityProviderConfigSchema, {})
    );
  const [configForLDAP, setConfigForLDAP] =
    useState<LDAPIdentityProviderConfig>(
      create(LDAPIdentityProviderConfigSchema, {})
    );
  const [scopesString, setScopesString] = useState("");
  const [fieldMapping, setFieldMapping] = useState<FieldMappingState>({
    identifier: "",
    displayName: "",
    phone: "",
    groups: "",
  });

  // Track whether secrets have been modified from the initial empty state
  const initializedRef = useRef(false);
  const [isClientSecretModified, setIsClientSecretModified] = useState(false);
  const [isBindPasswordModified, setIsBindPasswordModified] = useState(false);

  const idpName = useMemo(() => {
    if (!idpId) return "";
    return `${idpNamePrefix}${idpId}`;
  }, [idpId]);

  const initializeFromIdp = useCallback((idp: IdentityProvider) => {
    setLocalIdp(cloneDeep(idp));
    setIsClientSecretModified(false);
    setIsBindPasswordModified(false);

    const state = extractConfigState(idp);
    setConfigForOAuth2(state.configForOAuth2);
    setConfigForOIDC(state.configForOIDC);
    setConfigForLDAP(state.configForLDAP);
    setScopesString(state.scopesString);
    setFieldMapping(state.fieldMapping);
    initializedRef.current = true;
  }, []);

  // Fetch the IDP
  useEffect(() => {
    if (!idpName) {
      setIsLoading(false);
      return;
    }

    let cancelled = false;
    setIsLoading(true);
    setOriginalIdp(null);
    setLocalIdp(null);
    initializedRef.current = false;

    identityProviderStore
      .getOrFetchIdentityProviderByName(idpName)
      .then((idp) => {
        if (cancelled) return;
        if (idp) {
          setOriginalIdp(idp);
          initializeFromIdp(idp);
        }
      })
      .catch(() => {
        if (cancelled) return;
        setOriginalIdp(null);
        setLocalIdp(null);
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [idpName, identityProviderStore, initializeFromIdp]);

  const resourceId = useMemo(() => {
    if (!localIdp) return "";
    return getIdentityProviderResourceId(localIdp.name);
  }, [localIdp]);

  const allowEdit = hasWorkspacePermissionV2("bb.identityProviders.update");

  const buildUpdatedIdentityProvider =
    useCallback((): IdentityProvider | null => {
      if (!localIdp) return null;

      const result = create(IdentityProviderSchema, {
        ...localIdp,
        config: create(IdentityProviderConfigSchema, {}),
      });

      if (localIdp.type === IdentityProviderType.OAUTH2) {
        result.config = create(IdentityProviderConfigSchema, {
          config: {
            case: "oauth2Config",
            value: {
              ...configForOAuth2,
              scopes: scopesString.split(" ").filter(Boolean),
              fieldMapping: create(FieldMappingSchema, fieldMapping),
            },
          },
        });
      } else if (localIdp.type === IdentityProviderType.OIDC) {
        result.config = create(IdentityProviderConfigSchema, {
          config: {
            case: "oidcConfig",
            value: {
              ...configForOIDC,
              scopes: scopesString.split(" ").filter(Boolean),
              fieldMapping: create(FieldMappingSchema, fieldMapping),
            },
          },
        });
      } else if (localIdp.type === IdentityProviderType.LDAP) {
        result.config = create(IdentityProviderConfigSchema, {
          config: {
            case: "ldapConfig",
            value: {
              ...configForLDAP,
              fieldMapping: create(FieldMappingSchema, fieldMapping),
            },
          },
        });
      }

      return result;
    }, [
      localIdp,
      configForOAuth2,
      configForOIDC,
      configForLDAP,
      scopesString,
      fieldMapping,
    ]);

  const hasChanges = useMemo(() => {
    if (!originalIdp) return false;
    const current = buildUpdatedIdentityProvider();
    if (!current) return false;
    return !isEqual(current, originalIdp);
  }, [originalIdp, buildUpdatedIdentityProvider]);

  const isFormValid = useMemo(() => {
    if (!localIdp) return false;
    if (!localIdp.title) return false;
    if (!fieldMapping.identifier) return false;

    if (localIdp.type === IdentityProviderType.OAUTH2) {
      const isClientSecretValid = isClientSecretModified
        ? !!configForOAuth2.clientSecret
        : true;
      return !!(
        configForOAuth2.clientId &&
        isClientSecretValid &&
        configForOAuth2.authUrl &&
        configForOAuth2.tokenUrl &&
        configForOAuth2.userInfoUrl
      );
    } else if (localIdp.type === IdentityProviderType.OIDC) {
      const isClientSecretValid = isClientSecretModified
        ? !!configForOIDC.clientSecret
        : true;
      return !!(
        configForOIDC.clientId &&
        isClientSecretValid &&
        configForOIDC.issuer
      );
    } else if (localIdp.type === IdentityProviderType.LDAP) {
      const isBindPasswordValid = isBindPasswordModified
        ? !!configForLDAP.bindPassword
        : true;
      return !!(
        configForLDAP.host &&
        configForLDAP.port &&
        configForLDAP.bindDn &&
        isBindPasswordValid &&
        configForLDAP.baseDn &&
        configForLDAP.userFilter
      );
    }

    return false;
  }, [
    localIdp,
    fieldMapping.identifier,
    configForOAuth2,
    configForOIDC,
    configForLDAP,
    isClientSecretModified,
    isBindPasswordModified,
  ]);

  const canUpdate = allowEdit && hasChanges && isFormValid;

  const handleUpdate = async () => {
    if (!canUpdate) return;
    const updated = buildUpdatedIdentityProvider();
    if (!updated) return;

    setIsUpdating(true);
    try {
      const updatedProvider =
        await identityProviderStore.patchIdentityProvider(updated);

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });

      setOriginalIdp(updatedProvider);
      initializeFromIdp(updatedProvider);
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("identity-provider.identity-provider-update-failed"),
      });
    } finally {
      setIsUpdating(false);
    }
  };

  const handleDiscard = () => {
    if (originalIdp) {
      initializeFromIdp(originalIdp);
    }
  };

  const handleDelete = async () => {
    if (!hasWorkspacePermissionV2("bb.identityProviders.delete")) {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: t("identity-provider.identity-provider-permission-denied"),
      });
      return;
    }

    if (!originalIdp) return;

    try {
      await identityProviderStore.deleteIdentityProvider(originalIdp.name);

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("identity-provider.identity-provider-deleted"),
      });

      router.replace({
        name: WORKSPACE_ROUTE_IDENTITY_PROVIDERS,
      });
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("identity-provider.identity-provider-delete-failed"),
      });
    }

    setShowDeleteConfirm(false);
  };

  const handleOAuth2Update = useCallback(
    (config: OAuth2IdentityProviderConfig) => {
      if (
        initializedRef.current &&
        config.clientSecret !== configForOAuth2.clientSecret
      ) {
        setIsClientSecretModified(true);
      }
      setConfigForOAuth2(config);
    },
    [configForOAuth2.clientSecret]
  );

  const handleOIDCUpdate = useCallback(
    (config: OIDCIdentityProviderConfig) => {
      if (
        initializedRef.current &&
        config.clientSecret !== configForOIDC.clientSecret
      ) {
        setIsClientSecretModified(true);
      }
      setConfigForOIDC(config);
    },
    [configForOIDC.clientSecret]
  );

  const handleLDAPUpdate = useCallback(
    (config: LDAPIdentityProviderConfig) => {
      if (
        initializedRef.current &&
        config.bindPassword !== configForLDAP.bindPassword
      ) {
        setIsBindPasswordModified(true);
      }
      setConfigForLDAP(config);
    },
    [configForLDAP.bindPassword]
  );

  const builtIdp = useMemo(
    () => buildUpdatedIdentityProvider(),
    [buildUpdatedIdentityProvider]
  );

  // Loading state
  if (isLoading) {
    return (
      <div className="w-full h-64 flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent" />
      </div>
    );
  }

  if (!localIdp) {
    return (
      <div className="w-full h-64 flex items-center justify-center text-control-light">
        {t("error-page.not-found")}
      </div>
    );
  }

  const ProviderIcon = getProviderIcon(localIdp.type);

  return (
    <div className="w-full px-4 py-4 flex flex-col gap-y-6">
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

      <div className="divide-y divide-block-border">
        {/* General Section */}
        <div className="pb-6 lg:flex">
          <div className="text-left lg:w-1/4">
            <h1 className="text-3xl font-bold text-gray-900">
              {t("common.general")}
            </h1>
          </div>
          <div className="flex-1 mt-4 lg:px-4 lg:mt-0 flex flex-col gap-y-6">
            <div>
              <p className="text-base font-semibold text-gray-800 mb-2">
                {t("common.type")}
              </p>
              <div className="flex items-center gap-x-3 p-3 bg-gray-50 rounded-md">
                <ProviderIcon className="w-5 h-5 text-gray-600" />
                <span className="text-base font-medium text-gray-800">
                  {identityProviderTypeToString(localIdp.type)}
                </span>
              </div>
              <p className="text-sm text-gray-600 mt-1">
                {t("settings.sso.form.provider-type-readonly-hint")}
              </p>
            </div>

            <div>
              <p className="text-base font-semibold text-gray-800 mb-2">
                {t("settings.sso.form.name")}{" "}
                <span className="text-error">*</span>
              </p>
              <Input
                value={localIdp.title}
                disabled={!allowEdit}
                onChange={(e) =>
                  setLocalIdp({ ...localIdp, title: e.target.value })
                }
                placeholder={t("settings.sso.form.name-description")}
                maxLength={200}
              />
              <div className="mt-1">
                <ResourceIdField
                  resourceType="idp"
                  readonly={true}
                  value={resourceId}
                  resourceName={localIdp.name}
                />
              </div>
            </div>

            <div>
              <p className="text-base font-semibold text-gray-800 mb-2">
                {t("settings.sso.form.domain")}
              </p>
              <Input
                value={localIdp.domain}
                disabled={!allowEdit}
                onChange={(e) =>
                  setLocalIdp({ ...localIdp, domain: e.target.value })
                }
                placeholder={t("settings.sso.form.domain-description")}
              />
              <p className="text-sm text-gray-600 mt-1">
                {t("settings.sso.form.domain-optional-hint")}
              </p>
            </div>
          </div>
        </div>

        {/* Configuration Section */}
        <div className="py-6 lg:flex">
          <div className="text-left lg:w-1/4">
            <h1 className="text-3xl font-bold text-gray-900">
              {t("settings.sso.form.configuration")}
            </h1>
            <p className="text-base text-gray-600 mt-3">
              {t("settings.sso.form.configuration-description")}
            </p>
          </div>
          <div className="flex-1 mt-4 lg:px-4 lg:mt-0 flex flex-col gap-y-6">
            <ProviderConfigForm
              providerType={localIdp.type}
              configForOAuth2={configForOAuth2}
              configForOIDC={configForOIDC}
              configForLDAP={configForLDAP}
              scopesString={scopesString}
              isEditMode={true}
              onUpdateOAuth2={handleOAuth2Update}
              onUpdateOIDC={handleOIDCUpdate}
              onUpdateLDAP={handleLDAPUpdate}
              onUpdateScopes={setScopesString}
            />
          </div>
        </div>

        {/* User Information Mapping Section */}
        <div className="py-6 lg:flex">
          <div className="text-left lg:w-1/4">
            <h1 className="text-3xl font-bold text-gray-900">
              {t("settings.sso.form.user-information-mapping")}
            </h1>
            <p className="text-base text-gray-600 mt-3">
              {t("settings.sso.form.user-information-mapping-description")}
            </p>
          </div>
          <div className="flex-1 mt-4 lg:px-4 lg:mt-0 flex flex-col gap-y-6">
            <FieldMappingForm
              providerType={localIdp.type}
              fieldMapping={fieldMapping}
              onChange={setFieldMapping}
            />
            <div>
              {builtIdp && (
                <TestConnectionButton idp={builtIdp} disabled={!isFormValid} />
              )}
            </div>
          </div>
        </div>

        {/* Actions Section */}
        <div className="py-6">
          <div className="flex flex-row justify-between items-center">
            <Button
              variant="destructive"
              onClick={() => setShowDeleteConfirm(true)}
              disabled={
                !hasWorkspacePermissionV2("bb.identityProviders.delete")
              }
            >
              {t("settings.sso.delete")}
            </Button>
            <div className="gap-x-2 flex flex-row justify-end items-center">
              {hasChanges && (
                <Button variant="outline" onClick={handleDiscard}>
                  {t("common.discard-changes")}
                </Button>
              )}
              <Button
                disabled={!canUpdate || isUpdating}
                onClick={handleUpdate}
              >
                {t("common.update")}
              </Button>
            </div>
          </div>
        </div>
      </div>

      {showDeleteConfirm && (
        <DeleteConfirmDialog
          onConfirm={handleDelete}
          onCancel={() => setShowDeleteConfirm(false)}
        />
      )}
    </div>
  );
}
