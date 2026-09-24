import { create } from "@bufbuild/protobuf";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { WORKSPACE_ROUTE_SQL_REVIEW } from "@/app/router/handles";
import { ProjectPageLayout } from "@/components/ProjectPageLayout";
import { RouterLink } from "@/components/RouterLink";
import { StandardRuleSwitches } from "@/components/sql-review/StandardRuleSwitches";
import { SwitchRow } from "@/components/sql-review/SwitchRow";
import {
  effectiveWorkspaceRules,
  reviewRulesOfPolicy,
  STANDARD_RULE_TYPES,
  sameStandardRules,
  storedReviewRules,
} from "@/components/sql-review/standardRules";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { StickyActionFooter } from "@/components/ui/sticky-action-footer";
import { useWorkspaceResourceName } from "@/hooks/useAppState";
import { useProjectByName } from "@/hooks/useProjectByName";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { cn } from "@/lib/utils";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import type { Permission } from "@/types";
import {
  PolicyResourceType,
  PolicyType,
  ReviewRulePolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import type { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";

interface ProjectStandardRules {
  // False until both the project and the workspace policy have been read.
  loaded: boolean;
  readFailed: boolean;
  reload: () => void;
  // Whether the project has rules of its own; otherwise it follows the
  // workspace.
  customized: boolean;
  setCustomized: (customized: boolean) => void;
  // Whether the stored policy is the project's own; the draft may differ.
  storedCustomized: boolean;
  // Whether the project has a policy row at all. A row switched off through
  // the API is not customized, yet a save updates it rather than creating.
  hasOwnRow: boolean;
  // The rules in force: the project's own while customized, else the
  // workspace's, which are undefined when this user cannot read them.
  rules: ReviewRuleType[] | undefined;
  setRules: (rules: ReviewRuleType[]) => void;
  isDirty: boolean;
  saving: boolean;
  revert: () => void;
  save: () => Promise<void>;
}

interface Draft {
  customized: boolean;
  rules: ReviewRuleType[];
}

export function useProjectStandardRules(
  projectName: string
): ProjectStandardRules {
  const { t } = useTranslation();
  const workspace = useWorkspaceResourceName();
  const projectPolicy = useAppStore((state) =>
    state.getPolicyByParentAndType({
      parentPath: projectName,
      policyType: PolicyType.REVIEW_RULE,
    })
  );
  const workspacePolicy = useAppStore((state) =>
    state.getPolicyByParentAndType({
      parentPath: workspace,
      policyType: PolicyType.REVIEW_RULE,
    })
  );
  // The backend checks a workspace policy at workspace scope, so a role held
  // only on this project cannot read the rules the project follows.
  const canReadWorkspace = hasWorkspacePermissionV2("bb.policies.get");
  const [draft, setDraft] = useState<Draft>();
  const [attempt, setAttempt] = useState(0);
  const [loaded, setLoaded] = useState(false);
  const [readFailed, setReadFailed] = useState(false);
  useEffect(() => {
    if (!workspace) return;
    let active = true;
    // The route component is reused when moving to another project.
    setDraft(undefined);
    setLoaded(false);
    setReadFailed(false);
    const store = useAppStore.getState();
    const parents = canReadWorkspace ? [projectName, workspace] : [projectName];
    const finds = parents.map((parentPath) => ({
      parentPath,
      policyType: PolicyType.REVIEW_RULE,
    }));
    void Promise.all(
      finds.map((find) =>
        store.getOrFetchPolicyByParentAndType({ ...find, refresh: attempt > 0 })
      )
    ).then(() => {
      if (!active) return;
      // A read that failed leaves nothing in the cache, where a project
      // without its own policy leaves a policy with no payload.
      if (
        finds.some((find) => store.getPolicyByParentAndType(find) === undefined)
      ) {
        setReadFailed(true);
        return;
      }
      setLoaded(true);
    });
    return () => {
      active = false;
    };
  }, [projectName, workspace, canReadWorkspace, attempt]);

  const storedRules = reviewRulesOfPolicy(projectPolicy);
  const hasOwnRow = storedReviewRules(projectPolicy) !== undefined;
  const workspaceRules = canReadWorkspace
    ? effectiveWorkspaceRules(workspacePolicy)
    : undefined;
  const storedCustomized = storedRules !== undefined;

  const [saving, setSaving] = useState(false);
  const revert = useCallback(() => setDraft(undefined), []);
  const customized = draft?.customized ?? storedCustomized;
  const ownRules = draft?.rules ?? storedRules;
  const isDirty =
    draft !== undefined &&
    (draft.customized !== storedCustomized ||
      (draft.customized && !sameStandardRules(draft.rules, storedRules ?? [])));

  const setCustomized = (next: boolean) => {
    setDraft({
      customized: next,
      rules: ownRules ?? workspaceRules ?? [...STANDARD_RULE_TYPES],
    });
  };

  const save = async () => {
    if (!draft || !isDirty) return;
    setSaving(true);
    try {
      const store = useAppStore.getState();
      if (draft.customized) {
        await store.upsertPolicy({
          parentPath: projectName,
          policy: {
            type: PolicyType.REVIEW_RULE,
            resourceType: PolicyResourceType.PROJECT,
            enforce: true,
            policy: {
              case: "reviewRulePolicy",
              value: create(ReviewRulePolicySchema, { rules: draft.rules }),
            },
          },
        });
      } else if (projectPolicy) {
        await store.deletePolicy(projectPolicy.name);
        await store.getOrFetchPolicyByParentAndType({
          parentPath: projectName,
          policyType: PolicyType.REVIEW_RULE,
          refresh: true,
        });
      }
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
    loaded,
    readFailed,
    reload: () => setAttempt((n) => n + 1),
    customized,
    setCustomized,
    storedCustomized,
    hasOwnRow,
    rules: customized ? ownRules : workspaceRules,
    setRules: (rules) => setDraft({ customized: true, rules }),
    isDirty,
    saving,
    revert,
    save,
  };
}

// The rules read as one centered column, and the footer lines its actions up
// with it.
const STANDARD_RULES_COLUMN = "mx-auto w-full max-w-3xl";

export function ProjectSQLReviewPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useProjectByName(projectName);
  const standardRules = useProjectStandardRules(projectName);
  useUnsavedChangesGuard(standardRules.isDirty);

  const can = (permission: Permission) =>
    hasProjectPermissionV2(project, permission);
  // Switching customization on and editing the project's own rules save the
  // project policy: a create when the project has no row, else an update.
  // Switching customization off deletes the row. Undoing an unsaved switch
  // does neither.
  const canSave = standardRules.hasOwnRow
    ? can("bb.policies.update")
    : can("bb.policies.create");
  const customizeLocked = standardRules.customized
    ? standardRules.storedCustomized && !can("bb.policies.delete")
    : !standardRules.storedCustomized && !canSave;
  const canViewWorkspaceRules =
    hasWorkspacePermissionV2("bb.reviewConfigs.list") &&
    hasWorkspacePermissionV2("bb.policies.get");

  return (
    <ProjectPageLayout>
      <div className={cn(STANDARD_RULES_COLUMN, "flex flex-col gap-y-6 pt-4")}>
        <div className="flex flex-col gap-y-2">
          <h1 className="text-2xl font-bold text-main">
            {t("sql-review.standard-rules.self")}
          </h1>
          <p className="text-sm text-control-light">
            {t("sql-review.standard-rules.project-description")}
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
        {standardRules.loaded && (
          <>
            <SwitchRow
              className="rounded-sm border border-block-border px-4 py-3"
              title={t("sql-review.standard-rules.customize.self")}
              description={
                <>
                  {t("sql-review.standard-rules.customize.description")}
                  {canViewWorkspaceRules && (
                    <>
                      {" "}
                      <RouterLink
                        to={{ name: WORKSPACE_ROUTE_SQL_REVIEW }}
                        className="text-accent hover:underline"
                      >
                        {t(
                          "sql-review.standard-rules.customize.view-workspace-rules"
                        )}
                      </RouterLink>
                    </>
                  )}
                </>
              }
              checked={standardRules.customized}
              onCheckedChange={standardRules.setCustomized}
              disabled={customizeLocked || standardRules.saving}
            />
            {standardRules.rules ? (
              <StandardRuleSwitches
                rules={standardRules.rules}
                onChange={standardRules.setRules}
                disabled={!canSave || standardRules.saving}
                readOnly={!standardRules.customized}
              />
            ) : (
              <Alert
                description={t(
                  "sql-review.standard-rules.workspace-rules-hidden"
                )}
              />
            )}
          </>
        )}
      </div>

      {standardRules.isDirty && (
        <StickyActionFooter
          contentPadding={false}
          contentClassName={STANDARD_RULES_COLUMN}
          left={
            <Button
              appearance="outline"
              disabled={standardRules.saving}
              onClick={standardRules.revert}
            >
              {t("common.cancel")}
            </Button>
          }
          right={
            <Button
              disabled={standardRules.saving}
              onClick={() => void standardRules.save()}
            >
              {t("common.update")}
            </Button>
          }
        />
      )}
    </ProjectPageLayout>
  );
}
