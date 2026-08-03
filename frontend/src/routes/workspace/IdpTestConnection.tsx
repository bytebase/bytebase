import { create } from "@bufbuild/protobuf";
import type { ConnectError } from "@connectrpc/connect";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { identityProviderServiceClientConnect } from "@/api";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { pushNotification } from "@/stores";
import type { OAuthWindowEventPayload } from "@/types";
import type {
  IdentityProvider,
  TestIdentityProviderResponse,
} from "@/types/proto-es/v1/idp_service_pb";
import {
  CreateIdentityProviderRequestSchema,
  IdentityProviderType,
  TestIdentityProviderRequestSchema,
} from "@/types/proto-es/v1/idp_service_pb";
import { openWindowForSSO } from "@/utils";

// ============================================================
// KeyValueBox
// ============================================================

function KeyValueBox({ entries }: { entries: Record<string, string> }) {
  return (
    <div className="flex flex-col gap-y-1">
      {Object.entries(entries).map(([key, value]) => (
        <div
          key={key}
          className="grid grid-cols-3 gap-2 py-1 border-b border-block-border last:border-b-0"
        >
          <div
            className="text-sm font-medium text-control truncate"
            title={key}
          >
            {key}
          </div>
          <div className="col-span-2 text-sm text-main break-all" title={value}>
            {value}
          </div>
        </div>
      ))}
    </div>
  );
}

// ============================================================
// TestConnectionResultDialog
// ============================================================

function TestConnectionResultDialog({
  response,
  isLdap,
  connectivityOnly,
  onClose,
}: {
  response: TestIdentityProviderResponse;
  isLdap: boolean;
  connectivityOnly: boolean;
  onClose: () => void;
}) {
  const { t } = useTranslation();

  return (
    <Dialog open onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="w-[32rem] max-w-[calc(100vw-2rem)] p-6">
        <div className="flex items-center gap-x-2">
          <div className="size-6 text-success">&#10003;</div>
          <DialogTitle>
            {t("identity-provider.test-connection-success")}
          </DialogTitle>
        </div>

        {connectivityOnly ? (
          <p className="mt-4 text-sm text-control-light">
            {t("identity-provider.ldap-connectivity-verified")}
          </p>
        ) : (
          <div className="mt-4 flex flex-col gap-y-4">
            <p className="text-sm text-control-light">
              {t("identity-provider.userinfo-description")}
            </p>
            <div className="bg-control-bg rounded-xs p-4">
              {Object.keys(response.userInfo).length === 0 ? (
                <div className="text-sm text-control-light italic">
                  {t("identity-provider.no-user-info-mapped")}
                </div>
              ) : (
                <KeyValueBox entries={response.userInfo} />
              )}
            </div>

            <p className="text-sm text-control-light">
              {isLdap
                ? t("identity-provider.attributes-description")
                : t("identity-provider.claims-description")}
            </p>
            <div className="bg-control-bg rounded-xs p-4">
              {Object.keys(response.claims).length === 0 ? (
                <div className="text-sm text-control-light italic">
                  {t("identity-provider.no-claims")}
                </div>
              ) : (
                <KeyValueBox entries={response.claims} />
              )}
            </div>
          </div>
        )}

        <div className="flex justify-end mt-4">
          <Button onClick={onClose}>{t("common.close")}</Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ============================================================
// LdapTestDialog
// ============================================================

function LdapTestDialog({
  testing,
  onTest,
  onClose,
}: {
  testing: boolean;
  onTest: (credentials?: { username: string; password: string }) => void;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const canSignIn = username.trim() !== "" && password !== "";

  return (
    <Dialog open onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="w-[32rem] max-w-[calc(100vw-2rem)] p-6">
        <DialogTitle>{t("identity-provider.ldap-test-title")}</DialogTitle>
        <form
          className="mt-4 flex flex-col gap-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            if (canSignIn) {
              onTest({ username: username.trim(), password });
            }
          }}
        >
          <p className="text-sm text-control-light">
            {t("identity-provider.ldap-test-description")}
          </p>
          <div className="flex flex-col gap-y-1">
            <label className="text-sm font-medium text-control">
              {t("common.username")}
            </label>
            <Input
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="off"
            />
          </div>
          <div className="flex flex-col gap-y-1">
            <label className="text-sm font-medium text-control">
              {t("common.password")}
            </label>
            <Input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="new-password"
            />
          </div>
          <div className="flex justify-end gap-x-2">
            <Button
              type="button"
              appearance="outline"
              disabled={testing}
              onClick={onClose}
            >
              {t("common.cancel")}
            </Button>
            <Button
              type="button"
              appearance="outline"
              disabled={testing}
              onClick={() => onTest(undefined)}
            >
              {t("identity-provider.ldap-test-connectivity-only")}
            </Button>
            <Button type="submit" disabled={testing || !canSignIn}>
              {t("identity-provider.ldap-test-sign-in")}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ============================================================
// TestConnectionButton
// ============================================================

export function TestConnectionButton({
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
  const [testResult, setTestResult] = useState<{
    response: TestIdentityProviderResponse;
    connectivityOnly: boolean;
  } | null>(null);
  const [showLdapDialog, setShowLdapDialog] = useState(false);
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
      setTestResult({ response, connectivityOnly: false });
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

  const testLdap = async (credentials?: {
    username: string;
    password: string;
  }) => {
    if (testingRef.current) return;
    try {
      testingRef.current = true;
      setTesting(true);
      const request = create(TestIdentityProviderRequestSchema, {
        identityProvider: idpRef.current,
        ...(credentials && {
          context: { case: "ldapContext" as const, value: credentials },
        }),
      });
      const response =
        await identityProviderServiceClientConnect.testIdentityProvider(
          request
        );
      setShowLdapDialog(false);
      setTestResult({ response, connectivityOnly: !credentials });
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
  };

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
      setShowLdapDialog(true);
    }
  };

  return (
    <>
      <Button
        appearance="outline"
        disabled={disabled || testing}
        onClick={testConnection}
      >
        {t("identity-provider.test-connection")}
      </Button>
      {showLdapDialog && (
        <LdapTestDialog
          testing={testing}
          onTest={testLdap}
          onClose={() => setShowLdapDialog(false)}
        />
      )}
      {testResult && (
        <TestConnectionResultDialog
          response={testResult.response}
          isLdap={idp.type === IdentityProviderType.LDAP}
          connectivityOnly={testResult.connectivityOnly}
          onClose={() => setTestResult(null)}
        />
      )}
    </>
  );
}
