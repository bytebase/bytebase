import { create } from "@bufbuild/protobuf";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { ConnectError } from "@connectrpc/connect";
import { QRCodeSVG } from "qrcode.react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { userServiceClientConnect } from "@/api";
import { router } from "@/app/router";
import { ACCOUNT_ROUTE, AUTH_2FA_SETUP_MODULE } from "@/app/router/handles";
import { LearnMoreLink } from "@/components/LearnMoreLink";
import { Button } from "@/components/ui/button";
import { OtpInput } from "@/components/ui/otp-input";
import { StepIndicator } from "@/components/ui/step-indicator";
import { useCurrentUser } from "@/hooks/useAppState";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import {
  ConfirmRecoveryCodesRequestSchema,
  EnableMFARequestSchema,
  StartMFAEnrollmentRequestSchema,
} from "@/types/proto-es/v1/user_service_pb";
import { RecoveryCodesView } from "./RecoveryCodesView";
import { TwoFactorSecretModal } from "./TwoFactorSecretModal";

const ISSUER_NAME = "Bytebase";
const DIGITS = 6;

// What StartMFAEnrollment handed back. Nothing here is on the User resource:
// the secret and the codes exist only in that response, and the expiry is the
// server's, not a duration this page assumes.
interface Enrollment {
  otpSecret: string;
  recoveryCodes: string[];
  expireTime: Timestamp | undefined;
}

const SETUP_AUTH_APP_STEP = 0;
const DOWNLOAD_RECOVERY_CODES_STEP = 1;
type Step = typeof SETUP_AUTH_APP_STEP | typeof DOWNLOAD_RECOVERY_CODES_STEP;

interface TwoFactorSetupPageProps {
  cancelAction?: () => void;
}

