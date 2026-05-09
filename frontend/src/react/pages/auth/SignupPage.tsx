import { useEffect, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { AuthFooter } from "@/react/components/auth/AuthFooter";
import { UserPasswordFields } from "@/react/components/auth/UserPasswordFields";
import { computePasswordValidation } from "@/react/components/auth/userPasswordValidation";
import { BytebaseLogo } from "@/react/components/BytebaseLogo";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { AUTH_SIGNIN_MODULE } from "@/router/auth";
import { useActuatorV1Store, useAuthStore } from "@/store";
import { isValidEmail } from "@/utils";

export function SignupPage() {
  const { t } = useTranslation();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");
  const [name, setName] = useState("");
  const [nameManuallyEdited, setNameManuallyEdited] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const activeUserCount = useVueState(
    () => useActuatorV1Store().activeUserCount
  );
  const serverInfo = useVueState(() => useActuatorV1Store().serverInfo);
  const needAdminSetup = activeUserCount === 0;

  const [acceptTermsAndPolicy, setAcceptTermsAndPolicy] = useState(
    !needAdminSetup
  );

  const query = router.currentRoute.value.query;

  useEffect(() => {
    if (!needAdminSetup && serverInfo?.restriction?.disallowSignup) {
      router.replace({ name: AUTH_SIGNIN_MODULE, query });
    }
    if (needAdminSetup) {
      setAcceptTermsAndPolicy(false);
    }
  }, []);

  const passwordRestriction = serverInfo?.restriction?.passwordRestriction;
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
    !serverInfo?.restriction?.disallowSignup;

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
      await useAuthStore().signup({ email, password, name });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <div className="h-full flex flex-col justify-center mx-auto w-full max-w-sm">
        <div>
          <BytebaseLogo className="mx-auto mb-8" />
          <h2 className="text-2xl leading-9 font-medium text-main mt-4">
            {needAdminSetup ? (
              <p className="text-accent font-semibold text-center">
                <Trans
                  i18nKey="auth.sign-up.admin-title"
                  components={{
                    account: <span className="text-accent font-semibold" />,
                  }}
                />
              </p>
            ) : (
              <span>{t("auth.sign-up.title")}</span>
            )}
          </h2>
        </div>

        <div className="mt-8">
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

            {needAdminSetup && (
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
                {needAdminSetup
                  ? t("auth.sign-up.create-admin-account")
                  : t("common.sign-up")}
              </Button>
            </div>
          </form>
        </div>

        {!needAdminSetup && (
          <div className="mt-6 relative">
            <div
              aria-hidden="true"
              className="absolute inset-0 flex items-center"
            >
              <div className="w-full border-t border-control-border" />
            </div>
            <div className="relative flex justify-center text-sm">
              <span className="pl-2 bg-white text-control">
                {t("auth.sign-up.existing-user")}
              </span>
              <a
                href="#"
                className="accent-link px-2 bg-white"
                onClick={(e) => {
                  e.preventDefault();
                  router.push({ name: AUTH_SIGNIN_MODULE, query });
                }}
              >
                {t("common.sign-in")}
              </a>
            </div>
          </div>
        )}
      </div>

      <AuthFooter />
    </>
  );
}
