import { create } from "@bufbuild/protobuf";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { OtpInput } from "@/react/components/ui/otp-input";
import { pushNotification, useAuthStore } from "@/store";
import {
  type LoginRequest,
  LoginRequestSchema,
} from "@/types/proto-es/v1/auth_service_pb";
import { isValidEmail, resolveWorkspaceName } from "@/utils";

type Props = {
  readonly loading: boolean;
  readonly onSignin: (request: LoginRequest) => void;
};

export function EmailCodeSigninForm({ loading, onSignin }: Props) {
  const { t } = useTranslation();
  const [step, setStep] = useState<"email" | "code">("email");
  const [email, setEmail] = useState("");
  const [codeParts, setCodeParts] = useState<string[]>([]);
  const [sending, setSending] = useState(false);
  const [emailFromQuery, setEmailFromQuery] = useState(false);
  const [resendCountdown, setResendCountdown] = useState(0);
  const countdownTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    const url = new URL(window.location.href);
    const params = new URLSearchParams(url.search);
    const q = params.get("email") ?? "";
    if (q) {
      setEmail(q);
      setEmailFromQuery(true);
    }
    return () => {
      if (countdownTimerRef.current) clearInterval(countdownTimerRef.current);
    };
  }, []);

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

  const sendCode = async () => {
    if (!isValidEmail(email) || sending || resendCountdown > 0) return;
    setSending(true);
    try {
      await useAuthStore().sendEmailLoginCode(email, resolveWorkspaceName());
      setStep("code");
      startCountdown();
    } catch (e) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("auth.sign-in.failed-to-send-code", { error: String(e) }),
      });
    } finally {
      setSending(false);
    }
  };

  const submitCode = (parts: string[]) => {
    const code = parts.join("");
    if (!email || code.length !== 6) return;
    onSignin(
      create(LoginRequestSchema, {
        email,
        emailCode: code,
        workspace: resolveWorkspaceName(),
      })
    );
  };

  const handleSubmit = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (step === "email") {
      await sendCode();
      return;
    }
    submitCode(codeParts);
  };

  const allowSignin = !!email && codeParts.join("").length === 6;

  return (
    <form className="flex flex-col gap-y-6 px-1" onSubmit={handleSubmit}>
      <div>
        <label
          htmlFor="email-code-email"
          className="block text-sm font-medium leading-5 text-control"
        >
          {t("common.email")}
          <span className="text-error ml-0.5">*</span>
        </label>
        <div className="mt-1 rounded-md shadow-xs">
          <Input
            id="email-code-email"
            type="email"
            autoComplete="email"
            placeholder="jim@example.com"
            required
            value={email}
            disabled={step === "code" || emailFromQuery}
            onChange={(e) => setEmail(e.target.value)}
          />
        </div>
      </div>

      {step === "code" && (
        <div className="flex flex-col gap-y-2">
          <label className="block text-sm font-medium leading-5 text-control">
            {t("auth.sign-in.verification-code")}
            <span className="text-error ml-0.5">*</span>
          </label>
          <div className="text-sm text-control-light">
            {t("auth.sign-in.code-sent-hint", { email })}
          </div>
          <OtpInput
            value={codeParts}
            onChange={setCodeParts}
            onFinish={submitCode}
            length={6}
          />
          <div className="flex items-center justify-end">
            <button
              type="button"
              className="text-sm text-accent disabled:text-control-light disabled:cursor-not-allowed"
              disabled={resendCountdown > 0}
              onClick={sendCode}
            >
              {resendCountdown > 0
                ? t("auth.sign-in.resend-in", { seconds: resendCountdown })
                : t("auth.sign-in.resend-code")}
            </button>
          </div>
        </div>
      )}

      <div className="w-full">
        {step === "email" ? (
          <Button
            type="button"
            size="lg"
            className="w-full"
            disabled={!isValidEmail(email) || sending}
            onClick={sendCode}
          >
            {t("auth.sign-in.send-code")}
          </Button>
        ) : (
          <Button
            type="submit"
            size="lg"
            className="w-full"
            disabled={!allowSignin || loading}
          >
            {t("common.sign-in")}
          </Button>
        )}
      </div>
    </form>
  );
}
