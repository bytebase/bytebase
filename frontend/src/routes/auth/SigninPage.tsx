import { ExternalLink } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { AUTH_SIGNUP_MODULE } from "@/app/router/handles";
import { AuthDivider } from "@/components/auth/AuthDivider";
import { AuthFooter } from "@/components/auth/AuthFooter";
import { EmailCodeSigninForm } from "@/components/auth/EmailCodeSigninForm";
import { IdpBrandIcon } from "@/components/auth/IdpBrandIcon";
import { PasswordSigninForm } from "@/components/auth/PasswordSigninForm";
import { BytebaseLogo } from "@/components/BytebaseLogo";
import { RouterLink } from "@/components/RouterLink";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsPanel, TabsTrigger } from "@/components/ui/tabs";
import { useIdentityProviderList } from "@/hooks/useAppState";
import { resolveWorkspaceName } from "@/lib/workspace";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { idpNamePrefix } from "@/stores/modules/v1/common";
import type { LoginRequest } from "@/types/proto-es/v1/auth_service_pb";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";
import { openWindowForSSO } from "@/utils";

export type SigninPageProps = {
  readonly redirect?: boolean;
  readonly redirectUrl?: string;
  readonly allowSignup?: boolean;
  readonly hideFooter?: boolean;
  readonly footerOverride?: React.ReactNode;
};

type SSOFailure = {
  idpName: string;
};

const ADMIN_RECOVERY_URL =
  "https://docs.bytebase.com/get-started/self-host/admin-recovery?source=console";
const SUPPORT_URL = "https://docs.bytebase.com/faq#how-to-reach-us";

