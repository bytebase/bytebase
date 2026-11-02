import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { SecretInput } from "@/components/SecretInput";
import { Alert } from "@/components/ui/alert";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { FormField, FormFieldGroup, FormLabel } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import {
  ExportSettingSchema,
  Setting_SettingName,
  SettingValueSchema,
  type SharePointSetting,
  SharePointSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";

const MASK_PREFIX = "value.export.sharepoint";

export interface SharePointDraft {
  tenantId: string;
  clientId: string;
  // Undefined keeps the stored secret.
  clientSecret: string | undefined;
}

export function buildSharePointUpdate(
  stored: SharePointSetting,
  draft: SharePointDraft
): { setting: SharePointSetting; paths: string[] } {
  const tenantId = draft.tenantId.trim();
  const clientId = draft.clientId.trim();
  const paths: string[] = [];
  if (tenantId !== stored.tenantId) paths.push(`${MASK_PREFIX}.tenant_id`);
  if (clientId !== stored.clientId) paths.push(`${MASK_PREFIX}.client_id`);
  if (draft.clientSecret) paths.push(`${MASK_PREFIX}.client_secret`);
  return {
    setting: create(SharePointSettingSchema, {
      tenantId,
      clientId,
      clientSecret: draft.clientSecret ?? "",
    }),
    paths,
  };
}

async function saveSharePointSetting(
  setting: SharePointSetting,
  paths: string[]
) {
  await useAppStore.getState().upsertSetting({
    name: Setting_SettingName.EXPORT,
    value: create(SettingValueSchema, {
      value: {
        case: "export",
        value: create(ExportSettingSchema, { sharepoint: setting }),
      },
    }),
    updateMask: create(FieldMaskSchema, { paths }),
  });
}

interface Props {
  open: boolean;
  // Changes on every open so the form starts from the stored setting.
  sessionKey: number;
  stored: SharePointSetting;
  allowEdit: boolean;
  onClose: () => void;
}

export function SharePointSheet({
  open,
  sessionKey,
  stored,
  allowEdit,
  onClose,
}: Props) {
  // Keep the last open setting so the form stays stable through the close
  // animation after a save or removal updates the store.
  const storedRef = useRef(stored);
  if (open) {
    storedRef.current = stored;
  }
  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        <SharePointForm
          key={sessionKey}
          stored={storedRef.current}
          allowEdit={allowEdit}
          onClose={onClose}
        />
      </SheetContent>
    </Sheet>
  );
}

function SharePointForm({
  stored,
  allowEdit,
  onClose,
}: {
  stored: SharePointSetting;
  allowEdit: boolean;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const configured = stored.tenantId !== "";
  const [tenantId, setTenantId] = useState(stored.tenantId);
  const [clientId, setClientId] = useState(stored.clientId);
  const [clientSecret, setClientSecret] = useState<string | undefined>(
    undefined
  );
  const [saving, setSaving] = useState(false);
  const [confirmingRemove, setConfirmingRemove] = useState(false);

  const update = buildSharePointUpdate(stored, {
    tenantId,
    clientId,
    clientSecret,
  });
  const isDirty = update.paths.length > 0 || clientSecret === "";
  const isValid =
    update.setting.tenantId !== "" &&
    update.setting.clientId !== "" &&
    (stored.clientSecretSet ? clientSecret !== "" : !!clientSecret);

  const run = async (action: () => Promise<void>, title: string) => {
    setSaving(true);
    try {
      await action();
      pushNotification({ module: "bytebase", style: "SUCCESS", title });
      onClose();
    } catch {
      setSaving(false);
    }
  };

  const save = () =>
    run(
      () => saveSharePointSetting(update.setting, update.paths),
      t("common.updated")
    );

  // Clearing both IDs removes the application, secret included.
  const remove = () =>
    run(
      () =>
        saveSharePointSetting(create(SharePointSettingSchema, {}), [
          `${MASK_PREFIX}.tenant_id`,
          `${MASK_PREFIX}.client_id`,
        ]),
      t("common.removed")
    );

  return (
    <>
      <SheetHeader>
        <SheetTitle>{t("export-data.destination.sharepoint")}</SheetTitle>
        <SheetDescription>{t("settings.sharepoint.caption")}</SheetDescription>
      </SheetHeader>
      <SheetBody className="gap-y-6">
        <Alert
          variant="info"
          title={t("settings.sharepoint.setup.title")}
          description={
            <ol className="list-decimal pl-4">
              <li>{t("settings.sharepoint.setup.register")}</li>
              <li>{t("settings.sharepoint.setup.grant")}</li>
              <li>{t("settings.sharepoint.setup.project")}</li>
            </ol>
          }
        />
        <FormFieldGroup>
          <FormField
            title={
              <FormLabel htmlFor="sharepoint-tenant-id">
                {t("settings.sharepoint.tenant-id")}
              </FormLabel>
            }
          >
            <Input
              id="sharepoint-tenant-id"
              value={tenantId}
              disabled={!allowEdit}
              required
              onChange={(event) => setTenantId(event.target.value)}
            />
          </FormField>
          <FormField
            title={
              <FormLabel htmlFor="sharepoint-client-id">
                {t("settings.sharepoint.client-id")}
              </FormLabel>
            }
          >
            <Input
              id="sharepoint-client-id"
              value={clientId}
              disabled={!allowEdit}
              required
              onChange={(event) => setClientId(event.target.value)}
            />
          </FormField>
          <FormField
            title={
              <FormLabel htmlFor="sharepoint-client-secret">
                {t("settings.sharepoint.client-secret")}
              </FormLabel>
            }
            description={
              stored.clientSecretSet
                ? t("settings.sharepoint.client-secret-saved")
                : t("settings.sharepoint.client-secret-description")
            }
          >
            <SecretInput
              id="sharepoint-client-secret"
              value={clientSecret}
              isCreating={!stored.clientSecretSet}
              allowEmpty={false}
              disabled={!allowEdit}
              placeholder={t("common.sensitive-placeholder")}
              onValueChange={setClientSecret}
            />
          </FormField>
        </FormFieldGroup>
      </SheetBody>
      <SheetFooter>
        {configured && allowEdit && (
          <Button
            variant="destructive"
            appearance="outline"
            className="mr-auto"
            disabled={saving}
            onClick={() => setConfirmingRemove(true)}
          >
            {t("common.remove")}
          </Button>
        )}
        <Button appearance="outline" disabled={saving} onClick={onClose}>
          {t("common.cancel")}
        </Button>
        {allowEdit && (
          <Button
            disabled={!isValid || !isDirty || saving}
            onClick={() => void save()}
          >
            {t("common.save")}
          </Button>
        )}
      </SheetFooter>
      <AlertDialog
        open={confirmingRemove}
        onOpenChange={(next) => !next && setConfirmingRemove(false)}
      >
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("settings.sharepoint.remove.title")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("settings.sharepoint.remove.description")}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button
              appearance="outline"
              disabled={saving}
              onClick={() => setConfirmingRemove(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              disabled={saving}
              onClick={() => {
                setConfirmingRemove(false);
                void remove();
              }}
            >
              {t("common.remove")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
