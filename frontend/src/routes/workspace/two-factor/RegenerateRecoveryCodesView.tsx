import { create } from "@bufbuild/protobuf";
import type { ConnectError } from "@connectrpc/connect";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { userServiceClientConnect } from "@/api";
import {
  buildCredentialProof,
  CredentialProofInput,
  credentialProofCallOptions,
} from "@/components/CredentialProofInput";
import { Button } from "@/components/ui/button";
import { useCurrentUser } from "@/hooks/useAppState";
import { pushNotification } from "@/stores";
import type { RegenerateRecoveryCodesResponse } from "@/types/proto-es/v1/user_service_pb";
import {
  ConfirmRecoveryCodesRequestSchema,
  RegenerateRecoveryCodesRequestSchema,
} from "@/types/proto-es/v1/user_service_pb";
import { RecoveryCodesView } from "./RecoveryCodesView";

interface RegenerateRecoveryCodesViewProps {
  onClose: () => void;
}

// Mint-then-confirm: RegenerateRecoveryCodes returns a pending set (the old
// codes keep working), and ConfirmRecoveryCodes promotes exactly that set —
// proven with the live factor, since promotion is the moment the old codes
// stop working (docs/design/reauthenticate-credential-changes.md).
export function RegenerateRecoveryCodesView({
  onClose,
}: RegenerateRecoveryCodesViewProps) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const [pending, setPending] = useState<
    RegenerateRecoveryCodesResponse | undefined
  >(undefined);
  const [recoveryCodesDownloaded, setRecoveryCodesDownloaded] = useState(false);
  const [proofValue, setProofValue] = useState("");
  const [submitting, setSubmitting] = useState(false);

  // The codes exist only in the minting response, so this view fetches its own
  // rather than reading them off the user. The live set keeps working until
  // they are confirmed below.
  useEffect(() => {
    userServiceClientConnect
      .regenerateRecoveryCodes(
        create(RegenerateRecoveryCodesRequestSchema, { name: currentUser.name })
      )
      .then(setPending)
      .catch((error) => {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: (error as ConnectError).message,
        });
      });
  }, [currentUser.name]);

  const confirmRecoveryCodes = useCallback(async () => {
    if (!pending || submitting) return;
    setSubmitting(true);
    // Confirm the set this view minted and displayed, never whatever is
    // pending now: an enrollment started in another tab would otherwise be
    // promoted here, swapping the factor along with it.
    try {
      await userServiceClientConnect.confirmRecoveryCodes(
        create(ConfirmRecoveryCodesRequestSchema, {
          name: currentUser.name,
          credential: buildCredentialProof("factor", proofValue),
          pendingVersion: pending.pendingVersion,
        }),
        credentialProofCallOptions()
      );
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: (error as ConnectError).message,
      });
      setSubmitting(false);
      return;
    }
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("two-factor.messages.recovery-codes-regenerated"),
    });
    onClose();
  }, [pending, submitting, currentUser.name, proofValue, onClose, t]);

  return (
    <>
      <RecoveryCodesView
        recoveryCodes={pending ? [...pending.recoveryCodes] : []}
        onDownload={() => setRecoveryCodesDownloaded(true)}
      />
      <div className="max-w-sm mb-4">
        <CredentialProofInput value={proofValue} onChange={setProofValue} />
      </div>
      <div className="flex flex-row justify-between items-center mb-8">
        <Button appearance="outline" onClick={onClose}>
          {t("common.cancel")}
        </Button>
        <Button
          disabled={
            !recoveryCodesDownloaded ||
            !proofValue.trim() ||
            !pending ||
            submitting
          }
          onClick={confirmRecoveryCodes}
        >
          {t("two-factor.setup-steps.recovery-codes-saved")}
        </Button>
      </div>
    </>
  );
}
