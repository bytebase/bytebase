import { create } from "@bufbuild/protobuf";
import type { ConnectError } from "@connectrpc/connect";
import { QRCodeSVG } from "qrcode.react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { userServiceClientConnect } from "@/api";
import { router } from "@/app/router";
import { ACCOUNT_ROUTE, AUTH_2FA_SETUP_MODULE } from "@/app/router/handles";
import {
  buildCredentialProof,
  CredentialProofInput,
  credentialProofCallOptions,
  useCredentialProofMode,
} from "@/components/CredentialProofInput";
import { LearnMoreLink } from "@/components/LearnMoreLink";
import { Button } from "@/components/ui/button";
import { OtpInput } from "@/components/ui/otp-input";
import { StepIndicator } from "@/components/ui/step-indicator";
import { useCurrentUser } from "@/hooks/useAppState";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import type {
  StartMFAEnrollmentResponse,
  User,
} from "@/types/proto-es/v1/user_service_pb";
import {
  ConfirmRecoveryCodesRequestSchema,
  EnableMFARequestSchema,
  StartMFAEnrollmentRequestSchema,
} from "@/types/proto-es/v1/user_service_pb";
import { RecoveryCodesView } from "./RecoveryCodesView";
import { TwoFactorSecretModal } from "./TwoFactorSecretModal";

const ISSUER_NAME = "Bytebase";
const DIGITS = 6;

const SETUP_AUTH_APP_STEP = 0;
const DOWNLOAD_RECOVERY_CODES_STEP = 1;
type Step = typeof SETUP_AUTH_APP_STEP | typeof DOWNLOAD_RECOVERY_CODES_STEP;

interface TwoFactorSetupPageProps {
  cancelAction?: () => void;
}

