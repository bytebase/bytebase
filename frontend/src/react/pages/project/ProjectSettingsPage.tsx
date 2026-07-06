import { create } from "@bufbuild/protobuf";
import { cloneDeep, isEqual } from "lodash-es";
import { ShieldCheck, TriangleAlert, X } from "lucide-react";
import {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { LabelListEditor } from "@/react/components/LabelListEditor";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { ResourceIdField } from "@/react/components/ResourceIdField";
import { RouterLink } from "@/react/components/RouterLink";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import {
  FormControlRow,
  FormField,
  FormFieldGroup,
  FormSection,
} from "@/react/components/ui/form";
import { Input } from "@/react/components/ui/input";
import { NumberInput } from "@/react/components/ui/number-input";
import { StickyActionFooter } from "@/react/components/ui/sticky-action-footer";
import { Switch } from "@/react/components/ui/switch";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import { router, useCurrentRoute } from "@/react/router";
import {
  PROJECT_V1_ROUTE_DASHBOARD,
  WORKSPACE_ROUTE_CUSTOM_APPROVAL,
  WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
  WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
} from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { useSQLReviewStore } from "@/react/stores/sqlReview";
import { useWorkspaceApprovalSettingStore } from "@/react/stores/workspaceApprovalSetting";
import { pushNotification } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { Permission, SQLReviewPolicy } from "@/types";
import { isDefaultProject } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  PolicyResourceType,
  PolicyType,
  QueryDataPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import type { Label } from "@/types/proto-es/v1/project_service_pb";
import {
  LabelSchema,
  Project_ExecutionRetryPolicySchema,
} from "@/types/proto-es/v1/project_service_pb";
import { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  colorToHex,
  convertKVListToLabels,
  convertLabelsToKVList,
  extractProjectResourceName,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  hexToColor,
  sqlReviewPolicySlug,
} from "@/utils";

// ---------------------------------------------------------------------------
// ApprovalFlowIndicator
// ---------------------------------------------------------------------------
function ApprovalFlowIndicator({
  source,
}: {
  source: WorkspaceApprovalSetting_Rule_Source;
}) {
  const { t } = useTranslation();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (!hasWorkspacePermissionV2("bb.settings.get")) return;
    useWorkspaceApprovalSettingStore
      .getState()
      .fetchConfig()
      .then(() => setReady(true));
  }, []);

  const status = useMemo((): "source" | "fallback" | "none" => {
    if (!ready) return "none";
    const store = useWorkspaceApprovalSettingStore.getState();
    if (store.getRulesBySource(source).length > 0) return "source";
    if (
      store.getRulesBySource(
        WorkspaceApprovalSetting_Rule_Source.SOURCE_UNSPECIFIED
      ).length > 0
    )
      return "fallback";
    return "none";
  }, [ready, source]);

  if (!ready) return null;

  const configured = status !== "none";
  const tooltipText =
    status === "source"
      ? t("project.settings.issue-related.approval-flow-configured")
      : status === "fallback"
        ? t("project.settings.issue-related.approval-flow-fallback")
        : t("project.settings.issue-related.approval-flow-not-configured");

  return (
    <Tooltip
      content={
        <div className="flex flex-col gap-y-1">
          <span>{tooltipText}</span>
          <RouterLink
            to={{ name: WORKSPACE_ROUTE_CUSTOM_APPROVAL }}
            className="text-accent underline text-left hover:text-accent-hover"
          >
            {t("project.settings.issue-related.view-approval-flow")}
          </RouterLink>
        </div>
      }
    >
      {configured ? (
        <ShieldCheck className="size-4 text-success" />
      ) : (
        <TriangleAlert className="size-4 text-warning" />
      )}
    </Tooltip>
  );
}

