import { useEffect, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { AUTH_SIGNIN_MODULE } from "@/app/router/handles";
import { AuthDivider } from "@/components/auth/AuthDivider";
import { AuthFooter } from "@/components/auth/AuthFooter";
import { UserPasswordFields } from "@/components/auth/UserPasswordFields";
import { computePasswordValidation } from "@/components/auth/userPasswordValidation";
import { BytebaseLogo } from "@/components/BytebaseLogo";
import { RouterLink } from "@/components/RouterLink";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { useAppStore } from "@/stores/app";
import { isValidEmail } from "@/utils";

export function SignupPage() {
  const { t } = useTranslation();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");
  const [name, setName] = useState("");
  const [nameManuallyEdited, setNameManuallyEdited] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const authenticationInfo = useAppStore((s) => s.authenticationInfo);
  const needsInitialSetup = authenticationInfo
    ? !authenticationInfo.workspace &&
      !authenticationInfo.restriction?.disallowSignup
    : false;

  // This page renders outside any shell, so the workspace bootstrap hasn't
  // populated the app store yet — load authentication info so the setup and
  // signup restriction flags resolve.
  useEffect(() => {
    void useAppStore.getState().loadAuthenticationInfo();
  }, []);

  const [acceptTermsAndPolicy, setAcceptTermsAndPolicy] = useState(
    !needsInitialSetup
  );

  const query = router.currentRoute.value.query;

  useEffect(() => {
    if (!needsInitialSetup && authenticationInfo?.restriction?.disallowSignup) {
      router.replace({ name: AUTH_SIGNIN_MODULE, query });
    }
    if (needsInitialSetup) {
      setAcceptTermsAndPolicy(false);
    }
  }, []);

  const passwordRestriction =
    authenticationInfo?.restriction?.passwordRestriction;
  const validation = computePasswordValidation(
    password,
    passwordConfirm,
    passwordRestriction
  );

  const allowSignup =
    isValidEmail(email) &&
    password.length > 0 &&
    name.length > 0 &&
    !validation.hint &&
    !validation.mismatch &&
    acceptTermsAndPolicy &&
    !authenticationInfo?.restriction?.disallowSignup;

  const onEmailChange = (value: string) => {
    const normalized = value.trim().toLowerCase();
    setEmail(normalized);
    if (!nameManuallyEdited) {
      const parts = normalized.split("@");
      if (parts.length > 0 && parts[0].length > 0) {
        const candidate = parts[0].replace("_", ".");
        const segments = candidate.split(".");
        if (segments.length >= 2) {
          setName(
            [
              segments[0].charAt(0).toUpperCase() + segments[0].slice(1),
              segments[1].charAt(0).toUpperCase() + segments[1].slice(1),
            ].join(" ")
          );
        } else {
          setName(candidate.charAt(0).toUpperCase() + candidate.slice(1));
        }
      }
    }
  };

  const onNameChange = (value: string) => {
    setName(value);
    setNameManuallyEdited(value.trim().length > 0);
  };

  const trySignup = async (e: React.FormEvent) => {
    e.preventDefault();
    if (isLoading) return;
    setIsLoading(true);
    try {
      await useAppStore.getState().signup({ email, password, name });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <div className="h-full flex flex-col justify-center gap-y-4 mx-auto w-full max-w-sm">
        <div>
          <BytebaseLogo className="mx-auto" />
          <h2 className="text-2xl leading-9 font-medium text-main text-center mt-4">
            {needsInitialSetup ? (
              <Trans
                i18nKey="auth.sign-up.admin-title"
                components={{
                  account: <span />,
                }}
              />
            ) : (
              <span>{t("auth.sign-up.title")}</span>
            )}
          </h2>
        </div>

        <div>
          <form className="flex flex-col gap-y-6 mt-6" onSubmit={trySignup}>
            <div>
              <label
                htmlFor="email"
                className="block text-sm font-medium leading-5 text-control"
              >
                {t("common.email")}
                <span className="text-error ml-0.5">*</span>
              </label>
              <div className="mt-1 rounded-md shadow-xs">
                <Input
                  id="email"
                  type="email"
                  required
                  placeholder="jim@example.com"
                  value={email}
                  onChange={(e) => onEmailChange(e.target.value)}
                />
              </div>
            </div>

            <UserPasswordFields
              password={password}
              passwordConfirm={passwordConfirm}
              onPasswordChange={setPassword}
              onPasswordConfirmChange={setPasswordConfirm}
              passwordRestriction={passwordRestriction}
            />

            <div>
              <label
                htmlFor="name"
                className="block text-sm font-medium leading-5 text-control"
              >
                {t("common.username")}
                <span className="text-error ml-0.5">*</span>
              </label>
              <div className="mt-1 rounded-md shadow-xs">
                <Input
                  id="name"
                  required
                  placeholder="Jim Gray"
                  value={name}
                  onChange={(e) => onNameChange(e.target.value)}
                />
              </div>
            </div>

            {needsInitialSetup && (
              <div className="w-full flex flex-row justify-start items-start gap-x-2">
                <Checkbox
                  checked={acceptTermsAndPolicy}
                  id="accept-terms"
                  onCheckedChange={(checked) =>
                    setAcceptTermsAndPolicy(checked)
                  }
                />
                <label
                  htmlFor="accept-terms"
                  className="select-none text-sm text-control"
                >
                  <Trans
                    i18nKey="auth.sign-up.accept-terms-and-policy"
                    components={{
                      terms: (
                        <a
                          href="https://www.bytebase.com/terms?source=console"
                          className="text-accent"
                        />
                      ),
                      policy: (
                        <a
                          href="https://www.bytebase.com/privacy?source=console"
                          className="text-accent"
                        />
                      ),
                    }}
                  />
                </label>
              </div>
            )}

            <div className="w-full">
              <Button
                type="submit"
                size="lg"
                className="w-full"
                disabled={!allowSignup || isLoading}
              >
                {needsInitialSetup
                  ? t("auth.sign-up.create-admin-account")
                  : t("common.sign-up")}
              </Button>
            </div>
          </form>
        </div>

        {!needsInitialSetup && (
          <AuthDivider className="mt-6">
            <span className="pl-2 bg-white text-control">
              {t("auth.sign-up.existing-user")}
            </span>
            <RouterLink
              to={{ name: AUTH_SIGNIN_MODULE, query }}
              className="accent-link px-2 bg-white"
            >
              {t("common.sign-in")}
            </RouterLink>
          </AuthDivider>
        )}
      </div>

      <AuthFooter />
    </>
  );
}
