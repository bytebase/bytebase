import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import logoFull from "@/assets/logo-full.svg";
import { authServiceClientConnect } from "@/connect";
import { AuthDivider } from "@/react/components/auth/AuthDivider";
import { RouterLink } from "@/react/components/RouterLink";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { resolveWorkspaceName } from "@/react/lib/workspace";
import { router } from "@/react/router";
import {
  AUTH_PASSWORD_RESET_MODULE,
  AUTH_SIGNIN_MODULE,
} from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import { isValidEmail } from "@/utils";

export function PasswordForgotPage() {
  const { t } = useTranslation();
  const [email, setEmail] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const serverInfo = useAppStore((state) => state.serverInfo);
  const passwordResetEnabled =
    serverInfo?.restriction?.passwordResetEnabled ?? false;
  const disallowPasswordSignin =
    serverInfo?.restriction?.disallowPasswordSignin ?? false;

  // This page renders outside any shell, so the workspace bootstrap hasn't
  // populated the app store yet — load server info so the password-reset
  // restriction flags resolve.
  useEffect(() => {
    void useAppStore.getState().loadServerInfo();
  }, []);

  useEffect(() => {
    if (disallowPasswordSignin) {
      router.replace({
        name: AUTH_SIGNIN_MODULE,
        query: router.currentRoute.value.query,
      });
    }
  }, [disallowPasswordSignin]);

  const canSubmit = isValidEmail(email) && !isLoading;

  const onSubmit = async () => {
    if (!canSubmit) return;
    setIsLoading(true);
    try {
      await authServiceClientConnect.requestPasswordReset({
        email,
        workspace: resolveWorkspaceName(),
      });
      router.push({
        name: AUTH_PASSWORD_RESET_MODULE,
        query: { ...router.currentRoute.value.query, email },
      });
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("auth.password-forget.failed-to-send-code"),
      });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="h-full flex flex-col justify-center mx-auto w-full max-w-sm">
      <div>
        <img src={logoFull} alt="Bytebase" className="h-12 w-auto" />
        <h2 className="mt-6 text-3xl leading-9 font-extrabold text-main">
          {t("auth.password-forget.title")}
        </h2>
      </div>

      <div className="mt-8">
        <div className="mt-6 flex flex-col gap-y-4">
          {!passwordResetEnabled ? (
            <Alert
              variant="warning"
              description={t("auth.password-forget.selfhost")}
            />
          ) : (
            <>
              <div>
                <label
                  htmlFor="forgot-email"
                  className="block text-sm font-medium leading-5 text-control"
                >
                  {t("common.email")}
                </label>
                <div className="mt-1">
                  <Input
                    id="forgot-email"
                    type="email"
                    autoComplete="email"
                    placeholder="jim@example.com"
                    required
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    onKeyUp={(e) => {
                      if (e.key === "Enter") onSubmit();
                    }}
                  />
                </div>
              </div>
              <Button
                size="lg"
                className="w-full"
                disabled={!canSubmit}
                onClick={onSubmit}
              >
                {t("auth.password-forget.send-reset-code")}
              </Button>
            </>
          )}
        </div>
      </div>

      <AuthDivider className="mt-6">
        <RouterLink
          to={{
            name: AUTH_SIGNIN_MODULE,
            query: router.currentRoute.value.query,
          }}
          className="accent-link bg-white px-2"
        >
          {t("auth.password-forget.return-to-sign-in")}
        </RouterLink>
      </AuthDivider>
    </div>
  );
}