// The split enrollment (docs/design/reauthenticate-credential-changes.md):
// StartMFAEnrollment mints the pending secret and recovery codes — the only
// response that ever carries them — EnableMFA verifies the new device (and,
// for a rotation, promotes the secret), and ConfirmRecoveryCodes promotes the
// rest once the user confirms they saved the codes. Every step that changes
// live credential material carries a CredentialProof.
export function TwoFactorSetupPage({ cancelAction }: TwoFactorSetupPageProps) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const setCurrentUser = useAppStore((state) => state.setCurrentUser);
  const proofMode = useCredentialProofMode();
  // Captured at mount: EnableMFA flips mfaEnabled mid-flow on a rotation, and
  // the confirm step's request shape depends on which flow this is.
  const [rotation] = useState(currentUser.mfaEnabled);

  const [enrollment, setEnrollment] = useState<
    StartMFAEnrollmentResponse | undefined
  >(undefined);
  const [currentStep, setCurrentStep] = useState<Step>(SETUP_AUTH_APP_STEP);
  const [showSecretModal, setShowSecretModal] = useState(false);
  const [otpCodes, setOtpCodes] = useState<string[]>([]);
  const [proofValue, setProofValue] = useState("");
  const [confirmOtpCodes, setConfirmOtpCodes] = useState<string[]>([]);
  const [confirmProofValue, setConfirmProofValue] = useState("");
  const [recoveryCodesDownloaded, setRecoveryCodesDownloaded] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [timeRemaining, setTimeRemaining] = useState("5:00");
  const [isExpired, setIsExpired] = useState(false);
  const [isExpiringSoon, setIsExpiringSoon] = useState(false);

  const countdownRef = useRef<ReturnType<typeof setInterval> | null>(null);
  // Keep a ref to the enrollment so the interval callback always reads fresh state
  const enrollmentRef = useRef(enrollment);
  enrollmentRef.current = enrollment;

  // EnableMFA needs no proof at all for a first-time enrollment on an account
  // whose only possible proof is a single-use emailed code — that code is
  // spent once, at the confirm step, where the mutation actually happens.
  const needProofAtEnable = proofMode !== "email";

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
    setEnrollment(response);
  }, [currentUser.name]);

  // On mount: mint an enrollment and start counting down to its expiry
  useEffect(() => {
    startEnrollment().then(() => {
      startCountdown();
    });
    return stopCountdown;
  }, [startEnrollment, startCountdown, stopCountdown]);

  const otpauthUrl = `otpauth://totp/${ISSUER_NAME}:${currentUser.email}?algorithm=SHA1&digits=${DIGITS}&issuer=${ISSUER_NAME}&period=30&secret=${enrollment?.otpSecret ?? ""}`;

  const notifyError = useCallback((error: unknown) => {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: (error as ConnectError).message,
    });
  }, []);

  const verifyNewDevice = useCallback(
    async (codes: string[]) => {
      if (!enrollment) return false;
      try {
        await userServiceClientConnect.enableMFA(
          create(EnableMFARequestSchema, {
            name: currentUser.name,
            otpCode: codes.join(""),
            credential: needProofAtEnable
              ? buildCredentialProof(proofMode, proofValue)
              : undefined,
            pendingVersion: enrollment.pendingVersion,
          }),
          credentialProofCallOptions()
        );
      } catch (error) {
        notifyError(error);
        return false;
      }
      return true;
    },
    [
      enrollment,
      currentUser.name,
      needProofAtEnable,
      proofMode,
      proofValue,
      notifyError,
    ]
  );

  const handleOtpFinish = useCallback(
    async (value: string[]) => {
      setOtpCodes(value);
      if (needProofAtEnable && !proofValue.trim()) return;
      const result = await verifyNewDevice(value);
      if (result && currentStep === SETUP_AUTH_APP_STEP) {
        setCurrentStep(DOWNLOAD_RECOVERY_CODES_STEP);
      }
    },
    [verifyNewDevice, currentStep, needProofAtEnable, proofValue]
  );

  const handleNext = useCallback(async () => {
    const result = await verifyNewDevice(otpCodes);
    if (result) {
      setCurrentStep(DOWNLOAD_RECOVERY_CODES_STEP);
    }
  }, [verifyNewDevice, otpCodes]);

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
    if (!enrollment || submitting) return;
    setSubmitting(true);
    let enabled: User;
    try {
      const freshOtp = confirmOtpCodes.join("");
      if (rotation) {
        // Promotion happens here, so the proof binds here — and it is the
        // factor being replaced, which is still the live one: EnableMFA
        // verified the new device without promoting it. A fresh code, since
        // the one spent at the previous step has aged out of its window.
        enabled = await userServiceClientConnect.confirmRecoveryCodes(
          create(ConfirmRecoveryCodesRequestSchema, {
            name: currentUser.name,
            credential: buildCredentialProof("factor", confirmProofValue),
            pendingVersion: enrollment.pendingVersion,
          }),
          credentialProofCallOptions()
        );
      } else {
        // First-time enrollment: nothing is live until here. The fresh code
        // proves the device once more, and the credential proves the account
        // — the password held from the previous step, or the emailed code.
        enabled = await userServiceClientConnect.confirmRecoveryCodes(
          create(ConfirmRecoveryCodesRequestSchema, {
            name: currentUser.name,
            otpCode: freshOtp,
            credential: buildCredentialProof(
              proofMode,
              proofMode === "email" ? confirmProofValue : proofValue
            ),
            pendingVersion: enrollment.pendingVersion,
          }),
          credentialProofCallOptions()
        );
      }
    } catch (error) {
      notifyError(error);
      setSubmitting(false);
      return;
    }
    // Adopt the response rather than refetching. The navigation below runs
    // through the router guard, which sends an account without a live factor
    // back here; a refetch that failed would leave the store saying exactly
    // that and mint a second enrollment for a factor already enabled.
    setCurrentUser(enabled);
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
  }, [
    enrollment,
    submitting,
    rotation,
    currentUser.name,
    proofMode,
    proofValue,
    confirmProofValue,
    confirmOtpCodes,
    setCurrentUser,
    notifyError,
    t,
  ]);

  // Rotation proves the live factor here; a first-time enrollment re-proves
  // the new device with a fresh code, plus the emailed code when that is the
  // only proof the account has.
  const confirmNeedsEmailProof = !rotation && proofMode === "email";
  const confirmStepReady = rotation
    ? confirmProofValue.trim().length > 0
    : confirmOtpCodes.filter(Boolean).length === DIGITS &&
      (!confirmNeedsEmailProof || confirmProofValue.trim().length > 0);
  const allowNext =
    currentStep === SETUP_AUTH_APP_STEP
      ? otpCodes.filter(Boolean).length === DIGITS &&
        !isExpired &&
        (!needProofAtEnable || proofValue.trim().length > 0)
      : recoveryCodesDownloaded && confirmStepReady;

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
                <Button size="sm" onClick={handleRegenerateSecret}>
                  {t("two-factor.setup-steps.setup-auth-app.regenerate")}
                </Button>
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
          <div className="w-full flex flex-col justify-center items-center pb-4">
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
          {needProofAtEnable && (
            <div className="w-full max-w-sm mx-auto">
              <CredentialProofInput
                value={proofValue}
                onChange={setProofValue}
              />
            </div>
          )}
        </div>
      )}

      {currentStep === DOWNLOAD_RECOVERY_CODES_STEP && (
        <div className="w-full max-w-2xl mx-auto flex flex-col gap-y-6">
          <RecoveryCodesView
            recoveryCodes={enrollment ? [...enrollment.recoveryCodes] : []}
            onDownload={() => setRecoveryCodesDownloaded(true)}
          />
          {rotation ? (
            <div className="w-full max-w-sm mx-auto">
              <CredentialProofInput
                value={confirmProofValue}
                onChange={setConfirmProofValue}
              />
            </div>
          ) : (
            <div className="w-full flex flex-col justify-center items-center gap-y-2">
              <span className="text-sm font-medium">
                {t("two-factor.setup-steps.confirm-code")}
              </span>
              <OtpInput
                value={confirmOtpCodes}
                onChange={setConfirmOtpCodes}
                length={DIGITS}
              />
            </div>
          )}
          {confirmNeedsEmailProof && (
            <div className="w-full max-w-sm mx-auto">
              <CredentialProofInput
                value={confirmProofValue}
                onChange={setConfirmProofValue}
              />
            </div>
          )}
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
            <Button
              disabled={!allowNext || submitting}
              onClick={tryFinishSetup}
            >
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
