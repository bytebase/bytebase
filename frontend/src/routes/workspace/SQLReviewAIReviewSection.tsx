import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/app/router/handles";
import { RouterLink } from "@/components/RouterLink";
import { SwitchRow } from "@/components/sql-review/SwitchRow";
import { Alert } from "@/components/ui/alert";
import { useServerState } from "@/hooks/useAppState";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import {
  AISettingSchema,
  Setting_SettingName,
  SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

export interface WorkspaceAIReview {
  // Whether the review is switched on, or the unsaved choice.
  enabled: boolean;
  // Whether AI is on for the workspace. While it is off no review runs,
  // whatever the switch says.
  aiEnabled: boolean;
  // The switch is a field of the AI setting, which SaaS keeps read only.
  canEdit: boolean;
  setEnabled: (enabled: boolean) => void;
  isDirty: boolean;
  saving: boolean;
  revert: () => void;
  // Silent when another part of the page saves in the same click and reports
  // the update itself.
  save: (options?: { silent?: boolean }) => Promise<void>;
}

// The page owns the draft so the switch saves with the standard rules, from
// the one sticky footer.
export function useWorkspaceAIReview(): WorkspaceAIReview {
  const { t } = useTranslation();
  const { isSaaSMode } = useServerState();
  // The server info carries both flags, so a user who cannot read the AI
  // setting still sees whether the review is on.
  const stored = useAppStore(
    (state) => state.serverInfo?.aiSqlReviewEnabled === true
  );
  const aiEnabled = useAppStore(
    (state) => state.serverInfo?.aiEnabled === true
  );
  const [draft, setDraft] = useState<boolean>();
  const [saving, setSaving] = useState(false);
  const isDirty = draft !== undefined && draft !== stored;

  const save = async (options?: { silent?: boolean }) => {
    if (draft === undefined) return;
    setSaving(true);
    try {
      await useAppStore.getState().upsertSetting({
        name: Setting_SettingName.AI,
        value: create(SettingValueSchema, {
          value: {
            case: "ai",
            value: create(AISettingSchema, { sqlReviewEnabled: draft }),
          },
        }),
        // The mask leaves the provider, the key, and the model as stored.
        updateMask: create(FieldMaskSchema, {
          paths: ["value.ai.sql_review_enabled"],
        }),
      });
      setDraft(undefined);
      if (!options?.silent) {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
      }
    } catch {
      // The response middleware reports the failure; the draft stays for a
      // retry.
    } finally {
      setSaving(false);
    }
  };

  return {
    enabled: draft ?? stored,
    aiEnabled,
    canEdit: hasWorkspacePermissionV2("bb.settings.set") && !isSaaSMode,
    setEnabled: setDraft,
    isDirty,
    saving,
    revert: () => setDraft(undefined),
    save,
  };
}

export function SQLReviewAIReviewSection({
  aiReview,
}: {
  aiReview: WorkspaceAIReview;
}) {
  const { t } = useTranslation();
  // AI is turned on in the general settings, whose AI section shows its
  // controls only to a user who can open that page and read the AI setting.
  const canTurnOnAI =
    aiReview.canEdit &&
    hasWorkspacePermissionV2("bb.settings.get") &&
    hasWorkspacePermissionV2("bb.settings.getWorkspaceProfile");

  return (
    <section className="flex flex-col gap-y-6 pt-8">
      <div className="flex flex-col gap-y-2">
        <h1 className="text-2xl font-bold text-main">
          {t("sql-review.ai-review.self")}
        </h1>
        <p className="text-sm text-control-light">
          {t("sql-review.ai-review.description")}
        </p>
      </div>
      {!aiReview.aiEnabled && (
        <Alert
          variant="warning"
          description={
            <>
              {t("sql-review.ai-review.ai-off")}
              {canTurnOnAI && (
                <>
                  {" "}
                  <RouterLink
                    to={{
                      name: SETTING_ROUTE_WORKSPACE_GENERAL,
                      hash: "#ai-assistant",
                    }}
                    className="text-accent hover:underline"
                  >
                    {t("sql-review.ai-review.turn-on-ai")}
                  </RouterLink>
                </>
              )}
            </>
          }
        />
      )}
      <SwitchRow
        className="rounded-sm border border-block-border px-4 py-3"
        title={t("sql-review.ai-review.enable.self")}
        description={t("sql-review.ai-review.enable.description")}
        checked={aiReview.enabled}
        onCheckedChange={aiReview.setEnabled}
        disabled={aiReview.saving}
        readOnly={!aiReview.canEdit}
      />
    </section>
  );
}
