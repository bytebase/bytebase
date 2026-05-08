import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import logoFull from "@/assets/logo-full.svg";
import { authServiceClientConnect } from "@/connect";
import { UserPasswordFields } from "@/react/components/auth/UserPasswordFields";
import { computePasswordValidation } from "@/react/components/auth/userPasswordValidation";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { OtpInput } from "@/react/components/ui/otp-input";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { AUTH_SIGNIN_MODULE } from "@/router/auth";
import {
  pushNotification,
  useActuatorV1Store,
  useAuthStore,
  useCurrentUserV1,
  useUserStore,
} from "@/store";
import {
  LoginRequestSchema,
  ResetPasswordRequestSchema,
} from "@/types/proto-es/v1/auth_service_pb";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";
import { resolveWorkspaceName } from "@/utils";

export function PasswordResetPage() {
  const { t } = useTranslation();
  const [email, setEmail] = useState("");
  const [codeParts, setCodeParts] = useState<string[]>([]);
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");
  const [resendCountdown, setResendCountdown] = useState(60);
  const countdownTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const query = router.currentRoute.value.query;
  const codeMode = !!query.email;

  const passwordRestriction = useVueState(
    () => useActuatorV1Store().serverInfo?.restriction?.passwordRestriction
  );
  const disallowPasswordSignin = useVueState(
    () =>
      useActuatorV1Store().serverInfo?.restriction?.disallowPasswordSignin ??
      false
  );
  const requireResetPassword = useVueState(
    () => useAuthStore().requireResetPassword
  );
  const currentUser = useVueState(() => useCurrentUserV1().value);

  const redirectQuery = () => {
    const q = new URLSearchParams(window.location.search);
    return q.get("redirect") || "/";
  };

  const startCountdown = () => {
    setResendCountdown(60);
    if (countdownTimerRef.current) clearInterval(countdownTimerRef.current);
    countdownTimerRef.current = setInterval(() => {
      setResendCountdown((prev) => {
        if (prev <= 1) {
          if (countdownTimerRef.current) {
            clearInterval(countdownTimerRef.current);
            countdownTimerRef.current = null;
          }
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  useEffect(() => {
    if (codeMode) {
      if (disallowPasswordSignin) {
        router.replace({ name: AUTH_SIGNIN_MODULE, query });
        return;
      }
      setEmail(query.email as string);
      startCountdown();
      return () => {
        if (countdownTimerRef.current) clearInterval(countdownTimerRef.current);
      };
    }
    if (!requireResetPassword) {
      router.replace(redirectQuery());
    }
    return () => {
      if (countdownTimerRef.current) clearInterval(countdownTimerRef.current);
    };
  }, []);

  const validation = computePasswordValidation(
    password,
    passwordConfirm,
    passwordRestriction
  );

  const allowConfirm = (() => {
    if (!password) return false;
    if (codeMode && (!email || codeParts.join("").length !== 6)) return false;
    return !validation.hint && !validation.mismatch;
  })();

  const resendCode = async () => {
    if (resendCountdown > 0 || !email) return;
    try {
      await authServiceClientConnect.requestPasswordReset({
        email,
        workspace: resolveWorkspaceName(),
      });
      startCountdown();
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("auth.password-forget.failed-to-send-code"),
      });
    }
  };

  const onConfirm = async () => {
    if (codeMode) {
      try {
        await authServiceClientConnect.resetPassword(
          create(ResetPasswordRequestSchema, {
            email,
            code: codeParts.join(""),
            newPassword: password,
          })
        );
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
        await useAuthStore().login({
          request: create(LoginRequestSchema, {
            email,
            password,
            workspace: resolveWorkspaceName(),
          }),
        });
      } catch {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("auth.password-reset.invalid-or-expired-code"),
        });
      }
      return;
    }

    // Forced-reset mode
    if (!currentUser) return;
    const patch = { ...currentUser, password };
    await useUserStore().updateUser(
      create(UpdateUserRequestSchema, {
        user: patch,
        updateMask: create(FieldMaskSchema, { paths: ["password"] }),
      })
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    useAuthStore().setRequireResetPassword(false);
    router.replace(redirectQuery());
  };

  return (
    <div className="h-full flex flex-col justify-center mx-auto w-full max-w-sm">
      <img src={logoFull} alt="Bytebase" className="h-12 w-auto" />
      <h2 className="mt-6 text-3xl leading-9 font-extrabold text-main">
        {t("auth.password-reset.title")}
      </h2>
      <p className="textinfo mt-2">{t("auth.password-reset.content")}</p>

      <div className="mt-8 flex flex-col gap-y-6">
        {codeMode && (
          <>
            <div>
              <label className="block text-sm font-medium leading-5 text-control">
                {t("common.email")}
                <span className="text-error ml-0.5">*</span>
              </label>
              <Input
                className="mt-1"
                type="email"
                autoComplete="email"
                value={email}
                disabled
                required
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div>
              <label className="block text-sm font-medium leading-5 text-control">
                {t("auth.password-reset.code-label")}
                <span className="text-error ml-0.5">*</span>
              </label>
              <div className="mt-1">
                <OtpInput
                  value={codeParts}
                  onChange={setCodeParts}
                  length={6}
                />
              </div>
              <div className="mt-2 flex items-center justify-end">
                <button
                  type="button"
                  className="text-sm text-accent disabled:text-control-light disabled:cursor-not-allowed"
                  disabled={resendCountdown > 0}
                  onClick={resendCode}
                >
                  {resendCountdown > 0
                    ? t("auth.sign-in.resend-in", {
                        seconds: resendCountdown,
                      })
                    : t("auth.sign-in.resend-code")}
                </button>
              </div>
            </div>
          </>
        )}

        <UserPasswordFields
          password={password}
          passwordConfirm={passwordConfirm}
          onPasswordChange={setPassword}
          onPasswordConfirmChange={setPasswordConfirm}
          passwordRestriction={passwordRestriction}
        />

        <Button
          size="lg"
          className="w-full"
          disabled={!allowConfirm}
          onClick={onConfirm}
        >
          {t("common.confirm")}
        </Button>
      </div>

      {codeMode && (
        <div className="mt-6 relative">
          <div
            aria-hidden="true"
            className="absolute inset-0 flex items-center"
          >
            <div className="w-full border-t border-control-border" />
          </div>
          <div className="relative flex justify-center text-sm">
            <a
              href="#"
              className="accent-link bg-white px-2"
              onClick={(e) => {
                e.preventDefault();
                router.push({ name: AUTH_SIGNIN_MODULE });
              }}
            >
              {t("auth.password-forget.return-to-sign-in")}
            </a>
          </div>
        </div>
      )}
    </div>
  );
}
