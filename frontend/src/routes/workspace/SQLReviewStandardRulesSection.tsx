import { create } from "@bufbuild/protobuf";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { StandardRuleSwitches } from "@/components/sql-review/StandardRuleSwitches";
import {
  reviewRulesOfPolicy,
  sameStandardRules,
} from "@/components/sql-review/standardRules";
import { useWorkspaceResourceName } from "@/hooks/useAppState";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import {
  PolicyResourceType,
  PolicyType,
  ReviewRulePolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import type { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

export interface WorkspaceStandardRules {
  // Undefined until the workspace policy loads.
  rules: ReviewRuleType[] | undefined;
  setRules: (rules: ReviewRuleType[]) => void;
  isDirty: boolean;
  saving: boolean;
  revert: () => void;
  save: () => Promise<void>;
}

// The page owns the draft so its sticky footer can sit at the end of the page,
// below the SQL review policy list.
export function useWorkspaceStandardRules(
  enabled: boolean
): WorkspaceStandardRules {
  const { t } = useTranslation();
  const workspace = useWorkspaceResourceName();
  const policy = useAppStore((state) =>
    state.getPolicyByParentAndType({
      parentPath: workspace,
      policyType: PolicyType.REVIEW_RULE,
    })
  );
  useEffect(() => {
    if (!enabled || !workspace) return;
    void useAppStore.getState().getOrFetchPolicyByParentAndType({
      parentPath: workspace,
      policyType: PolicyType.REVIEW_RULE,
    });
  }, [enabled, workspace]);

  const stored = reviewRulesOfPolicy(policy);
  const [draft, setDraft] = useState<ReviewRuleType[]>();
  const [saving, setSaving] = useState(false);
  const isDirty =
    draft !== undefined &&
    stored !== undefined &&
    !sameStandardRules(draft, stored);

  const save = async () => {
    if (!draft) return;
    setSaving(true);
    try {
      await useAppStore.getState().upsertPolicy({
        parentPath: workspace,
        policy: {
          type: PolicyType.REVIEW_RULE,
          resourceType: PolicyResourceType.WORKSPACE,
          enforce: true,
          policy: {
            case: "reviewRulePolicy",
            value: create(ReviewRulePolicySchema, { rules: draft }),
          },
        },
      });
      setDraft(undefined);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch {
      // The response middleware reports the failure; the draft stays for a
      // retry.
    } finally {
      setSaving(false);
    }
  };

  return {
    rules: draft ?? stored,
    setRules: setDraft,
    isDirty,
    saving,
    revert: () => setDraft(undefined),
    save,
  };
}

export function SQLReviewStandardRulesSection({
  standardRules,
}: {
  standardRules: WorkspaceStandardRules;
}) {
  const { t } = useTranslation();
  // The workspace has no policy row until the first save, which creates it.
  const canUpdate =
    hasWorkspacePermissionV2("bb.policies.update") &&
    hasWorkspacePermissionV2("bb.policies.create");

  return (
    <section className="flex flex-col gap-y-6 pt-8">
      <div className="flex flex-col gap-y-2">
        <h1 className="text-2xl font-bold text-main">
          {t("sql-review.standard-rules.self")}
        </h1>
        <p className="text-sm text-control-light">
          {t("sql-review.standard-rules.description")}
        </p>
      </div>
      {standardRules.rules && (
        <StandardRuleSwitches
          rules={standardRules.rules}
          onChange={standardRules.setRules}
          disabled={!canUpdate || standardRules.saving}
        />
      )}
    </section>
  );
}
