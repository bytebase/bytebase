import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { AuthFooter } from "@/react/components/auth/AuthFooter";
import { DemoSigninForm } from "@/react/components/auth/DemoSigninForm";
import { EmailCodeSigninForm } from "@/react/components/auth/EmailCodeSigninForm";
import { PasswordSigninForm } from "@/react/components/auth/PasswordSigninForm";
import { BytebaseLogo } from "@/react/components/BytebaseLogo";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { AUTH_SIGNUP_MODULE } from "@/router/auth";
import {
  pushNotification,
  useActuatorV1Store,
  useAuthStore,
  useIdentityProviderStore,
} from "@/store";
import { idpNamePrefix } from "@/store/modules/v1/common";
import type { LoginRequest } from "@/types/proto-es/v1/auth_service_pb";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";
import { openWindowForSSO, resolveWorkspaceName } from "@/utils";

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

  const serverInfo = useVueState(() => useActuatorV1Store().serverInfo);
  const isDemo = useVueState(() => useActuatorV1Store().isDemo);
  const isSaaSMode = useVueState(() => useActuatorV1Store().isSaaSMode);
  const activeUserCount = useVueState(
    () => useActuatorV1Store().activeUserCount
  );
  const identityProviders = useVueState(
    () => useIdentityProviderStore().identityProviderList
  );

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

  const showSignInForm =
    !serverInfo?.restriction?.disallowPasswordSignin ||
    groupedIdps.length > 0 ||
    serverInfo?.restriction?.allowEmailCodeSignin;

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
      router.push({ name: AUTH_SIGNUP_MODULE, replace: true });
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
      await useAuthStore().login({
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
      const idpStore = useIdentityProviderStore();
      const actuatorStore = useActuatorV1Store();
      try {
        const [idpList] = await Promise.all([
          idpStore.fetchIdentityProviderList(workspaceName),
          actuatorStore.fetchServerInfo(workspaceName),
        ]);
        if (idpList.length === 0 && workspaceName) {
          await idpStore.fetchIdentityProviderList();
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
        const idp = idpStore.identityProviderList.find(
          (i: IdentityProvider) => i.name === name
        );
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

  return (
    <>
      <div className="h-full flex flex-col justify-center mx-auto w-full max-w-sm">
        <BytebaseLogo className="mx-auto mb-8" />

        {invitedEmail && (
          <Alert
            variant="info"
            className="mb-4 mt-4"
            description={t("auth.sign-in.invited-email", {
              email: invitedEmail,
            })}
          />
        )}

        {showSignInForm && (
          <div className="rounded-sm border border-control-border bg-white p-4">
            <Tabs defaultValue={defaultTab}>
              <TabsList>
                {!serverInfo?.restriction?.disallowPasswordSignin && (
                  <TabsTrigger value="standard">Standard</TabsTrigger>
                )}
                {serverInfo?.restriction?.allowEmailCodeSignin && (
                  <TabsTrigger value="email-code">
                    {t("auth.sign-in.email-code-tab")}
                  </TabsTrigger>
                )}
                {groupedIdps.map((idp) => (
                  <TabsTrigger key={idp.name} value={idp.name}>
                    {idp.title}
                  </TabsTrigger>
                ))}
              </TabsList>
              {!serverInfo?.restriction?.disallowPasswordSignin && (
                <TabsPanel value="standard" className="pt-3">
                  {isDemo ? (
                    <DemoSigninForm loading={isLoading} onSignin={trySignin} />
                  ) : (
                    <>
                      <PasswordSigninForm
                        loading={isLoading}
                        onSignin={trySignin}
                      />
                      {!disallowSignup && (
                        <div className="mt-3 flex justify-center items-center text-sm text-control gap-x-2">
                          <span>{t("auth.sign-in.new-user")}</span>
                          <a
                            href="#"
                            className="accent-link"
                            onClick={(e) => {
                              e.preventDefault();
                              router.push({
                                name: AUTH_SIGNUP_MODULE,
                                query,
                              });
                            }}
                          >
                            {t("common.sign-up")}
                          </a>
                        </div>
                      )}
                    </>
                  )}
                </TabsPanel>
              )}
              {serverInfo?.restriction?.allowEmailCodeSignin && (
                <TabsPanel value="email-code" className="pt-3">
                  <EmailCodeSigninForm
                    loading={isLoading}
                    onSignin={trySignin}
                  />
                </TabsPanel>
              )}
              {groupedIdps.map((idp) => (
                <TabsPanel key={idp.name} value={idp.name} className="pt-3">
                  <PasswordSigninForm
                    loading={isLoading}
                    showForgotPassword={false}
                    credentialLabel={t("common.username")}
                    credentialPlaceholder="jim"
                    credentialInputType="text"
                    credentialAutocomplete="username"
                    onSignin={(req) => trySignin({ ...req, idpName: idp.name })}
                  />
                </TabsPanel>
              ))}
            </Tabs>
          </div>
        )}

        {separatedIdps.length > 0 && (
          <div className="mb-3 px-1">
            {showSignInForm && (
              <div className="relative my-4">
                <div
                  aria-hidden="true"
                  className="absolute inset-0 flex items-center"
                >
                  <div className="w-full border-t border-control-border" />
                </div>
                <div className="relative flex justify-center text-sm">
                  <span className="px-2 bg-white text-control">
                    {t("common.or")}
                  </span>
                </div>
              </div>
            )}
            {separatedIdps.map((idp) => (
              <div key={idp.name} className="w-full mb-2">
                <Button
                  variant="outline"
                  size="lg"
                  className="w-full"
                  onClick={() => trySigninWithIdp(idp)}
                >
                  {t("auth.sign-in.sign-in-with-idp", { idp: idp.title })}
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>

      {footerOverride ?? (hideFooter ? null : <AuthFooter />)}
    </>
  );
}
