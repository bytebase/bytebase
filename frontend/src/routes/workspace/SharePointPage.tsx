import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FeatureAttention } from "@/components/FeatureAttention";
import { SecretInput } from "@/components/SecretInput";
import { Button } from "@/components/ui/button";
import { FormField, FormFieldGroup, FormLabel } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { StickyActionFooter } from "@/components/ui/sticky-action-footer";
import {
  WorkspacePageInfo,
  WorkspacePageLayout,
} from "@/components/WorkspacePageLayout";
import { usePlanFeature } from "@/hooks/useAppState";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import {
  Setting_SettingName,
  SettingValueSchema,
  type SharePointSetting,
  SharePointSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

const MASK_PREFIX = "value.sharepoint";

function getStoredSharePointSetting(): SharePointSetting {
  const setting = useAppStore
    .getState()
    .getSettingByName(Setting_SettingName.SHAREPOINT);
  if (setting?.value?.value?.case !== "sharepoint") {
    return create(SharePointSettingSchema, {});
  }
  return setting.value.value.value;
}

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

export function SharePointPage() {
  const { t } = useTranslation();
  const settingsByName = useAppStore((s) => s.settingsByName);
  const stored = useMemo(() => getStoredSharePointSetting(), [settingsByName]);
  const [initialized, setInitialized] = useState(false);
  const [savedCount, setSavedCount] = useState(0);

  useEffect(() => {
    void useAppStore
      .getState()
      .getOrFetchSettingByName(Setting_SettingName.SHAREPOINT, true)
      .finally(() => setInitialized(true));
  }, []);

  return (
    <WorkspacePageLayout>
      <WorkspacePageInfo description={t("settings.sharepoint.description")} />
      <FeatureAttention feature={PlanFeature.FEATURE_SHAREPOINT_EXPORT} />
      {initialized && (
        <SharePointForm
          key={savedCount}
          stored={stored}
          onSaved={() => setSavedCount((count) => count + 1)}
        />
      )}
    </WorkspacePageLayout>
  );
}

function SharePointForm({
  stored,
  onSaved,
}: {
  stored: SharePointSetting;
  onSaved: () => void;
}) {
  const { t } = useTranslation();
  const hasFeature = usePlanFeature(PlanFeature.FEATURE_SHAREPOINT_EXPORT);
  const allowEdit = hasWorkspacePermissionV2("bb.settings.set") && hasFeature;

  const [tenantId, setTenantId] = useState(stored.tenantId);
  const [clientId, setClientId] = useState(stored.clientId);
  const [clientSecret, setClientSecret] = useState<string | undefined>(
    undefined
  );
  const [secretResetKey, setSecretResetKey] = useState(0);
  const [saving, setSaving] = useState(false);

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

  useUnsavedChangesGuard(isDirty);

  const revert = () => {
    setTenantId(stored.tenantId);
    setClientId(stored.clientId);
    setClientSecret(undefined);
    setSecretResetKey((key) => key + 1);
  };

  const save = async () => {
    setSaving(true);
    try {
      await useAppStore.getState().upsertSetting({
        name: Setting_SettingName.SHAREPOINT,
        value: create(SettingValueSchema, {
          value: { case: "sharepoint", value: update.setting },
        }),
        updateMask: create(FieldMaskSchema, { paths: update.paths }),
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
      onSaved();
    } catch {
      setSaving(false);
    }
  };

  return (
    <>
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
            className="w-full sm:w-96"
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
            className="w-full sm:w-96"
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
            className="w-full sm:w-96"
            value={clientSecret}
            isCreating={!stored.clientSecretSet}
            allowEmpty={false}
            disabled={!allowEdit}
            resetKey={secretResetKey}
            placeholder={t("common.sensitive-placeholder")}
            onValueChange={setClientSecret}
          />
        </FormField>
      </FormFieldGroup>
      {isDirty && (
        <StickyActionFooter
          contentPadding={false}
          left={
            <Button appearance="outline" disabled={saving} onClick={revert}>
              {t("common.cancel")}
            </Button>
          }
          right={
            <Button
              disabled={!allowEdit || !isValid || saving}
              onClick={() => void save()}
            >
              {t("common.save")}
            </Button>
          }
        />
      )}
    </>
  );
}
