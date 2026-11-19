import { create } from "@bufbuild/protobuf";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { WORKSPACE_ROUTE_SQL_REVIEW } from "@/app/router/handles";
import { FeatureAttention } from "@/components/FeatureAttention";
import { ProjectPageLayout } from "@/components/ProjectPageLayout";
import { RouterLink } from "@/components/RouterLink";
import {
  ReviewAIPolicyField,
  useReviewAIPolicyDraft,
} from "@/components/sql-review/ReviewAIPolicyField";
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
import { FormFieldGroup, FormSection } from "@/components/ui/form";
import { StickyActionFooter } from "@/components/ui/sticky-action-footer";
import { usePlanFeature, useWorkspaceResourceName } from "@/hooks/useAppState";
import { useProjectByName } from "@/hooks/useProjectByName";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
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
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";
import { sqlReviewV2Enabled } from "@/utils/featureGates";

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
  // Resolves whether the rules saved. The page reports the update; a failure
  // is reported by the response middleware and keeps the draft.
  save: () => Promise<boolean>;
}

interface Draft {
  customized: boolean;
  rules: ReviewRuleType[];
}

// `enabled` is false beside SQL Review V1, which has no standard rules to read.
export function useProjectStandardRules(
  projectName: string,
  enabled = true
): ProjectStandardRules {
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
    if (!workspace || !enabled) return;
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
        store.fetchPolicyByParentAndType({ ...find, refresh: attempt > 0 })
      )
    ).then((policies) => {
      if (!active) return;
      // A read that failed may leave an earlier policy in the cache, so the
      // results decide, not the cache.
      if (policies.some((policy) => policy === undefined)) {
        setReadFailed(true);
        return;
      }
      setLoaded(true);
    });
    return () => {
      active = false;
    };
  }, [projectName, workspace, canReadWorkspace, attempt, enabled]);

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
    if (!draft || !isDirty) return true;
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
      return true;
    } catch {
      return false;
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

export function ProjectSQLReviewPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useProjectByName(projectName);
  const can = (permission: Permission) =>
    hasProjectPermissionV2(project, permission);
  // Beside SQL Review V1 the page is the AI review's project policy alone.
  const sqlReviewV2 = sqlReviewV2Enabled();
  const standardRules = useProjectStandardRules(projectName, sqlReviewV2);
  // The AI review is switched for the whole workspace, runs only while AI
  // itself is on, and is an Enterprise feature.
  const licensed = usePlanFeature(PlanFeature.FEATURE_AI_SQL_REVIEW);
  const workspaceAIReviewOn = useAppStore(
    (state) =>
      state.serverInfo?.aiEnabled === true &&
      state.serverInfo?.aiSqlReviewEnabled === true
  );
  const aiReviewOn = licensed && workspaceAIReviewOn;
  const canViewWorkspaceRules = hasWorkspacePermissionV2("bb.policies.get");
  const aiReviewPolicy = useReviewAIPolicyDraft({
    parentPath: projectName,
    resourceType: PolicyResourceType.PROJECT,
    can,
    enabled: aiReviewOn,
  });
  const parts = sqlReviewV2
    ? [standardRules, aiReviewPolicy]
    : [aiReviewPolicy];
  const isDirty = parts.some((part) => part.isDirty);
  const saving = parts.some((part) => part.saving);
  useUnsavedChangesGuard(isDirty);

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
  // Beside V1 the page is one section, centered at the width of the
  // workspace's SQL Review page, and so is its footer.
  const centered = sqlReviewV2 ? undefined : "mx-auto w-full max-w-6xl";
  return (
    // The page fills the height so the footer sits at the bottom even when the
    // content is short.
    <ProjectPageLayout className="min-h-full">
      {sqlReviewV2 && (
        <FormSection
          layout="single-column"
          padding="none"
          title={t("sql-review.standard-rules.self")}
          description={t("sql-review.standard-rules.project-description")}
        >
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
                  // The rules remembered while SYNTAX is off belong to one
                  // project.
                  key={projectName}
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
        </FormSection>
      )}
      <FormSection
        className={centered}
        layout="single-column"
        padding="none"
        title={t("sql-review.ai-review.self")}
        description={t("sql-review.ai-review.project-description")}
      >
        {!licensed && (
          <FeatureAttention
            feature={PlanFeature.FEATURE_AI_SQL_REVIEW}
            compact
          />
        )}
        <FormFieldGroup>
          <SwitchRow
            className="rounded-sm border border-block-border px-4 py-3"
            title={t("sql-review.ai-review.enable.self")}
            description={
              <>
                {t("sql-review.ai-review.set-for-workspace")}
                {canViewWorkspaceRules && (
                  <>
                    {" "}
                    <RouterLink
                      to={{ name: WORKSPACE_ROUTE_SQL_REVIEW }}
                      className="text-accent hover:underline"
                    >
                      {t("sql-review.ai-review.view-workspace-setting")}
                    </RouterLink>
                  </>
                )}
              </>
            }
            checked={aiReviewOn}
            onCheckedChange={() => {}}
            readOnly
          />
          {aiReviewOn && (
            <ReviewAIPolicyField
              policy={aiReviewPolicy}
              standardRules={sqlReviewV2}
              placeholder={t("sql-review.ai-review.policy.project-placeholder")}
              title={t("sql-review.ai-review.policy.project-self")}
              description={t("sql-review.ai-review.policy.project-description")}
              emptyText={t("sql-review.ai-review.policy.project-none")}
            />
          )}
        </FormFieldGroup>
      </FormSection>

      {isDirty && (
        <StickyActionFooter
          className="-mb-4 mt-auto"
          contentClassName={centered}
          contentPadding={false}
          left={
            <Button
              appearance="outline"
              disabled={saving}
              onClick={() => {
                for (const part of parts) part.revert();
              }}
            >
              {t("common.cancel")}
            </Button>
          }
          right={
            <Button
              disabled={saving || aiReviewPolicy.error !== undefined}
              onClick={async () => {
                const saves = parts
                  .filter((part) => part.isDirty)
                  .map((part) => part.save());
                // One click reports one update, and only when every changed
                // part saved.
                if ((await Promise.all(saves)).every(Boolean)) {
                  pushNotification({
                    module: "bytebase",
                    style: "SUCCESS",
                    title: t("common.updated"),
                  });
                }
              }}
            >
              {t("common.update")}
            </Button>
          }
        />
      )}
    </ProjectPageLayout>
  );
}
