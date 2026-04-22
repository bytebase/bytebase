import { create } from "@bufbuild/protobuf";
import { KeyRound, Smartphone } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import logoFull from "@/assets/logo-full.svg";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { OtpInput } from "@/react/components/ui/otp-input";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { useAuthStore } from "@/store";
import { LoginRequestSchema } from "@/types/proto-es/v1/auth_service_pb";
import { resolveWorkspaceName } from "@/utils";

type MFAType = "OTP" | "RECOVERY_CODE";

export function MultiFactorPage() {
  const { t } = useTranslation();
  const [mfaType, setMfaType] = useState<MFAType>("OTP");
  const [otpCodes, setOtpCodes] = useState<string[]>([]);
  const [recoveryCode, setRecoveryCode] = useState("");

  const mfaTempToken = useVueState(
    () =>
      (router.currentRoute.value.query.mfaTempToken as string | undefined) ?? ""
  );

  const challengeDescription = useMemo(() => {
    if (mfaType === "OTP") {
      return t("multi-factor.other-methods.use-auth-app.description");
    }
    if (mfaType === "RECOVERY_CODE") {
      return t("multi-factor.other-methods.use-recovery-code.description");
    }
    return "";
  }, [mfaType, t]);

  const challenge = async (codes?: string[]) => {
    const effectiveOtp = (codes ?? otpCodes).join("");
    const request = create(LoginRequestSchema, {
      mfaTempToken,
      workspace: resolveWorkspaceName(),
      ...(mfaType === "OTP" ? { otpCode: effectiveOtp } : { recoveryCode }),
    });
    await useAuthStore().login({ request, redirect: true });
  };

  const onOtpFinish = (value: string[]) => {
    setOtpCodes(value);
    challenge(value);
  };

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    challenge();
  };

  return (
    <div className="mx-auto max-w-2xl h-full py-6 flex flex-col justify-center items-center">
      <div className="w-80 p-8 py-6 shadow-sm border border-control-border rounded-sm bg-white">
        <img
          src={logoFull}
          alt="Bytebase"
          className="h-12 w-auto mx-auto mb-8"
        />
        <form
          className="w-full mt-4 h-auto flex flex-col justify-start items-center"
          onSubmit={onSubmit}
        >
          {mfaType === "OTP" ? (
            <>
              <Smartphone className="w-8 h-auto opacity-60" />
              <p className="my-2 mb-4">{t("multi-factor.auth-code")}</p>
              <OtpInput
                value={otpCodes}
                onChange={setOtpCodes}
                onFinish={onOtpFinish}
              />
            </>
          ) : (
            <>
              <KeyRound className="w-8 h-auto opacity-60" />
              <p className="my-2 mb-4">{t("multi-factor.recovery-code")}</p>
              <Input
                value={recoveryCode}
                onChange={(e) => setRecoveryCode(e.target.value)}
                placeholder="XXXXXXXXXX"
                className="w-full"
              />
            </>
          )}
          <div className="w-full mt-4">
            <Button type="submit" className="w-full">
              {t("common.verify")}
            </Button>
          </div>
          <p className="textinfolabel mt-2">{challengeDescription}</p>
        </form>
        <hr className="my-3" />
        <div className="text-sm mb-2">
          <p>{t("multi-factor.other-methods.self")}:</p>
          <ul className="list-disc list-inside pl-2 pt-1">
            {mfaType !== "OTP" && (
              <li>
                <button
                  type="button"
                  className="accent-link"
                  onClick={() => setMfaType("OTP")}
                >
                  {t("multi-factor.other-methods.use-auth-app.self")}
                </button>
              </li>
            )}
            {mfaType !== "RECOVERY_CODE" && (
              <li>
                <button
                  type="button"
                  className="accent-link"
                  onClick={() => setMfaType("RECOVERY_CODE")}
                >
                  {t("multi-factor.other-methods.use-recovery-code.self")}
                </button>
              </li>
            )}
          </ul>
        </div>
      </div>
    </div>
  );
}
