import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/app/router/handles";
import { FeatureAttention } from "@/components/FeatureAttention";
import { RouterLink } from "@/components/RouterLink";
import {
  type ReviewAIPolicyDraft,
  ReviewAIPolicyField,
  useReviewAIPolicyDraft,
} from "@/components/sql-review/ReviewAIPolicyField";
import { SwitchRow } from "@/components/sql-review/SwitchRow";
import { Alert } from "@/components/ui/alert";
import { FormFieldGroup, FormSection } from "@/components/ui/form";
import {
  usePlanFeature,
  useServerState,
  useWorkspaceResourceName,
} from "@/hooks/useAppState";
import {
  AI_ASSISTANT_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
} from "@/lib/productIntro";
import { useAppStore } from "@/stores/app";
import { PolicyResourceType } from "@/types/proto-es/v1/org_policy_service_pb";
import {
  AISettingSchema,
  Setting_SettingName,
  SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

export interface WorkspaceAIReview {
  // Whether the review is switched on, or the unsaved choice.
  enabled: boolean;
  // Whether AI is on for the workspace. While it is off no AI review runs,
  // whatever the switch says.
  aiEnabled: boolean;
  // The switch is a field of the AI setting.
  canEdit: boolean;
  setEnabled: (enabled: boolean) => void;
  isDirty: boolean;
  saving: boolean;
  revert: () => void;
  // Resolves whether the switch saved. The page reports the update; a failure
  // is reported by the response middleware and keeps the draft.
  save: () => Promise<boolean>;
}

// The page owns the draft so the switch saves with the standard rules, from
// the one sticky footer.
export function useWorkspaceAIReview(): WorkspaceAIReview {
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

  const save = async () => {
    if (draft === undefined) return true;
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
      return true;
    } catch {
      return false;
    } finally {
      setSaving(false);
    }
  };

  return {
    enabled: draft ?? stored,
    aiEnabled,
    canEdit: aiEnabled && hasWorkspacePermissionV2("bb.settings.set"),
    setEnabled: setDraft,
    isDirty,
    saving,
    revert: () => setDraft(undefined),
    save,
  };
}

// The workspace policy reaches the review of every project, together with the
// project's own.
export function useWorkspaceReviewAIPolicy(): ReviewAIPolicyDraft {
  const workspace = useWorkspaceResourceName();
  return useReviewAIPolicyDraft({
    parentPath: workspace,
    resourceType: PolicyResourceType.WORKSPACE,
    can: hasWorkspacePermissionV2,
    enabled: true,
  });
}

// `standardRules` says whether the standard rules review beside the AI. Beside
// V1's review policies they do not, the AI review's findings are comments, and
// its policy is named apart from the SQL review policies.
export function SQLReviewAIReviewSection({
  aiReview,
  policy,
  standardRules = true,
}: {
  aiReview: WorkspaceAIReview;
  policy: ReviewAIPolicyDraft;
  standardRules?: boolean;
}) {
  const { t } = useTranslation();
  const { isSaaSMode } = useServerState();
  // The AI review is an Enterprise feature: without it the review is off,
  // whatever the switch says.
  const licensed = usePlanFeature(PlanFeature.FEATURE_AI_SQL_REVIEW);
  const reviewOn = licensed && aiReview.aiEnabled && aiReview.enabled;
  // AI is turned on in the general settings, whose AI section shows its
  // controls only to a user who can open that page and read the AI setting.
  const canTurnOnAI =
    !isSaaSMode &&
    hasWorkspacePermissionV2("bb.settings.set") &&
    hasWorkspacePermissionV2("bb.settings.get") &&
    hasWorkspacePermissionV2("bb.settings.getWorkspaceProfile");

  return (
    <FormSection
      layout="single-column"
      padding="none"
      title={t("sql-review.ai-review.self")}
      description={
        standardRules
          ? t("sql-review.ai-review.description")
          : t("sql-review.ai-review.comments-description")
      }
    >
      {!licensed && (
        <FeatureAttention feature={PlanFeature.FEATURE_AI_SQL_REVIEW} compact />
      )}
      {licensed && !aiReview.aiEnabled && (
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
                      query: {
                        [PRODUCT_INTRO_QUERY_KEY]: AI_ASSISTANT_PRODUCT_INTRO,
                      },
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
      <FormFieldGroup>
        <SwitchRow
          className="rounded-sm border border-block-border px-4 py-3"
          title={t("sql-review.ai-review.enable.self")}
          description={t("sql-review.ai-review.enable.description")}
          checked={reviewOn}
          onCheckedChange={aiReview.setEnabled}
          disabled={aiReview.saving}
          readOnly={!licensed || !aiReview.canEdit}
        />
        {reviewOn && (
          <ReviewAIPolicyField
            policy={policy}
            placeholder={t("sql-review.ai-review.policy.placeholder")}
            standardRules={standardRules}
            title={
              standardRules
                ? undefined
                : t("sql-review.ai-review.policy.ai-review-policy")
            }
          />
        )}
      </FormFieldGroup>
    </FormSection>
  );
}
