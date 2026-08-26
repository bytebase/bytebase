import { create } from "@bufbuild/protobuf";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { userServiceClientConnect } from "@/api";
import { Button } from "@/components/ui/button";
import { useCurrentUser } from "@/hooks/useAppState";
import { pushNotification } from "@/stores";
import {
  ConfirmRecoveryCodesRequestSchema,
  RegenerateRecoveryCodesRequestSchema,
} from "@/types/proto-es/v1/user_service_pb";
import { RecoveryCodesView } from "./RecoveryCodesView";

interface RegenerateRecoveryCodesViewProps {
  onClose: () => void;
}

export function RegenerateRecoveryCodesView({
  onClose,
}: RegenerateRecoveryCodesViewProps) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);
  const [pendingVersion, setPendingVersion] = useState<Timestamp | undefined>();
  const [recoveryCodesDownloaded, setRecoveryCodesDownloaded] = useState(false);

  // The codes exist only in the minting response, so this view fetches its own
  // rather than reading them off the user. The live set keeps working until
  // they are confirmed below.
  useEffect(() => {
    userServiceClientConnect
      .regenerateRecoveryCodes(
        create(RegenerateRecoveryCodesRequestSchema, { name: currentUser.name })
      )
      .then((response) => {
        setRecoveryCodes([...response.recoveryCodes]);
        setPendingVersion(response.pendingVersion);
      });
  }, [currentUser.name]);

  const confirmRecoveryCodes = useCallback(async () => {
    // Confirm the set this view minted and displayed, never whatever is
    // pending now: an enrollment started in another tab would otherwise be
    // promoted here, swapping the factor along with it.
    await userServiceClientConnect.confirmRecoveryCodes(
      create(ConfirmRecoveryCodesRequestSchema, {
        name: currentUser.name,
        pendingVersion,
      })
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("two-factor.messages.recovery-codes-regenerated"),
    });
    onClose();
  }, [currentUser.name, onClose, pendingVersion, t]);

  return (
    <>
      <RecoveryCodesView
        recoveryCodes={recoveryCodes}
        onDownload={() => setRecoveryCodesDownloaded(true)}
      />
      <div className="flex flex-row justify-between items-center mb-8">
        <Button appearance="outline" onClick={onClose}>
          {t("common.cancel")}
        </Button>
        <Button
          disabled={!recoveryCodesDownloaded}
          onClick={confirmRecoveryCodes}
        >
          {t("two-factor.setup-steps.recovery-codes-saved")}
        </Button>
      </div>
    </>
  );
}
