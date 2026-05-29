import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";
import { RecoveryCodesView } from "./RecoveryCodesView";

interface RegenerateRecoveryCodesViewProps {
  recoveryCodes: string[];
  onClose: () => void;
}

export function RegenerateRecoveryCodesView({
  recoveryCodes,
  onClose,
}: RegenerateRecoveryCodesViewProps) {
  const { t } = useTranslation();
  const updateUser = useAppStore((state) => state.updateUser);
  const currentUser = useCurrentUser();
  const [recoveryCodesDownloaded, setRecoveryCodesDownloaded] = useState(false);

  useEffect(() => {
    updateUser(
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
  }, []);

  const regenerateRecoveryCodes = useCallback(async () => {
    await updateUser(
      create(UpdateUserRequestSchema, {
        user: {
          name: currentUser.name,
        },
        updateMask: create(FieldMaskSchema, {
          paths: [],
        }),
        regenerateRecoveryCodes: true,
      })
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("two-factor.messages.recovery-codes-regenerated"),
    });
    onClose();
  }, [currentUser.name, onClose, t, updateUser]);

  return (
    <>
      <RecoveryCodesView
        recoveryCodes={recoveryCodes}
        onDownload={() => setRecoveryCodesDownloaded(true)}
      />
      <div className="flex flex-row justify-between items-center mb-8">
        <Button variant="outline" onClick={onClose}>
          {t("common.cancel")}
        </Button>
        <Button
          disabled={!recoveryCodesDownloaded}
          onClick={regenerateRecoveryCodes}
        >
          {t("two-factor.setup-steps.recovery-codes-saved")}
        </Button>
      </div>
    </>
  );
}
