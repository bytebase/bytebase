import { useEffect, useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { AuthDivider } from "@/react/components/auth/AuthDivider";
import { AuthFooter } from "@/react/components/auth/AuthFooter";
import { EmailCodeSigninForm } from "@/react/components/auth/EmailCodeSigninForm";
import { IdpBrandIcon } from "@/react/components/auth/IdpBrandIcon";
import { PasswordSigninForm } from "@/react/components/auth/PasswordSigninForm";
import { BytebaseLogo } from "@/react/components/BytebaseLogo";
import { RouterLink } from "@/react/components/RouterLink";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { useIdentityProviderList } from "@/react/hooks/useAppState";
import { resolveWorkspaceName } from "@/react/lib/workspace";
import { router } from "@/react/router";
import { AUTH_SIGNUP_MODULE } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import { idpNamePrefix } from "@/store/modules/v1/common";
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

  const serverInfo = useAppStore((s) => s.serverInfo);
  const isSaaSMode = useAppStore((s) => s.isSaaSMode());
  const activeUserCount = useAppStore((s) => s.activeUserCount());
  const identityProviders = useIdentityProviderList();

  const query = router.currentRoute.value.query;
  const invitedEmail = (query.email as string | undefined) ?? "";

  const disallowSignup =
    !allowSignupProp || !!serverInfo?.restriction?.disallowSignup;

  const separatedIdps = identityProviders.filter(
    (idp) => idp.type !== IdentityProviderType.LDAP
  );
  const groupedIdps = identityProviders.filter(
    (idp) => idp.type === IdentityProviderType.LDAP
  );

  const defaultTab = (() => {
    if (serverInfo?.restriction?.allowEmailCodeSignin) return "email-code";
    if (!serverInfo?.restriction?.disallowPasswordSignin) return "standard";
    if (groupedIdps.length > 0) return groupedIdps[0].name;
    return "standard";
  })();

  // Redirect to signup when an admin setup is needed.
  useEffect(() => {
    if (!initialized) return;
    if (activeUserCount === 0 && !disallowSignup && !isSaaSMode) {
      router.replace({ name: AUTH_SIGNUP_MODULE });
    }
  }, [initialized, activeUserCount, disallowSignup, isSaaSMode]);

  const trySigninWithIdp = async (idp: IdentityProvider) => {
    try {
      await openWindowForSSO(idp, false, query.redirect as string);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Request error occurred",
        description: (error as Error).message,
      });
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

  // Initial load: fetch server info + IDPs + handle `idp` query param.
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
          useAppStore.getState().fetchServerInfo(workspaceName),
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
  if (!serverInfo?.restriction?.disallowPasswordSignin) {
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
  if (serverInfo?.restriction?.allowEmailCodeSignin) {
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

  // The email-code flow signs an unknown email up on the spot, so the page
  // doubles as the signup surface — the copy and the terms line reflect that.
  // On SaaS the restriction reports disallowSignup=true, but that flag only
  // gates the password Signup RPC, not email-code onboarding.
  const combinedSignupSurface =
    methods.length === 1 &&
    methods[0].value === "email-code" &&
    allowSignupProp &&
    (isSaaSMode || !serverInfo?.restriction?.disallowSignup);

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

        {separatedIdps.length > 0 && (
          <div className="flex flex-col gap-y-2">
            {separatedIdps.map((idp) => (
              <Button
                key={idp.name}
                variant="outline"
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

        {isSaaSMode && combinedSignupSurface && (
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
