import { create } from "@bufbuild/protobuf";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { StandardRuleSwitches } from "@/components/sql-review/StandardRuleSwitches";
import {
  effectiveWorkspaceRules,
  sameStandardRules,
} from "@/components/sql-review/standardRules";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
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
  // The rules in force; undefined until read, or when the read failed.
  rules: ReviewRuleType[] | undefined;
  readFailed: boolean;
  reload: () => void;
  // Whether this user may save. Every workspace gets a review rule policy
  // row by default, so a save is an update.
  canEdit: boolean;
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
  const [attempt, setAttempt] = useState(0);
  const [readFailed, setReadFailed] = useState(false);
  const [draft, setDraft] = useState<ReviewRuleType[]>();
  useEffect(() => {
    if (!enabled || !workspace) return;
    let active = true;
    setDraft(undefined);
    setReadFailed(false);
    void useAppStore
      .getState()
      .fetchPolicyByParentAndType({
        parentPath: workspace,
        policyType: PolicyType.REVIEW_RULE,
        refresh: attempt > 0,
      })
      .then((policy) => {
        if (active && policy === undefined) setReadFailed(true);
      });
    return () => {
      active = false;
    };
  }, [enabled, workspace, attempt]);

  const stored = effectiveWorkspaceRules(policy);
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
    readFailed,
    reload: () => setAttempt((n) => n + 1),
    canEdit: hasWorkspacePermissionV2("bb.policies.update"),
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
  const workspace = useWorkspaceResourceName();

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
      {standardRules.readFailed && (
        <Alert
          variant="error"
          description={t("sql-review.standard-rules.load-failed")}
        >
          <Button
            className="mt-3"
            appearance="outline"
            size="sm"
            onClick={standardRules.reload}
          >
            {t("sql-review.standard-rules.retry")}
          </Button>
        </Alert>
      )}
      {standardRules.rules && (
        <StandardRuleSwitches
          // The rules remembered while SYNTAX is off belong to one workspace.
          key={workspace}
          rules={standardRules.rules}
          onChange={standardRules.setRules}
          disabled={!standardRules.canEdit || standardRules.saving}
        />
      )}
    </section>
  );
}
