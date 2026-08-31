import { create } from "@bufbuild/protobuf";
import { Code, createContextValues } from "@connectrpc/connect";
import {
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { userServiceClientConnect } from "@/api";
import { ignoredCodesContextKey } from "@/api/context-key";
import { Button } from "@/components/ui/button";
import { FormControlRow, FormField, FormLabel } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { useCurrentUser } from "@/hooks/useAppState";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import type { CredentialProof } from "@/types/proto-es/v1/user_service_pb";
import {
  CredentialProofSchema,
  RequestReauthCodeRequestSchema,
} from "@/types/proto-es/v1/user_service_pb";

// How the signed-in account can prove it still holds a credential
// (CredentialProof in user_service.proto). The server enforces the same rule;
// this only picks which input to render:
// - a live MFA factor is proven with the factor (an OTP or a recovery code),
// - else on Cloud, a one-time emailed code (Cloud accounts never have a
//   caller-chosen password, and only Cloud can send email),
// - else the password — self-hosted has no email capability, so an account
//   that never had a password needs an admin reset before it can prove
//   anything (the server says so).
export type CredentialProofMode = "factor" | "password" | "email";

export function useCredentialProofMode(): CredentialProofMode {
  const currentUser = useCurrentUser();
  const saas = useAppStore((state) => state.serverInfo?.saas ?? false);
  if (currentUser.mfaEnabled) return "factor";
  if (saas) return "email";
  return "password";
}

// A recovery code is longer than a 6-digit OTP and never purely numeric.
function factorProofOf(value: string): CredentialProof {
  if (/^\d{6}$/.test(value)) {
    return create(CredentialProofSchema, {
      proof: { case: "otpCode", value },
    });
  }
  return create(CredentialProofSchema, {
    proof: { case: "recoveryCode", value },
  });
}

interface CredentialProofInputProps {
  value: string;
  onChange: (value: string) => void;
}

// credentialProofCallOptions marks a call that carries a CredentialProof: a
// wrong proof answers Unauthenticated, which must reach the caller as a plain
// error — not trigger the interceptor's refresh-and-retry, which would
// re-submit the same wrong proof (burning a second lockout slot) and then
// declare the whole session expired.
export function credentialProofCallOptions() {
  return {
    contextValues: createContextValues().set(ignoredCodesContextKey, [
      Code.Unauthenticated,
    ]),
  };
}

// isCredentialProofReady reports whether the input holds enough to submit.
// A password is whatever the user typed, whitespace included — login compares
// it verbatim, so an account whose password is spaces must still be able to
// change it. The codes are transcribed, so their surrounding whitespace is a
// paste artifact rather than content.
export function isCredentialProofReady(
  mode: CredentialProofMode,
  value: string
): boolean {
  return mode === "password" ? value.length > 0 : value.trim().length > 0;
}

// buildCredentialProof turns the raw input value into the proof message for
// the mode the account is in, or undefined while the input is empty.
export function buildCredentialProof(
  mode: CredentialProofMode,
  value: string
): CredentialProof | undefined {
  if (!isCredentialProofReady(mode, value)) return undefined;
  const trimmed = value.trim();
  switch (mode) {
    case "factor":
      return factorProofOf(trimmed);
    case "password":
      return create(CredentialProofSchema, {
        proof: { case: "currentPassword", value },
      });
    case "email":
      return create(CredentialProofSchema, {
        proof: { case: "emailCode", value: trimmed },
      });
  }
}

// CredentialProofInput renders the one input the account can answer with. The
// email mode carries its own "email me a code" sender with a resend cooldown
// matching the server's.
export function CredentialProofInput({
  value,
  onChange,
}: Readonly<CredentialProofInputProps>) {
  const { t } = useTranslation();
  // Only one mode's Input renders at a time, so a single id serves them all.
  const inputId = useId();
  const mode = useCredentialProofMode();
  const currentUser = useCurrentUser();
  const [sending, setSending] = useState(false);
  const [resendCountdown, setResendCountdown] = useState(0);
  const countdownRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => {
      if (countdownRef.current) clearInterval(countdownRef.current);
    };
  }, []);

  const requestCode = useCallback(async () => {
    setSending(true);
    try {
      await userServiceClientConnect.requestReauthCode(
        create(RequestReauthCodeRequestSchema, { name: currentUser.name })
      );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("credential-proof.code-sent"),
      });
      setResendCountdown(60);
      countdownRef.current = setInterval(() => {
        setResendCountdown((remaining) => {
          if (remaining <= 1 && countdownRef.current) {
            clearInterval(countdownRef.current);
            countdownRef.current = null;
          }
          return Math.max(0, remaining - 1);
        });
      }, 1000);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: (error as Error).message,
      });
    } finally {
      setSending(false);
    }
  }, [currentUser.name, t]);

  const label = useMemo(() => {
    switch (mode) {
      case "factor":
        return t("credential-proof.factor-label");
      case "password":
        return t("credential-proof.password-label");
      case "email":
        return t("credential-proof.email-label");
    }
  }, [mode, t]);

  return (
    <FormField>
      <FormLabel htmlFor={inputId}>
        {label}
        <span className="text-error">*</span>
      </FormLabel>
      {mode === "factor" && (
        <Input
          id={inputId}
          value={value}
          placeholder={t("credential-proof.factor-placeholder")}
          autoComplete="one-time-code"
          onChange={(e) => onChange(e.target.value)}
        />
      )}
      {mode === "password" && (
        <Input
          id={inputId}
          type="password"
          value={value}
          autoComplete="current-password"
          onChange={(e) => onChange(e.target.value)}
        />
      )}
      {mode === "email" && (
        <FormControlRow>
          <Input
            id={inputId}
            value={value}
            placeholder={t("credential-proof.email-placeholder")}
            autoComplete="one-time-code"
            onChange={(e) => onChange(e.target.value)}
          />
          <Button
            appearance="outline"
            disabled={sending || resendCountdown > 0}
            onClick={requestCode}
          >
            {resendCountdown > 0
              ? t("credential-proof.resend-in", { seconds: resendCountdown })
              : t("credential-proof.email-me-a-code")}
          </Button>
        </FormControlRow>
      )}
    </FormField>
  );
}
