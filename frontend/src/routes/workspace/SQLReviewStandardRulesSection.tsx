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
  type Policy,
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
  // Whether this user may save. The first save creates the workspace's
  // policy row and later saves update it.
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
  const canCreate = hasWorkspacePermissionV2("bb.policies.create");
  const canUpdate = hasWorkspacePermissionV2("bb.policies.update");
  const canList = hasWorkspacePermissionV2("bb.policies.list");
  const policy = useAppStore((state) =>
    state.getPolicyByParentAndType({
      parentPath: workspace,
      policyType: PolicyType.REVIEW_RULE,
    })
  );
  const [attempt, setAttempt] = useState(0);
  const [readFailed, setReadFailed] = useState(false);
  // The workspace's own row, whatever its enforce flag: null when it has
  // none, undefined while unknown. GetPolicy stands in for a missing row, so
  // only the list tells them apart.
  const [row, setRow] = useState<Policy | null | undefined>(undefined);
  const [draft, setDraft] = useState<ReviewRuleType[]>();
  useEffect(() => {
    if (!enabled || !workspace) return;
    let active = true;
    setDraft(undefined);
    setRow(undefined);
    setReadFailed(false);
    const store = useAppStore.getState();
    const find = {
      parentPath: workspace,
      policyType: PolicyType.REVIEW_RULE,
    };
    // A read that failed leaves nothing in the cache.
    void store
      .getOrFetchPolicyByParentAndType({ ...find, refresh: attempt > 0 })
      .then(() => {
        if (active && store.getPolicyByParentAndType(find) === undefined) {
          setReadFailed(true);
        }
      });
    if (canList) {
      void store
        .listPolicies({ ...find, showDeleted: true })
        .then((policies) => {
          // A save that landed first already knows the row.
          if (active) {
            setRow((known) =>
              known === undefined ? (policies[0] ?? null) : known
            );
          }
        })
        .catch(() => {
          // Unknown keeps the conservative gate below.
        });
    }
    return () => {
      active = false;
    };
  }, [enabled, workspace, canList, attempt]);

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
      const saved = await useAppStore.getState().upsertPolicy({
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
      setRow(saved);
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

  // Without the row known, the save may be either a create or an update.
  const canEdit =
    row === undefined
      ? canCreate && canUpdate
      : row === null
        ? canCreate
        : canUpdate;

  return {
    rules: draft ?? stored,
    readFailed,
    reload: () => setAttempt((n) => n + 1),
    canEdit,
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
          rules={standardRules.rules}
          onChange={standardRules.setRules}
          disabled={!standardRules.canEdit || standardRules.saving}
        />
      )}
    </section>
  );
}