// ---------------------------------------------------------------------------
// Main Page
// ---------------------------------------------------------------------------
export function ProjectSettingsPage() {
  const { t } = useTranslation();
  const projectsByName = useAppStore((s) => s.projectsByName);
  const reviewStore = useSQLReviewStore();

  const projectId = useCurrentRoute().params.projectId as string;
  const projectName = `${projectNamePrefix}${projectId}`;
  // subscribe to re-render on project cache change
  void projectsByName;
  const project = useProjectByName(projectName);
  const isDefault = isDefaultProject(projectName);

  const hasPermission = useCallback(
    (permission: Permission) =>
      project ? hasProjectPermissionV2(project, permission) : false,
    [project]
  );

  const allowEdit = useMemo(
    () =>
      hasPermission("bb.projects.update") && project?.state === State.ACTIVE,
    [hasPermission, project]
  );

  // -----------------------------------------------------------------------
  // General state
  // -----------------------------------------------------------------------
  const [title, setTitle] = useState(project?.title ?? "");
  const [labelKVList, setLabelKVList] = useState<
    { key: string; value: string }[]
  >(() => convertLabelsToKVList(project?.labels ?? {}, true));

  // Sync label state when project labels change externally
  const projectLabels = project?.labels;
  useEffect(() => {
    setLabelKVList(convertLabelsToKVList(projectLabels ?? {}, true));
  }, [projectLabels]);

  // -----------------------------------------------------------------------
  // Security state
  // -----------------------------------------------------------------------
  const reviewPolicyList = reviewStore.reviewPolicyList;
  const currentReviewPolicy = reviewStore.getReviewPolicyByResouce(projectName);
  const [pendingReviewPolicy, setPendingReviewPolicy] = useState<
    SQLReviewPolicy | undefined
  >(undefined);
  const [enforceReview, setEnforceReview] = useState(false);
  const [showReviewDialog, setShowReviewDialog] = useState(false);

  const queryDataPolicy = useAppStore((s) =>
    s.getQueryDataPolicyByParent(projectName)
  );
  const getInitialMaxRows = useCallback(() => {
    const rows = Number(queryDataPolicy?.maximumResultRows ?? 0);
    return rows < 0 ? 0 : rows;
  }, [queryDataPolicy]);
  // `null` represents a transiently empty input while the user is typing;
  // coerced to 0 on save.
  const [maxRows, setMaxRows] = useState<number | null>(() =>
    getInitialMaxRows()
  );

  const [allowRequestRole, setAllowRequestRole] = useState(
    project?.allowRequestRole ?? false
  );
  const [allowJustInTimeAccess, setAllowJustInTimeAccess] = useState(
    project?.allowJustInTimeAccess ?? false
  );

  const hasQueryPolicyFeature = useAppStore((s) =>
    s.hasInstanceFeature(PlanFeature.FEATURE_QUERY_POLICY)
  );

  // -----------------------------------------------------------------------
  // Issue-related state
  // -----------------------------------------------------------------------
  const [issueLabels, setIssueLabels] = useState<Label[]>(() =>
    cloneDeep(project?.issueLabels ?? [])
  );
  const [forceIssueLabels, setForceIssueLabels] = useState(
    project?.forceIssueLabels ?? false
  );
  const [enforceIssueTitle, setEnforceIssueTitle] = useState(
    project?.enforceIssueTitle ?? false
  );
  const [enforceSqlReview, setEnforceSqlReview] = useState(
    project?.enforceSqlReview ?? false
  );
  const [allowSelfApproval, setAllowSelfApproval] = useState(
    project?.allowSelfApproval ?? false
  );
  const [requireIssueApproval, setRequireIssueApproval] = useState(
    project?.requireIssueApproval ?? false
  );
  const [requirePlanCheckNoError, setRequirePlanCheckNoError] = useState(
    project?.requirePlanCheckNoError ?? false
  );
  const [postgresDatabaseTenantMode, setPostgresDatabaseTenantMode] = useState(
    project?.postgresDatabaseTenantMode ?? false
  );
  const [maxRetries, setMaxRetries] = useState<number | null>(
    project?.executionRetryPolicy?.maximumRetries ?? 0
  );
  const [ciSamplingSize, setCiSamplingSize] = useState<number | null>(
    project?.ciSamplingSize ?? 0
  );
  const [parallelTasksPerRollout, setParallelTasksPerRollout] = useState<
    number | null
  >(project?.parallelTasksPerRollout ?? 1);

  // New issue label input
  const [newLabelValue, setNewLabelValue] = useState("");

  // -----------------------------------------------------------------------
  // Danger zone
  // -----------------------------------------------------------------------
  const [dangerAction, setDangerAction] = useState<
    "archive" | "restore" | "delete" | null
  >(null);
  const [executing, setExecuting] = useState(false);

  // -----------------------------------------------------------------------
  // Fetch on mount and when projectName changes
  // -----------------------------------------------------------------------
  const lastFetchedProject = useRef("");
  useEffect(() => {
    if (lastFetchedProject.current === projectName) return;
    lastFetchedProject.current = projectName;
    useSQLReviewStore.getState().fetchReviewPolicyList();
    useAppStore.getState().getOrFetchPolicyByParentAndType({
      parentPath: projectName,
      policyType: PolicyType.DATA_QUERY,
    });
  }, [projectName]);

  // Sync review policy state when it loads or changes externally
  useEffect(() => {
    setPendingReviewPolicy(currentReviewPolicy);
    setEnforceReview(currentReviewPolicy?.enforce ?? false);
  }, [currentReviewPolicy]);

  // Sync max rows when policy loads
  useEffect(() => {
    setMaxRows(getInitialMaxRows());
  }, [getInitialMaxRows]);

  // Saving state
  const [saving, setSaving] = useState(false);

  // -----------------------------------------------------------------------
  // Dirty tracking
  // -----------------------------------------------------------------------
  const isDirty = useMemo(() => {
    if (!project) return false;
    if (title !== project.title) return true;
    if (!isEqual(convertKVListToLabels(labelKVList, false), project.labels))
      return true;
    // SQL review
    if (!isEqual(pendingReviewPolicy, currentReviewPolicy)) return true;
    if (
      enforceReview !== (pendingReviewPolicy?.enforce ?? false) &&
      pendingReviewPolicy
    )
      return true;
    // Max rows
    if ((maxRows ?? 0) !== getInitialMaxRows()) return true;
    // Project toggles
    if (allowRequestRole !== project.allowRequestRole) return true;
    if (allowJustInTimeAccess !== project.allowJustInTimeAccess) return true;
    // Issue-related
    if (!isEqual(issueLabels, project.issueLabels)) return true;
    if (forceIssueLabels !== project.forceIssueLabels) return true;
    if (enforceIssueTitle !== project.enforceIssueTitle) return true;
    if (enforceSqlReview !== project.enforceSqlReview) return true;
    if (allowSelfApproval !== project.allowSelfApproval) return true;
    if (requireIssueApproval !== project.requireIssueApproval) return true;
    if (requirePlanCheckNoError !== project.requirePlanCheckNoError)
      return true;
    if (postgresDatabaseTenantMode !== project.postgresDatabaseTenantMode)
      return true;
    if (
      (maxRetries ?? 0) !== (project.executionRetryPolicy?.maximumRetries ?? 0)
    )
      return true;
    if ((ciSamplingSize ?? 0) !== (project.ciSamplingSize ?? 0)) return true;
    if (
      (parallelTasksPerRollout ?? 0) !== (project.parallelTasksPerRollout ?? 0)
    )
      return true;
    return false;
  }, [
    project,
    title,
    labelKVList,
    pendingReviewPolicy,
    currentReviewPolicy,
    enforceReview,
    maxRows,
    getInitialMaxRows,
    allowRequestRole,
    allowJustInTimeAccess,
    issueLabels,
    forceIssueLabels,
    enforceIssueTitle,
    enforceSqlReview,
    allowSelfApproval,
    requireIssueApproval,
    requirePlanCheckNoError,
    postgresDatabaseTenantMode,
    maxRetries,
    ciSamplingSize,
    parallelTasksPerRollout,
  ]);

  // Unsaved-changes protection: browser close + in-app navigation
  useEffect(() => {
    if (!isDirty) return;
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      e.returnValue = t("common.leave-without-saving");
      e.preventDefault();
    };
    window.addEventListener("beforeunload", onBeforeUnload);
    const removeGuard = router.beforeEach((_to, _from, next) => {
      if (window.confirm(t("common.leave-without-saving"))) {
        next();
      } else {
        next(false);
      }
    });
    return () => {
      window.removeEventListener("beforeunload", onBeforeUnload);
      removeGuard();
    };
  }, [isDirty, t]);

  // -----------------------------------------------------------------------
  // Revert
  // -----------------------------------------------------------------------
  const revert = useCallback(() => {
    if (!project) return;
    setTitle(project.title);
    setLabelKVList(convertLabelsToKVList(project.labels, true));
    setPendingReviewPolicy(currentReviewPolicy);
    setEnforceReview(currentReviewPolicy?.enforce ?? false);
    setMaxRows(getInitialMaxRows());
    setAllowRequestRole(project.allowRequestRole);
    setAllowJustInTimeAccess(project.allowJustInTimeAccess);
    setIssueLabels(cloneDeep(project.issueLabels));
    setForceIssueLabels(project.forceIssueLabels);
    setEnforceIssueTitle(project.enforceIssueTitle);
    setEnforceSqlReview(project.enforceSqlReview);
    setAllowSelfApproval(project.allowSelfApproval);
    setRequireIssueApproval(project.requireIssueApproval);
    setRequirePlanCheckNoError(project.requirePlanCheckNoError);
    setPostgresDatabaseTenantMode(project.postgresDatabaseTenantMode);
    setMaxRetries(project.executionRetryPolicy?.maximumRetries ?? 0);
    setCiSamplingSize(project.ciSamplingSize);
    setParallelTasksPerRollout(project.parallelTasksPerRollout);
  }, [project, currentReviewPolicy, getInitialMaxRows]);

  // -----------------------------------------------------------------------
  // Save
  // -----------------------------------------------------------------------
  const handleSave = useCallback(async () => {
    if (!project) return;
    setSaving(true);
    try {
      // 1. SQL review policy (separate API, not part of project update)
      if (!isEqual(pendingReviewPolicy, currentReviewPolicy)) {
        if (currentReviewPolicy) {
          await reviewStore.upsertReviewConfigTag({
            oldResources: [...currentReviewPolicy.resources],
            newResources: currentReviewPolicy.resources.filter(
              (r) => r !== projectName
            ),
            review: currentReviewPolicy.id,
          });
        }
        if (pendingReviewPolicy) {
          await reviewStore.upsertReviewConfigTag({
            oldResources: [...pendingReviewPolicy.resources],
            newResources: [...pendingReviewPolicy.resources, projectName],
            review: pendingReviewPolicy.id,
          });
        }
      }
      if (
        pendingReviewPolicy &&
        pendingReviewPolicy.enforce !== enforceReview
      ) {
        await reviewStore.upsertReviewPolicy({
          id: pendingReviewPolicy.id,
          enforce: enforceReview,
        });
      }

      // 2. Max rows policy (separate API)
      const maxRowsValue = maxRows ?? 0;
      if (maxRowsValue !== getInitialMaxRows()) {
        await useAppStore.getState().upsertPolicy({
          parentPath: projectName,
          policy: {
            type: PolicyType.DATA_QUERY,
            resourceType: PolicyResourceType.PROJECT,
            policy: {
              case: "queryDataPolicy",
              value: create(QueryDataPolicySchema, {
                ...queryDataPolicy,
                maximumResultRows: maxRowsValue,
              }),
            },
          },
        });
      }

      // 3. All project fields in a single updateProject call
      const updateMask: string[] = [];
      const projectPatch = cloneDeep(project);
      if (title !== project.title && title.trim()) {
        projectPatch.title = title.trim();
        updateMask.push("title");
      }
      const currentLabels = convertKVListToLabels(labelKVList, false);
      if (!isEqual(currentLabels, project.labels)) {
        projectPatch.labels = currentLabels;
        updateMask.push("labels");
      }
      if (allowRequestRole !== project.allowRequestRole) {
        projectPatch.allowRequestRole = allowRequestRole;
        updateMask.push("allow_request_role");
      }
      if (allowJustInTimeAccess !== project.allowJustInTimeAccess) {
        projectPatch.allowJustInTimeAccess = allowJustInTimeAccess;
        updateMask.push("allow_just_in_time_access");
      }
      if (!isEqual(issueLabels, project.issueLabels)) {
        projectPatch.issueLabels = issueLabels;
        updateMask.push("issue_labels");
      }
      if (forceIssueLabels !== project.forceIssueLabels) {
        projectPatch.forceIssueLabels = forceIssueLabels;
        updateMask.push("force_issue_labels");
      }
      if (enforceIssueTitle !== project.enforceIssueTitle) {
        projectPatch.enforceIssueTitle = enforceIssueTitle;
        updateMask.push("enforce_issue_title");
      }
      if (enforceSqlReview !== project.enforceSqlReview) {
        projectPatch.enforceSqlReview = enforceSqlReview;
        updateMask.push("enforce_sql_review");
      }
      if (allowSelfApproval !== project.allowSelfApproval) {
        projectPatch.allowSelfApproval = allowSelfApproval;
        updateMask.push("allow_self_approval");
      }
      if (requireIssueApproval !== project.requireIssueApproval) {
        projectPatch.requireIssueApproval = requireIssueApproval;
        updateMask.push("require_issue_approval");
      }
      if (requirePlanCheckNoError !== project.requirePlanCheckNoError) {
        projectPatch.requirePlanCheckNoError = requirePlanCheckNoError;
        updateMask.push("require_plan_check_no_error");
      }
      if (postgresDatabaseTenantMode !== project.postgresDatabaseTenantMode) {
        projectPatch.postgresDatabaseTenantMode = postgresDatabaseTenantMode;
        updateMask.push("postgres_database_tenant_mode");
      }
      const maxRetriesValue = maxRetries ?? 0;
      if (
        maxRetriesValue !== (project.executionRetryPolicy?.maximumRetries ?? 0)
      ) {
        projectPatch.executionRetryPolicy = create(
          Project_ExecutionRetryPolicySchema,
          { maximumRetries: maxRetriesValue }
        );
        updateMask.push("execution_retry_policy");
      }
      const ciSamplingSizeValue = ciSamplingSize ?? 0;
      if (ciSamplingSizeValue !== (project.ciSamplingSize ?? 0)) {
        projectPatch.ciSamplingSize = ciSamplingSizeValue;
        updateMask.push("ci_sampling_size");
      }
      const parallelTasksPerRolloutValue = parallelTasksPerRollout ?? 0;
      if (
        parallelTasksPerRolloutValue !== (project.parallelTasksPerRollout ?? 0)
      ) {
        projectPatch.parallelTasksPerRollout = parallelTasksPerRolloutValue;
        updateMask.push("parallel_tasks_per_rollout");
      }
      if (updateMask.length > 0) {
        const updatedProject = await useAppStore
          .getState()
          .updateProject(projectPatch, updateMask);
        if (updateMask.includes("issue_labels")) {
          setIssueLabels(cloneDeep(updatedProject.issueLabels));
        }
      }

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("project.settings.success-updated"),
      });
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("project.settings.update-failed"),
      });
    } finally {
      setSaving(false);
    }
  }, [
    project,
    reviewStore,
    projectName,
    title,
    labelKVList,
    pendingReviewPolicy,
    currentReviewPolicy,
    enforceReview,
    maxRows,
    getInitialMaxRows,
    queryDataPolicy,
    allowRequestRole,
    allowJustInTimeAccess,
    issueLabels,
    forceIssueLabels,
    enforceIssueTitle,
    enforceSqlReview,
    allowSelfApproval,
    requireIssueApproval,
    requirePlanCheckNoError,
    postgresDatabaseTenantMode,
    maxRetries,
    ciSamplingSize,
    parallelTasksPerRollout,
    t,
  ]);

  // -----------------------------------------------------------------------
  // Danger zone handlers
  // -----------------------------------------------------------------------
  const handleDangerConfirm = useCallback(async () => {
    if (!project || !dangerAction) return;
    setExecuting(true);
    try {
      if (dangerAction === "archive") {
        await useAppStore.getState().archiveProject(project);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: `${project.title || project.name} ${t("common.archived")}`,
        });
      } else if (dangerAction === "restore") {
        await useAppStore.getState().restoreProject(project);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: `${project.title || project.name} ${t("common.restored")}`,
        });
      } else if (dangerAction === "delete") {
        await useAppStore.getState().deleteProject(project.name);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: `${project.title || project.name} ${t("common.deleted")}`,
        });
      }
      router.push({ name: PROJECT_V1_ROUTE_DASHBOARD });
    } finally {
      setExecuting(false);
      setDangerAction(null);
    }
  }, [project, dangerAction, t]);

  // Label validation errors (set by LabelListEditor via onErrorsChange)
  const [labelErrors, setLabelErrors] = useState<string[]>([]);

  // -----------------------------------------------------------------------
  // Issue label helpers
  // -----------------------------------------------------------------------
  const addIssueLabel = useCallback(() => {
    const trimmed = newLabelValue.trim();
    if (!trimmed) return;
    if (issueLabels.some((l) => l.value === trimmed)) return;
    setIssueLabels((prev) => [
      ...prev,
      create(LabelSchema, {
        value: trimmed,
        color: hexToColor("#4f46e5"),
        group: "",
      }),
    ]);
    setNewLabelValue("");
  }, [newLabelValue, issueLabels]);

  const removeIssueLabel = useCallback((index: number) => {
    setIssueLabels((prev) => {
      const next = prev.filter((_, i) => i !== index);
      if (next.length === 0) {
        setForceIssueLabels(false);
      }
      return next;
    });
  }, []);

  const updateIssueLabelColor = useCallback((index: number, color: string) => {
    setIssueLabels((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], color: hexToColor(color) };
      return next;
    });
  }, []);

  if (!project) return null;

  const canUpdateProject = hasPermission("bb.projects.update");
  const canGetPolicies = hasPermission("bb.policies.get");
  const canUpdatePolicies = hasPermission("bb.policies.update");
  const canDelete = hasPermission("bb.projects.delete");
  const canUndelete = hasPermission("bb.projects.undelete");

  // -----------------------------------------------------------------------
  // Render
  // -----------------------------------------------------------------------
  return (
    <div className="w-full flex flex-col gap-y-0 pt-4 px-4">
      <div className="divide-y divide-block-border">
        {/* ============================================================= */}
        {/* Section 1: General */}
        {/* ============================================================= */}
        <FormSection title={t("common.general")}>
          <PermissionGuard
            permissions={["bb.projects.update"]}
            project={project}
            display="block"
          >
            <form className="w-full">
              <FormFieldGroup>
                <FormField
                  title={
                    <>
                      {t("common.name")} <span className="text-error">*</span>
                    </>
                  }
                >
                  <Input
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    disabled={
                      !canUpdateProject ||
                      isDefault ||
                      project.state !== State.ACTIVE
                    }
                    required
                  />
                  <ResourceIdField
                    readonly
                    value={extractProjectResourceName(project.name)}
                    resourceName={t("common.project")}
                  />
                </FormField>

                {/* Project Labels */}
                <FormField
                  title={t("project.settings.project-labels.self")}
                  description={t("project.settings.project-labels.description")}
                >
                  <LabelListEditor
                    kvList={labelKVList}
                    onChange={setLabelKVList}
                    readonly={!canUpdateProject}
                    onErrorsChange={setLabelErrors}
                  />
                </FormField>
              </FormFieldGroup>
            </form>
          </PermissionGuard>
        </FormSection>

        {/* ============================================================= */}
        {/* Section 2: Security & Policy */}
        {/* ============================================================= */}
        <FormSection title={t("settings.sidebar.security-and-policy")}>
          <FormFieldGroup>
            {/* SQL Review + Max Rows: gated on bb.policies.get */}
            {canGetPolicies && (
              <>
                {/* SQL Review */}
                {hasWorkspacePermissionV2("bb.reviewConfigs.get") && (
                  <FormField title={t("sql-review.title")}>
                    <div>
                      {pendingReviewPolicy ? (
                        <FormControlRow>
                          <Switch
                            checked={enforceReview}
                            onCheckedChange={setEnforceReview}
                            disabled={
                              !hasWorkspacePermissionV2(
                                "bb.reviewConfigs.update"
                              )
                            }
                          />
                          <span
                            className="text-sm font-medium text-accent cursor-pointer hover:underline"
                            onClick={() =>
                              router.push({
                                name: WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
                                params: {
                                  sqlReviewPolicySlug:
                                    sqlReviewPolicySlug(pendingReviewPolicy),
                                },
                              })
                            }
                          >
                            {pendingReviewPolicy.name}
                          </span>
                          {canUpdatePolicies && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => {
                                setPendingReviewPolicy(undefined);
                                setEnforceReview(false);
                              }}
                            >
                              <X className="size-4" />
                            </Button>
                          )}
                        </FormControlRow>
                      ) : (
                        <Button
                          variant="outline"
                          disabled={
                            !canUpdatePolicies ||
                            !hasWorkspacePermissionV2("bb.reviewConfigs.list")
                          }
                          onClick={() => setShowReviewDialog(true)}
                        >
                          {t("sql-review.configure-policy")}
                        </Button>
                      )}
                    </div>
                  </FormField>
                )}

                {/* Maximum SQL Result Rows */}
                <FormField
                  title={
                    <FormControlRow className="w-auto">
                      <span>
                        {t(
                          "settings.general.workspace.maximum-sql-result.rows.self"
                        )}
                      </span>
                      <FeatureBadge
                        feature={PlanFeature.FEATURE_QUERY_POLICY}
                      />
                    </FormControlRow>
                  }
                  description={
                    <>
                      {t(
                        "settings.general.workspace.maximum-sql-result.rows.description"
                      )}{" "}
                      <span className="font-semibold">
                        {t("settings.general.workspace.no-limit")}
                      </span>
                    </>
                  }
                >
                  <FormControlRow>
                    <NumberInput
                      className="w-60"
                      min={0}
                      value={maxRows}
                      onValueChange={setMaxRows}
                      disabled={!hasQueryPolicyFeature || !canUpdatePolicies}
                    />
                    <span className="text-sm text-control-light">
                      {t(
                        "settings.general.workspace.maximum-sql-result.rows.rows"
                      )}
                    </span>
                  </FormControlRow>
                </FormField>
              </>
            )}

            {/* Allow Request Role */}
            <ToggleRow
              checked={allowRequestRole}
              onCheckedChange={setAllowRequestRole}
              disabled={!canUpdateProject}
              label={t(
                "project.settings.issue-related.allow-request-role.self"
              )}
              description={t(
                "project.settings.issue-related.allow-request-role.description"
              )}
              trailing={
                <ApprovalFlowIndicator
                  source={WorkspaceApprovalSetting_Rule_Source.REQUEST_ROLE}
                />
              }
            />

            {/* Allow JIT Access */}
            <ToggleRow
              checked={allowJustInTimeAccess}
              onCheckedChange={setAllowJustInTimeAccess}
              disabled={!canUpdateProject}
              label={t("project.settings.issue-related.allow-jit.self")}
              description={t(
                "project.settings.issue-related.allow-jit.description"
              )}
              trailing={
                <ApprovalFlowIndicator
                  source={WorkspaceApprovalSetting_Rule_Source.REQUEST_ACCESS}
                />
              }
            />
          </FormFieldGroup>
        </FormSection>

        {/* ============================================================= */}
        {/* Section 3: Issue-Related */}
        {/* ============================================================= */}
        <FormSection
          id="issue-related"
          title={t("project.settings.issue-related.self")}
        >
          <FormFieldGroup>
            {/* Issue Labels */}
            <FormField
              title={t("project.settings.issue-related.labels.self")}
              description={t(
                "project.settings.issue-related.labels.description"
              )}
            >
              <div className="flex flex-wrap items-center gap-2">
                {issueLabels.map((label, index) => (
                  <span
                    key={index}
                    className="inline-flex h-9 items-center gap-x-2 border border-control-border rounded-sm px-1 py-1"
                  >
                    <input
                      type="color"
                      value={label.color ? colorToHex(label.color) : "#4f46e5"}
                      onChange={(e) =>
                        updateIssueLabelColor(index, e.target.value)
                      }
                      disabled={!canUpdateProject}
                      className="w-5 h-6 rounded-sm cursor-pointer border-0 p-0"
                    />
                    <span className="text-sm">{label.value}</span>
                    {canUpdateProject && (
                      <button
                        type="button"
                        className="text-control-light hover:text-main"
                        onClick={() => removeIssueLabel(index)}
                      >
                        <X className="size-3" />
                      </button>
                    )}
                  </span>
                ))}
                {canUpdateProject && (
                  <div className="inline-flex items-center gap-x-1">
                    <Input
                      className="w-48"
                      placeholder={t(
                        "project.settings.issue-related.labels.placeholder"
                      )}
                      value={newLabelValue}
                      onChange={(e) => setNewLabelValue(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") {
                          e.preventDefault();
                          addIssueLabel();
                        }
                      }}
                    />
                    <Button
                      variant="outline"
                      size="md"
                      onClick={addIssueLabel}
                      disabled={!newLabelValue.trim()}
                    >
                      {t("common.add")}
                    </Button>
                  </div>
                )}
              </div>
            </FormField>

            {/* Boolean toggles */}
            <ToggleRow
              checked={forceIssueLabels}
              onCheckedChange={setForceIssueLabels}
              disabled={!canUpdateProject || issueLabels.length === 0}
              label={t(
                "project.settings.issue-related.labels.force-issue-labels.self"
              )}
              description={t(
                "project.settings.issue-related.labels.force-issue-labels.description"
              )}
              warning={
                canUpdateProject && issueLabels.length === 0
                  ? t(
                      "project.settings.issue-related.labels.force-issue-labels.warning"
                    )
                  : undefined
              }
            />
            <ToggleRow
              checked={enforceIssueTitle}
              onCheckedChange={setEnforceIssueTitle}
              disabled={!canUpdateProject}
              label={t(
                "project.settings.issue-related.enforce-issue-title.self"
              )}
              description={t(
                "project.settings.issue-related.enforce-issue-title.description"
              )}
            />
            <ToggleRow
              checked={enforceSqlReview}
              onCheckedChange={setEnforceSqlReview}
              disabled={!canUpdateProject}
              label={t(
                "project.settings.issue-related.enforce-sql-review.self"
              )}
              description={t(
                "project.settings.issue-related.enforce-sql-review.description"
              )}
            />
            <ToggleRow
              checked={allowSelfApproval}
              onCheckedChange={setAllowSelfApproval}
              disabled={!canUpdateProject}
              label={t(
                "project.settings.issue-related.allow-self-approval.self"
              )}
              description={t(
                "project.settings.issue-related.allow-self-approval.description"
              )}
            />
            <ToggleRow
              checked={requireIssueApproval}
              onCheckedChange={setRequireIssueApproval}
              disabled={!canUpdateProject}
              label={t(
                "project.settings.issue-related.require-issue-approval.self"
              )}
              description={t(
                "project.settings.issue-related.require-issue-approval.description"
              )}
            />
            <ToggleRow
              checked={requirePlanCheckNoError}
              onCheckedChange={setRequirePlanCheckNoError}
              disabled={!canUpdateProject}
              label={t(
                "project.settings.issue-related.require-plan-check-no-error.self"
              )}
              description={t(
                "project.settings.issue-related.require-plan-check-no-error.description"
              )}
            />
            <ToggleRow
              checked={postgresDatabaseTenantMode}
              onCheckedChange={setPostgresDatabaseTenantMode}
              disabled={!canUpdateProject}
              label={t(
                "project.settings.issue-related.postgres-database-tenant-mode.self"
              )}
              description={t(
                "project.settings.issue-related.postgres-database-tenant-mode.description"
              )}
            />

            {/* Numeric inputs */}
            <NumericRow
              label={t("project.settings.issue-related.max-retries.self")}
              description={t(
                "project.settings.issue-related.max-retries.description"
              )}
              value={maxRetries}
              onChange={setMaxRetries}
              disabled={!canUpdateProject}
              suffix="Times"
            />
            <NumericRow
              label={t("project.settings.issue-related.ci-sampling-size.self")}
              description={t(
                "project.settings.issue-related.ci-sampling-size.description"
              )}
              value={ciSamplingSize}
              onChange={setCiSamplingSize}
              disabled={!canUpdateProject}
            />
            <NumericRow
              label={t(
                "project.settings.issue-related.parallel_tasks_per_rollout.self"
              )}
              description={t(
                "project.settings.issue-related.parallel_tasks_per_rollout.description"
              )}
              value={parallelTasksPerRollout}
              onChange={setParallelTasksPerRollout}
              disabled={!canUpdateProject}
            />
          </FormFieldGroup>
        </FormSection>

        {/* ============================================================= */}
        {/* Section 4: Danger Zone */}
        {/* ============================================================= */}
        <FormSection title={t("common.danger-zone")}>
          <div className="border border-error-alpha bg-error-alpha rounded-sm divide-y divide-error-alpha">
            {/* Archive / Restore */}
            <div className="p-6 flex items-start justify-between gap-x-6">
              {project.state === State.ACTIVE ? (
                <>
                  <FormField
                    className="flex-1"
                    title={t("common.archive-resource", {
                      type: t("common.project"),
                    })}
                    description={t("common.archive-description", {
                      name: project.title || project.name,
                    })}
                  />
                  <Button
                    variant="outline"
                    disabled={!canDelete || executing}
                    onClick={() => setDangerAction("archive")}
                  >
                    {t("common.archive")}
                  </Button>
                </>
              ) : project.state === State.DELETED ? (
                <>
                  <FormField
                    className="flex-1"
                    title={t("project.settings.restore.title")}
                    description={t("project.settings.restore.btn-text")}
                  />
                  <Button
                    variant="outline"
                    disabled={!canUndelete || executing}
                    onClick={() => setDangerAction("restore")}
                  >
                    {t("common.restore")}
                  </Button>
                </>
              ) : null}
            </div>

            {/* Delete */}
            <div className="p-6 flex items-start justify-between gap-x-6">
              <FormField
                className="flex-1"
                title={
                  <span className="text-error">
                    {t("common.delete-resource", {
                      type: t("common.project"),
                    })}
                  </span>
                }
                description={t("common.delete-resource-description", {
                  name: project.title || project.name,
                })}
              />
              <Button
                variant="destructive"
                disabled={!canDelete || executing}
                onClick={() => setDangerAction("delete")}
              >
                {t("common.delete")}
              </Button>
            </div>
          </div>
        </FormSection>

        {/* ============================================================= */}
        {/* Save / Cancel Bar */}
        {/* ============================================================= */}
        {allowEdit && isDirty && (
          <StickyActionFooter
            left={
              <Button variant="outline" onClick={revert}>
                {t("common.cancel")}
              </Button>
            }
            right={
              <Button
                onClick={handleSave}
                disabled={labelErrors.length > 0 || saving || !title.trim()}
              >
                {t("common.update")}
              </Button>
            }
          />
        )}
      </div>

      {/* ============================================================= */}
      {/* Dialogs */}
      {/* ============================================================= */}

      {/* SQL Review Policy Select Dialog */}
      <Dialog open={showReviewDialog} onOpenChange={setShowReviewDialog}>
        <DialogContent className="p-6">
          <DialogTitle>{t("sql-review.configure-policy")}</DialogTitle>
          <div className="mt-4 flex flex-col gap-y-2 max-h-96 overflow-y-auto">
            {reviewPolicyList.length === 0 ? (
              <div className="flex flex-col items-center gap-y-3 py-4">
                <p className="text-sm text-control-light">
                  {t("common.no-data")}
                </p>
                {hasWorkspacePermissionV2("bb.reviewConfigs.create") && (
                  <Button
                    variant="outline"
                    onClick={() => {
                      setShowReviewDialog(false);
                      router.push({
                        name: WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
                        query: { attachedResource: projectName },
                      });
                    }}
                  >
                    {t("common.create")}
                  </Button>
                )}
              </div>
            ) : (
              reviewPolicyList.map((policy) => (
                <button
                  key={policy.id}
                  type="button"
                  className="w-full text-left px-4 py-3 border border-control-border rounded-sm hover:bg-control-bg transition-colors"
                  onClick={() => {
                    setPendingReviewPolicy(policy);
                    setEnforceReview(true);
                    setShowReviewDialog(false);
                  }}
                >
                  <div className="font-medium">{policy.name}</div>
                  {policy.resources.length > 0 && (
                    <div className="text-xs text-control-light mt-1">
                      {policy.resources.length}{" "}
                      {t("common.resource", {
                        count: policy.resources.length,
                      })}
                    </div>
                  )}
                </button>
              ))
            )}
          </div>
          <div className="mt-4 flex justify-end">
            <Button
              variant="outline"
              onClick={() => setShowReviewDialog(false)}
            >
              {t("common.cancel")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Danger Zone Confirmation Dialog */}
      <Dialog
        open={dangerAction !== null}
        onOpenChange={(open) => {
          if (!open) setDangerAction(null);
        }}
      >
        <DialogContent className="p-6 max-w-md">
          <DialogTitle>
            {dangerAction === "archive" && t("common.confirm-archive")}
            {dangerAction === "restore" && t("project.settings.restore.title")}
            {dangerAction === "delete" && t("common.confirm-delete")}
          </DialogTitle>
          <p className="text-sm text-control-light mt-2">
            {dangerAction === "archive" &&
              t("project.settings.confirm-archive-project", {
                name: project.title || project.name,
              })}
            {dangerAction === "restore" &&
              `${t("project.settings.restore.title")} '${project.title || project.name}'?`}
            {dangerAction === "delete" &&
              t("project.settings.confirm-delete-project", {
                name: project.title || project.name,
              })}
          </p>
          <div className="mt-6 flex justify-end items-center gap-x-2">
            <Button
              variant="outline"
              onClick={() => setDangerAction(null)}
              disabled={executing}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant={dangerAction === "delete" ? "destructive" : "default"}
              onClick={handleDangerConfirm}
              disabled={executing}
            >
              {t("common.confirm")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Helper components
// ---------------------------------------------------------------------------

function ToggleRow({
  checked,
  onCheckedChange,
  disabled,
  label,
  description,
  trailing,
  warning,
}: {
  checked: boolean;
  onCheckedChange: (v: boolean) => void;
  disabled: boolean;
  label: string;
  description: string;
  trailing?: ReactNode;
  warning?: string;
}) {
  return (
    <FormField
      title={
        <FormControlRow>
          <Switch
            checked={checked}
            onCheckedChange={onCheckedChange}
            disabled={disabled}
          />
          <span className="text-sm font-medium">{label}</span>
          {trailing}
          {warning && (
            <Tooltip content={warning}>
              <TriangleAlert className="size-4 text-warning" />
            </Tooltip>
          )}
        </FormControlRow>
      }
      description={description}
    />
  );
}

function NumericRow({
  label,
  description,
  value,
  onChange,
  disabled,
  suffix,
}: {
  label: string;
  description: string;
  value: number | null;
  onChange: (v: number | null) => void;
  disabled: boolean;
  suffix?: string;
}) {
  return (
    <FormField title={label} description={description}>
      <FormControlRow>
        <NumberInput
          className="w-60"
          min={0}
          value={value}
          onValueChange={onChange}
          disabled={disabled}
        />
        {suffix && <span className="text-sm text-control-light">{suffix}</span>}
      </FormControlRow>
    </FormField>
  );
}