function queryString(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function queryFailure(query: Record<string, unknown>): SSOFailure | undefined {
  const idpName = queryString(query.ssoError);
  if (!idpName) return undefined;
  return { idpName };
}

export function SigninPage(props: SigninPageProps) {
  const {
    redirect = true,
    redirectUrl,
    allowSignup: allowSignupProp = true,
    hideFooter = false,
    footerOverride,
  } = props;
  const { t } = useTranslation();
  const [isLoading, setIsLoading] = useState(false);
  const [initialized, setInitialized] = useState(false);
  const query = router.currentRoute.value.query;
  const initialSSOFailure = queryFailure(query);
  const [ssoFailure, setSSOFailure] = useState<SSOFailure | undefined>(
    initialSSOFailure
  );

  const authenticationInfo = useAppStore((s) => s.authenticationInfo);
  const identityProviders = useIdentityProviderList();
  const isSaaSMode = useAppStore((s) => s.isSaaSMode());

  const invitedEmail = (query.email as string | undefined) ?? "";

  const disallowSignup =
    !allowSignupProp || !!authenticationInfo?.restriction?.disallowSignup;
  const needsInitialSetup = authenticationInfo
    ? !authenticationInfo.workspace &&
      !authenticationInfo.restriction?.disallowSignup
    : false;

  const separatedIdps = identityProviders.filter(
    (idp) => idp.type !== IdentityProviderType.LDAP
  );
  const groupedIdps = identityProviders.filter(
    (idp) => idp.type === IdentityProviderType.LDAP
  );

  const defaultTab = (() => {
    if (authenticationInfo?.restriction?.allowEmailCodeSignin)
      return "email-code";
    if (!authenticationInfo?.restriction?.disallowPasswordSignin)
      return "standard";
    if (groupedIdps.length > 0) return groupedIdps[0].name;
    return "standard";
  })();

  // A self-hosted server without a workspace needs its initial administrator.
  useEffect(() => {
    if (!initialized) return;
    if (needsInitialSetup && !disallowSignup) {
      router.replace({ name: AUTH_SIGNUP_MODULE });
    }
  }, [initialized, needsInitialSetup, disallowSignup]);

  const trySigninWithIdp = async (idp: IdentityProvider) => {
    try {
      await openWindowForSSO(idp, false, query.redirect as string);
    } catch {
      setSSOFailure({ idpName: idp.name });
    }
  };

  const trySignin = async (request: LoginRequest) => {
    if (isLoading) return;
    setIsLoading(true);
    try {
      await useAppStore.getState().login({
        request,
        redirect,
        redirectUrl,
      });
    } finally {
      setIsLoading(false);
    }
  };

  // Initial load: fetch authentication info + IDPs + handle `idp` query param.
  // Ref guard is critical — the `?idp=<name>` path triggers an SSO redirect
  // via `trySigninWithIdp`, which must not fire twice under StrictMode.
  const initRef = useRef(false);
  useEffect(() => {
    if (initRef.current) return;
    initRef.current = true;
    (async () => {
      const workspaceName = resolveWorkspaceName();
      const listIdentityProviders =
        useAppStore.getState().listIdentityProviders;
      try {
        const [idpList] = await Promise.all([
          listIdentityProviders(workspaceName),
          useAppStore.getState().fetchAuthenticationInfo(workspaceName),
        ]);
        if (idpList.length === 0 && workspaceName) {
          await listIdentityProviders();
        }
      } catch (error) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: "Request error occurred",
          description: (error as Error).message,
        });
      }
      const idpQuery = query.idp;
      if (idpQuery) {
        const name = `${idpNamePrefix}${idpQuery}`;
        const idp = useAppStore
          .getState()
          .identityProviderList()
          .find((i: IdentityProvider) => i.name === name);
        if (idp) {
          // On success this navigates away; on failure `trySigninWithIdp`
          // pushes a notification and we still need to show the form so the
          // user can recover (e.g., pick a different IdP or retry).
          await trySigninWithIdp(idp);
        }
      }
      setInitialized(true);
    })();
  }, []);

  if (!initialized) {
    return (
      <div className="inset-0 absolute flex flex-row justify-center items-center">
        <div className="h-6 w-6 border-2 border-control-border border-t-accent rounded-full animate-spin" />
      </div>
    );
  }

  const methods: {
    value: string;
    label: string;
    panel: React.ReactNode;
  }[] = [];
  if (!authenticationInfo?.restriction?.disallowPasswordSignin) {
    methods.push({
      value: "standard",
      label: t("auth.sign-in.standard-tab"),
      panel: (
        <>
          <PasswordSigninForm loading={isLoading} onSignin={trySignin} />
          {!disallowSignup && (
            <div className="mt-3 flex justify-center items-center text-sm text-control gap-x-2">
              <span>{t("auth.sign-in.new-user")}</span>
              <RouterLink
                to={{
                  name: AUTH_SIGNUP_MODULE,
                  query,
                }}
                className="accent-link"
              >
                {t("common.sign-up")}
              </RouterLink>
            </div>
          )}
        </>
      ),
    });
  }
  if (authenticationInfo?.restriction?.allowEmailCodeSignin) {
    methods.push({
      value: "email-code",
      label: t("auth.sign-in.email-code-tab"),
      panel: <EmailCodeSigninForm loading={isLoading} onSignin={trySignin} />,
    });
  }
  for (const idp of groupedIdps) {
    methods.push({
      value: idp.name,
      label: idp.title,
      panel: (
        <PasswordSigninForm
          loading={isLoading}
          showForgotPassword={false}
          credentialLabel={t("common.username")}
          credentialPlaceholder="jim"
          credentialInputType="text"
          credentialAutocomplete="username"
          onSignin={(req) => trySignin({ ...req, idpName: idp.name })}
        />
      ),
    });
  }

  const emailCodeEnabled = methods.some(({ value }) => value === "email-code");
  const combinedSignupSurface =
    methods.length === 1 &&
    methods[0].value === "email-code" &&
    allowSignupProp &&
    !authenticationInfo?.restriction?.disallowSignup;
  const showTerms = allowSignupProp && emailCodeEnabled;
  const disallowPasswordSignin =
    !!authenticationInfo?.restriction?.disallowPasswordSignin;
  const hasNoPasswordFallback =
    disallowPasswordSignin &&
    !authenticationInfo?.restriction?.allowEmailCodeSignin;
  const noSigninMethodAvailable =
    hasNoPasswordFallback && identityProviders.length === 0;
  const failedIdp = ssoFailure
    ? identityProviders.find((idp) => idp.name === ssoFailure.idpName)
    : undefined;
  const failedIdpTitle =
    failedIdp?.title ?? ssoFailure?.idpName.replace(idpNamePrefix, "") ?? "";
  const recoveryLinks = (
    <ul className="mt-2 list-disc pl-5">
      <li>{t("auth.sign-in.sso-failure.contact-administrator")}</li>
      <li>
        {isSaaSMode ? (
          <a
            href={SUPPORT_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="font-medium underline"
          >
            {t("auth.sign-in.sso-failure.contact-support")}
            <ExternalLink className="ml-1 inline size-3" />
          </a>
        ) : (
          <>
            {t("auth.sign-in.sso-failure.administrators")}{" "}
            <a
              href={ADMIN_RECOVERY_URL}
              target="_blank"
              rel="noopener noreferrer"
              className="font-medium underline"
            >
              {t("auth.sign-in.sso-failure.recovery-guide")}
              <ExternalLink className="ml-1 inline size-3" />
            </a>
          </>
        )}
      </li>
    </ul>
  );

  return (
    <>
      <div className="h-full flex flex-col justify-center mx-auto w-full max-w-sm">
        <BytebaseLogo className="mx-auto mb-2" />
        <p className="mb-8 text-center text-sm text-control">
          {combinedSignupSurface
            ? t("auth.sign-in.sign-in-or-create")
            : t("auth.sign-in.sign-in-to-account")}
        </p>

        {invitedEmail && (
          <Alert
            variant="info"
            className="mb-4"
            description={t("auth.sign-in.invited-email", {
              email: invitedEmail,
            })}
          />
        )}

        {noSigninMethodAvailable && (
          <Alert
            variant="info"
            className="mb-4"
            title={t("auth.sign-in.no-method-available.title")}
            description={t("auth.sign-in.no-method-available.description")}
          >
            {recoveryLinks}
          </Alert>
        )}

        {ssoFailure && (
          <Alert
            variant="warning"
            className="mb-4"
            title={t("auth.sign-in.sso-failure.title", {
              idp: failedIdpTitle,
            })}
            description={t("auth.sign-in.sso-failure.description")}
          >
            {disallowPasswordSignin && (
              <div className="mt-2">
                <p>
                  {hasNoPasswordFallback
                    ? t("auth.sign-in.sso-failure.no-fallback")
                    : t("auth.sign-in.sso-failure.password-disabled")}
                </p>
                {recoveryLinks}
              </div>
            )}
          </Alert>
        )}

        {separatedIdps.length > 0 && (
          <div className="flex flex-col gap-y-2">
            {separatedIdps.map((idp) => (
              <Button
                key={idp.name}
                appearance="outline"
                size="lg"
                className="w-full"
                onClick={() => trySigninWithIdp(idp)}
              >
                <IdpBrandIcon idp={idp} className="size-4 shrink-0" />
                {t("auth.sign-in.continue-with-idp", { idp: idp.title })}
              </Button>
            ))}
          </div>
        )}

        {separatedIdps.length > 0 && methods.length > 0 && (
          <AuthDivider className="my-4">
            <span className="px-2 bg-white text-control">{t("common.or")}</span>
          </AuthDivider>
        )}

        {methods.length === 1 && methods[0].panel}
        {methods.length > 1 && (
          <div className="rounded-sm border border-control-border bg-white p-4">
            <Tabs defaultValue={defaultTab}>
              <TabsList>
                {methods.map((method) => (
                  <TabsTrigger key={method.value} value={method.value}>
                    {method.label}
                  </TabsTrigger>
                ))}
              </TabsList>
              {methods.map((method) => (
                <TabsPanel
                  key={method.value}
                  value={method.value}
                  className="pt-3"
                >
                  {method.panel}
                </TabsPanel>
              ))}
            </Tabs>
          </div>
        )}

        {showTerms && (
          <p className="mt-6 text-center text-xs text-control-light leading-5">
            <Trans
              i18nKey="auth.sign-in.tos"
              components={{
                // The anchor children are fallbacks — Trans replaces them
                // with the localized text inside <terms>/<privacy> tags.
                terms: (
                  <a
                    href="https://www.bytebase.com/terms"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="underline hover:text-control"
                  >
                    Terms of Service
                  </a>
                ),
                privacy: (
                  <a
                    href="https://www.bytebase.com/privacy"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="underline hover:text-control"
                  >
                    Privacy Policy
                  </a>
                ),
              }}
            />
          </p>
        )}
      </div>

      {footerOverride ?? (hideFooter ? null : <AuthFooter />)}
    </>
  );
}