export function TwoFactorSetupPage({ cancelAction }: TwoFactorSetupPageProps) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const fetchCurrentUser = useAppStore((state) => state.fetchCurrentUser);
  const [enrollment, setEnrollment] = useState<Enrollment | undefined>();

  const [currentStep, setCurrentStep] = useState<Step>(SETUP_AUTH_APP_STEP);
  const [showSecretModal, setShowSecretModal] = useState(false);
  const [otpCodes, setOtpCodes] = useState<string[]>([]);
  const [recoveryCodesDownloaded, setRecoveryCodesDownloaded] = useState(false);
  const [timeRemaining, setTimeRemaining] = useState("5:00");
  const [isExpired, setIsExpired] = useState(false);
  const [isExpiringSoon, setIsExpiringSoon] = useState(false);

  const countdownRef = useRef<ReturnType<typeof setInterval> | null>(null);
  // Keep a ref to the enrollment so the interval callback always reads fresh state
  const enrollmentRef = useRef(enrollment);
  enrollmentRef.current = enrollment;

  const stopCountdown = useCallback(() => {
    if (countdownRef.current) {
      clearInterval(countdownRef.current);
      countdownRef.current = null;
    }
  }, []);

  const updateCountdown = useCallback(() => {
    const expireTime = enrollmentRef.current?.expireTime;
    if (!expireTime) {
      setIsExpired(true);
      setTimeRemaining("0:00");
      return;
    }

    const remaining = Number(expireTime.seconds) * 1000 - Date.now();

    if (remaining <= 0) {
      setIsExpired(true);
      setTimeRemaining("0:00");
      setIsExpiringSoon(false);
      stopCountdown();
    } else {
      setIsExpired(false);
      const minutes = Math.floor(remaining / 60000);
      const seconds = Math.floor((remaining % 60000) / 1000);
      setTimeRemaining(`${minutes}:${seconds.toString().padStart(2, "0")}`);
      setIsExpiringSoon(remaining < 60000);
    }
  }, [stopCountdown]);

  const startCountdown = useCallback(() => {
    updateCountdown();
    stopCountdown();
    countdownRef.current = setInterval(updateCountdown, 1000);
  }, [updateCountdown, stopCountdown]);

  const startEnrollment = useCallback(async () => {
    const response = await userServiceClientConnect.startMFAEnrollment(
      create(StartMFAEnrollmentRequestSchema, { name: currentUser.name })
    );
    setEnrollment({
      otpSecret: response.otpSecret,
      recoveryCodes: [...response.recoveryCodes],
      expireTime: response.expireTime,
    });
  }, [currentUser.name]);

  // On mount: mint an enrollment and start counting down to its expiry
  useEffect(() => {
    startEnrollment().then(() => {
      startCountdown();
    });
    return stopCountdown;
  }, [startEnrollment, startCountdown, stopCountdown]);

  const otpauthUrl = `otpauth://totp/${ISSUER_NAME}:${currentUser.email}?algorithm=SHA1&digits=${DIGITS}&issuer=${ISSUER_NAME}&period=30&secret=${enrollment?.otpSecret ?? ""}`;

  const verifyOTPCode = useCallback(
    async (codes: string[]) => {
      try {
        await userServiceClientConnect.enableMFA(
          create(EnableMFARequestSchema, {
            name: currentUser.name,
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
    [currentUser.name]
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
    await startEnrollment();
    startCountdown();
  }, [startEnrollment, startCountdown]);

  const cancelSetup = useCallback(() => {
    if (cancelAction) {
      cancelAction();
    } else {
      router.replace({
        name: ACCOUNT_ROUTE,
      });
    }
  }, [cancelAction]);

  const tryFinishSetup = useCallback(async () => {
    // Confirming the codes is what makes the factor live: the secret and the
    // codes that recover it start existing in the same write.
    await userServiceClientConnect.confirmRecoveryCodes(
      create(ConfirmRecoveryCodesRequestSchema, { name: currentUser.name })
    );
    await fetchCurrentUser();
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("two-factor.messages.2fa-enabled"),
    });

    if (router.currentRoute.value.name === AUTH_2FA_SETUP_MODULE) {
      router.replace({ path: "/" });
    } else {
      router.replace({ name: ACCOUNT_ROUTE });
    }
  }, [currentUser.name, fetchCurrentUser, t]);

  const allowNext =
    currentStep === SETUP_AUTH_APP_STEP
      ? otpCodes.filter((v) => v).length === DIGITS && !isExpired
      : recoveryCodesDownloaded;

  const steps = [
    { title: t("two-factor.setup-steps.setup-auth-app.self") },
    { title: t("two-factor.setup-steps.download-recovery-codes.self") },
  ];

  return (
    <div className="px-4 py-4">
      <p className="text-sm text-gray-500 mb-4">
        {t("two-factor.description")}
        <LearnMoreLink
          href="https://docs.bytebase.com/administration/2fa?source=console"
          className="ml-1 text-accent"
        />
      </p>

      {/* Step indicator */}
      <StepIndicator
        className="mb-8"
        steps={steps}
        currentIndex={currentStep}
      />

      {/* Step content */}
      {currentStep === SETUP_AUTH_APP_STEP && (
        <div className="w-full max-w-2xl mx-auto flex flex-col justify-start items-start gap-y-4 my-8">
          <p>{t("two-factor.setup-steps.setup-auth-app.description")}</p>
          <div
            className={`w-full border rounded-sm p-3 ${
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
            recoveryCodes={enrollment?.recoveryCodes ?? []}
            onDownload={() => setRecoveryCodesDownloaded(true)}
          />
        </div>
      )}

      {/* Navigation buttons */}
      <div className="flex items-center justify-between mt-4">
        <Button appearance="outline" onClick={cancelSetup}>
          {t("common.cancel")}
        </Button>
        <div className="flex items-center gap-x-2">
          {currentStep === DOWNLOAD_RECOVERY_CODES_STEP && (
            <Button appearance="outline" onClick={handleBack}>
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
        secret={enrollment?.otpSecret ?? ""}
        open={showSecretModal}
        onClose={() => setShowSecretModal(false)}
      />
    </div>
  );
}
