import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { pushNotification, useCurrentUserV1, useUserStore } from "@/store";
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
  const userStore = useUserStore();
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const [recoveryCodesDownloaded, setRecoveryCodesDownloaded] = useState(false);

  useEffect(() => {
    userStore.updateUser(
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
    await userStore.updateUser(
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
  }, [currentUser.name, onClose, t, userStore]);

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
