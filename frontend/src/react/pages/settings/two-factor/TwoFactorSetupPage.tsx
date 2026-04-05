import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import type { ConnectError } from "@connectrpc/connect";
import { Check, ExternalLink } from "lucide-react";
import { QRCodeSVG } from "qrcode.react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { OtpInput } from "@/react/components/ui/otp-input";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { AUTH_2FA_SETUP_MODULE } from "@/router/auth";
import { SETTING_ROUTE_PROFILE } from "@/router/dashboard/workspaceSetting";
import { pushNotification, useCurrentUserV1, useUserStore } from "@/store";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";
import { RecoveryCodesView } from "./RecoveryCodesView";
import { TwoFactorSecretModal } from "./TwoFactorSecretModal";

const ISSUER_NAME = "Bytebase";
const DIGITS = 6;
const MFA_TEMP_SECRET_EXPIRATION = 5 * 60 * 1000; // 5 minutes

const SETUP_AUTH_APP_STEP = 0;
const DOWNLOAD_RECOVERY_CODES_STEP = 1;
type Step = typeof SETUP_AUTH_APP_STEP | typeof DOWNLOAD_RECOVERY_CODES_STEP;

interface TwoFactorSetupPageProps {
  cancelAction?: () => void;
}

export function TwoFactorSetupPage({ cancelAction }: TwoFactorSetupPageProps) {
  const { t } = useTranslation();
  const userStore = useUserStore();
  const currentUser = useVueState(() => useCurrentUserV1().value);

  const [currentStep, setCurrentStep] = useState<Step>(SETUP_AUTH_APP_STEP);
  const [showSecretModal, setShowSecretModal] = useState(false);
  const [otpCodes, setOtpCodes] = useState<string[]>([]);
  const [recoveryCodesDownloaded, setRecoveryCodesDownloaded] = useState(false);
  const [timeRemaining, setTimeRemaining] = useState("5:00");
  const [isExpired, setIsExpired] = useState(false);
  const [isExpiringSoon, setIsExpiringSoon] = useState(false);

  const countdownRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const updateCountdown = useCallback(() => {
    if (!currentUser.tempOtpSecretCreatedTime) {
      setIsExpired(true);
      setTimeRemaining("0:00");
      return;
    }

    const createdAt =
      Number(currentUser.tempOtpSecretCreatedTime.seconds) * 1000;
    const now = Date.now();
    const elapsed = now - createdAt;
    const remaining = MFA_TEMP_SECRET_EXPIRATION - elapsed;

    if (remaining <= 0) {
      setIsExpired(true);
      setTimeRemaining("0:00");
      setIsExpiringSoon(false);
      if (countdownRef.current) {
        clearInterval(countdownRef.current);
        countdownRef.current = null;
      }
    } else {
      setIsExpired(false);
      const minutes = Math.floor(remaining / 60000);
      const seconds = Math.floor((remaining % 60000) / 1000);
      setTimeRemaining(`${minutes}:${seconds.toString().padStart(2, "0")}`);
      setIsExpiringSoon(remaining < 60000);
    }
  }, [currentUser.tempOtpSecretCreatedTime]);

  const startCountdown = useCallback(() => {
    updateCountdown();
    if (countdownRef.current) {
      clearInterval(countdownRef.current);
    }
    countdownRef.current = setInterval(updateCountdown, 1000);
  }, [updateCountdown]);

  const regenerateTempMfaSecret = useCallback(async () => {
    await userStore.updateUser(
      create(UpdateUserRequestSchema, {
        user: {
          name: currentUser.name,
        },
        updateMask: create(FieldMaskSchema, {
          paths: [],
        }),
        regenerateTempMfaSecret: true,
      })
    );
  }, [currentUser.name, userStore]);

  // On mount: regenerate secret and start countdown
  useEffect(() => {
    regenerateTempMfaSecret().then(() => {
      startCountdown();
    });
    return () => {
      if (countdownRef.current) {
        clearInterval(countdownRef.current);
        countdownRef.current = null;
      }
    };
  }, []);

  const otpauthUrl = `otpauth://totp/${ISSUER_NAME}:${currentUser.email}?algorithm=SHA1&digits=${DIGITS}&issuer=${ISSUER_NAME}&period=30&secret=${currentUser.tempOtpSecret}`;

  const verifyOTPCode = useCallback(
    async (codes: string[]) => {
      try {
        await userStore.updateUser(
          create(UpdateUserRequestSchema, {
            user: {
              name: currentUser.name,
            },
            updateMask: create(FieldMaskSchema, {
              paths: [],
            }),
            otpCode: codes.join(""),
          })
        );
      } catch (error) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: (error as ConnectError).message,
        });
        return false;
      }
      return true;
    },
    [currentUser.name, userStore]
  );

  const handleOtpFinish = useCallback(
    async (value: string[]) => {
      setOtpCodes(value);
      const result = await verifyOTPCode(value);
      if (result && currentStep === SETUP_AUTH_APP_STEP) {
        setCurrentStep(DOWNLOAD_RECOVERY_CODES_STEP);
      }
    },
    [verifyOTPCode, currentStep]
  );

  const handleNext = useCallback(async () => {
    const result = await verifyOTPCode(otpCodes);
    if (result) {
      setCurrentStep(DOWNLOAD_RECOVERY_CODES_STEP);
    }
  }, [verifyOTPCode, otpCodes]);

  const handleBack = useCallback(() => {
    setOtpCodes([]);
    setCurrentStep(SETUP_AUTH_APP_STEP);
  }, []);

  const handleRegenerateSecret = useCallback(async () => {
    setOtpCodes([]);
    await regenerateTempMfaSecret();
    startCountdown();
  }, [regenerateTempMfaSecret, startCountdown]);

  const cancelSetup = useCallback(() => {
    if (cancelAction) {
      cancelAction();
    } else {
      router.replace({
        name: SETTING_ROUTE_PROFILE,
      });
    }
  }, [cancelAction]);

  const tryFinishSetup = useCallback(async () => {
    await userStore.updateUser(
      create(UpdateUserRequestSchema, {
        user: {
          name: currentUser.name,
          mfaEnabled: true,
        },
        updateMask: create(FieldMaskSchema, {
          paths: ["mfa_enabled"],
        }),
      })
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("two-factor.messages.2fa-enabled"),
    });

    if (router.currentRoute.value.name === AUTH_2FA_SETUP_MODULE) {
      router.replace({ path: "/" });
    } else {
      router.replace({ name: SETTING_ROUTE_PROFILE });
    }
  }, [currentUser.name, t, userStore]);

  const allowNext =
    currentStep === SETUP_AUTH_APP_STEP
      ? otpCodes.filter((v) => v).length === DIGITS && !isExpired
      : recoveryCodesDownloaded;

  const steps = [
    t("two-factor.setup-steps.setup-auth-app.self"),
    t("two-factor.setup-steps.download-recovery-codes.self"),
  ];

  return (
    <div className="px-4 py-4">
      <p className="text-sm text-gray-500 mb-4">
        {t("two-factor.description")}
        <a
          href="https://docs.bytebase.com/administration/2fa?source=console"
          target="_blank"
          rel="noopener noreferrer"
          className="ml-1 inline-flex items-center gap-x-0.5 text-accent hover:underline"
        >
          {t("common.learn-more")}
          <ExternalLink className="w-3 h-3" />
        </a>
      </p>

      {/* Step indicator */}
      <div className="flex items-center gap-x-4 mb-8">
        {steps.map((title, index) => (
          <div key={index} className="flex items-center gap-x-2">
            <div
              className={`flex items-center justify-center w-7 h-7 rounded-full text-sm font-medium ${
                index < currentStep
                  ? "bg-accent text-white"
                  : index === currentStep
                    ? "bg-accent text-white"
                    : "bg-gray-200 text-gray-600"
              }`}
            >
              {index < currentStep ? <Check className="w-4 h-4" /> : index + 1}
            </div>
            <span
              className={`text-sm font-medium ${
                index === currentStep ? "text-accent" : "text-gray-500"
              }`}
            >
              {title}
            </span>
            {index < steps.length - 1 && (
              <div className="w-12 h-px bg-gray-300 ml-2" />
            )}
          </div>
        ))}
      </div>

      {/* Step content */}
      {currentStep === SETUP_AUTH_APP_STEP && (
        <div className="w-full max-w-2xl mx-auto flex flex-col justify-start items-start gap-y-4 my-8">
          <p>{t("two-factor.setup-steps.setup-auth-app.description")}</p>
          <div
            className={`w-full border rounded-md p-3 ${
              isExpired || isExpiringSoon
                ? "bg-red-50 border-red-200"
                : "bg-yellow-50 border-yellow-200"
            }`}
          >
            <div className="flex items-center justify-between">
              <p
                className={`text-sm ${
                  isExpired || isExpiringSoon
                    ? "text-red-800"
                    : "text-yellow-800"
                }`}
              >
                {isExpired
                  ? t("two-factor.setup-steps.setup-auth-app.expired-notice")
                  : t("two-factor.setup-steps.setup-auth-app.time-remaining", {
                      time: timeRemaining,
                    })}
              </p>
              {isExpired && (
                <button
                  type="button"
                  className="ml-3 px-3 py-1 text-sm font-medium text-white bg-blue-600 rounded-sm hover:bg-blue-700"
                  onClick={handleRegenerateSecret}
                >
                  {t("two-factor.setup-steps.setup-auth-app.regenerate")}
                </button>
              )}
            </div>
          </div>
          <p className="text-2xl">
            {t("two-factor.setup-steps.setup-auth-app.scan-qr-code.self")}
          </p>
          <p>
            {(() => {
              const raw = t(
                "two-factor.setup-steps.setup-auth-app.scan-qr-code.description"
              );
              const placeholder = "{{action}}";
              const idx = raw.indexOf(placeholder);
              if (idx === -1) return raw;
              return (
                <>
                  {raw.slice(0, idx)}
                  <span
                    className={
                      !showSecretModal
                        ? "cursor-pointer text-blue-600"
                        : undefined
                    }
                    onClick={() => setShowSecretModal(true)}
                  >
                    {t(
                      "two-factor.setup-steps.setup-auth-app.scan-qr-code.enter-the-text"
                    )}
                  </span>
                  {raw.slice(idx + placeholder.length)}
                </>
              );
            })()}
          </p>
          <div className="w-full flex flex-col justify-center items-center pb-8">
            <QRCodeSVG value={otpauthUrl} size={150} />
            <span className="mt-4 mb-2 text-sm font-medium">
              {t("two-factor.setup-steps.setup-auth-app.verify-code")}
            </span>
            <OtpInput
              value={otpCodes}
              onChange={setOtpCodes}
              onFinish={handleOtpFinish}
              length={DIGITS}
            />
          </div>
        </div>
      )}

      {currentStep === DOWNLOAD_RECOVERY_CODES_STEP && (
        <div className="w-full max-w-2xl mx-auto">
          <RecoveryCodesView
            recoveryCodes={[...currentUser.tempRecoveryCodes]}
            onDownload={() => setRecoveryCodesDownloaded(true)}
          />
        </div>
      )}

      {/* Navigation buttons */}
      <div className="flex items-center justify-between mt-4">
        <Button variant="outline" onClick={cancelSetup}>
          {t("common.cancel")}
        </Button>
        <div className="flex items-center gap-x-2">
          {currentStep === DOWNLOAD_RECOVERY_CODES_STEP && (
            <Button variant="outline" onClick={handleBack}>
              {t("common.back")}
            </Button>
          )}
          {currentStep === SETUP_AUTH_APP_STEP && (
            <Button disabled={!allowNext} onClick={handleNext}>
              {t("common.next")}
            </Button>
          )}
          {currentStep === DOWNLOAD_RECOVERY_CODES_STEP && (
            <Button disabled={!allowNext} onClick={tryFinishSetup}>
              {t("two-factor.setup-steps.recovery-codes-saved")}
            </Button>
          )}
        </div>
      </div>

      <TwoFactorSecretModal
        secret={currentUser.tempOtpSecret}
        open={showSecretModal}
        onClose={() => setShowSecretModal(false)}
      />
    </div>
  );
}
