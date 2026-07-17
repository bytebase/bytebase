import { clone, create } from "@bufbuild/protobuf";
import { EllipsisVertical, Loader2 } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect, planServiceClientConnect } from "@/connect";
import {
  type IssueLabel,
  IssueLabelSelect,
} from "@/react/components/IssueLabelSelect";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { cn } from "@/react/lib/utils";
import { router } from "@/react/router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  BatchUpdateIssuesStatusRequestSchema,
  IssueSchema,
  IssueStatus,
  ListIssueCommentsRequestSchema,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  extractPlanUID,
  extractProjectResourceName,
  extractSheetUID,
  hasProjectPermissionV2,
} from "@/utils";
import { usePlanDetailSpecValidation } from "../hooks/usePlanDetailSpecValidation";
import { focusPlanPhase } from "../shell/focusPhase";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import {
  createPlanWithDraftReview,
  DraftReviewIssueCreationError,
  getCreateIssueBlockingErrors,
  getCreateIssueConfirmErrors,
  getCreatePlanBlockingReasons,
  hasChecksWarning,
  submitDraftReview,
} from "../utils/header";
import { getLocalSheetByName, removeLocalSheet } from "../utils/localSheet";
import { PlanLifecycleSlot } from "./lifecycle/PlanLifecycleSlot";
import { PlanLifecycleStamp } from "./lifecycle/PlanLifecycleStamp";
import { slotHasPrimaryControl } from "./lifecycle/planLifecycleHeaderState";
import { usePlanLifecycleHeader } from "./lifecycle/usePlanLifecycleHeader";

// The sticky title/action row. Description + metadata live in
// PlanDetailHeaderDetails so they scroll away while this row stays pinned.
export function PlanDetailHeader() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { patchState, setEditing } = page;
  const currentUser = page.currentUser;
  const project = page.project;
  const draftIssue = page.issue?.draft === true;
  const persistedTitle =
    page.issue && !draftIssue ? page.issue.title : page.plan.title;
  const [title, setTitle] = useState(persistedTitle);
  const [editingTitle, setEditingTitle] = useState(false);
  const [updating, setUpdating] = useState(false);
  const [showReviewPopover, setShowReviewPopover] = useState(false);
  const [selectedLabels, setSelectedLabels] = useState<string[]>([]);
  const [checksWarningAcknowledged, setChecksWarningAcknowledged] =
    useState(false);
  const [submittingReview, setSubmittingReview] = useState(false);
  const { emptySpecIdSet } = usePlanDetailSpecValidation(page.plan.specs ?? []);
  const titleInputRef = useRef<HTMLInputElement>(null);
  const titleAutoFocusedRef = useRef(false);
  const pageKeyRef = useRef(page.pageKey);
  pageKeyRef.current = page.pageKey;
  const createPlanBlockingReasons = useMemo(
    () =>
      getCreatePlanBlockingReasons({
        title: page.plan.title,
        emptySpecCount: emptySpecIdSet.size,
        t,
      }),
    [emptySpecIdSet.size, page.plan.title, t]
  );
  const canCreatePlan = hasProjectPermissionV2(project, "bb.plans.create");
  const canCreateIssue = hasProjectPermissionV2(project, "bb.issues.create");
  const canCreateDraftReview = canCreatePlan && canCreateIssue;
  const createPermissionReason = canCreateDraftReview
    ? undefined
    : t("common.missing-required-permission", {
        permissions: ["bb.plans.create", "bb.issues.create"]
          .filter(
            (permission) =>
              !hasProjectPermissionV2(
                project,
                permission as "bb.plans.create" | "bb.issues.create"
              )
          )
          .join(", "),
      });

  const canUpdatePlan =
    page.plan.creator === currentUser.name ||
    hasProjectPermissionV2(project, "bb.plans.update");

  useEffect(() => {
    setTitle(persistedTitle);
    setEditingTitle(false);
    setUpdating(false);
    setShowReviewPopover(false);
    setSelectedLabels(draftIssue ? (page.issue?.labels ?? []) : []);
    setChecksWarningAcknowledged(false);
    setSubmittingReview(false);
    titleAutoFocusedRef.current = false;
    setEditing("title", false);
  }, [page.pageKey]);

  useEffect(() => {
    if (!editingTitle) {
      setTitle((prev) => (prev === persistedTitle ? prev : persistedTitle));
    }
  }, [editingTitle, persistedTitle]);

  useEffect(() => {
    if (draftIssue && !showReviewPopover) {
      setSelectedLabels(page.issue?.labels ?? []);
    }
  }, [draftIssue, page.issue?.labels, showReviewPopover]);

  const allowTitleEdit = useMemo(() => {
    if (page.readonly) return false;
    if (page.isCreating) return true;
    if (!page.issue && page.plan.hasRollout) return false;
    if (draftIssue) return canUpdatePlan;
    if (page.issue) {
      return hasProjectPermissionV2(project, "bb.issues.update");
    }
    return canUpdatePlan;
  }, [
    canUpdatePlan,
    draftIssue,
    page.isCreating,
    page.issue,
    page.plan.hasRollout,
    page.readonly,
    project,
  ]);

  useEffect(() => {
    // Route changes within the plan-detail page (create → existing → create)
    // re-render the same React root, so the guard must reset when leaving
    // create mode — otherwise the next create visit never re-focuses.
    if (!page.isCreating) {
      titleAutoFocusedRef.current = false;
      return;
    }
    if (!page.ready || titleAutoFocusedRef.current) return;
    titleAutoFocusedRef.current = true;
    titleInputRef.current?.focus();
  }, [page.isCreating, page.ready]);

  // The resolver owns "what does the header show": the create / ready-for-review
  // advances surface as lifecycle states, replacing the old ad-hoc booleans.
  const lifecycle = usePlanLifecycleHeader(page);
  const showClosePlan =
    !page.isCreating &&
    !page.plan.hasRollout &&
    page.plan.state === State.ACTIVE &&
    (!page.issue || draftIssue) &&
    canUpdatePlan;
  const showReopenPlan =
    !page.isCreating &&
    !page.plan.hasRollout &&
    page.plan.state === State.DELETED &&
    (!page.issue || draftIssue) &&
    canUpdatePlan;

  // Submitted issues retain issue-status actions. Draft lifecycle changes flow
  // through the Plan service, which synchronizes the linked draft Issue.
  const canUpdateIssue = hasProjectPermissionV2(project, "bb.issues.update");
  const showCloseIssue =
    !!page.issue &&
    !draftIssue &&
    page.issue.status === IssueStatus.OPEN &&
    !page.plan.hasRollout &&
    canUpdateIssue;
  const showReopenIssue =
    !!page.issue &&
    !draftIssue &&
    page.issue.status === IssueStatus.CANCELED &&
    canUpdateIssue;

  const submitDisabled = page.isEditing || !canUpdateIssue;
  const submitDisabledReason = page.isEditing
    ? t("plan.editor.save-changes-before-continuing")
    : !canUpdateIssue
      ? t("plan.draft-update-permission-required")
      : undefined;

  const saveTitle = async () => {
    if (page.isCreating) {
      patchState({
        plan: {
          ...page.plan,
          title,
        },
      });
      setEditingTitle(false);
      setEditing("title", false);
      return;
    }

    // Skip the API round-trip when nothing changed so we don't pollute the
    // issue timeline with "changed name from X to X".
    const trimmed = title.trim();
    const currentTitle =
      page.issue && !draftIssue ? page.issue.title : page.plan.title;
    if (trimmed === currentTitle) {
      setTitle(currentTitle);
      setEditingTitle(false);
      setEditing("title", false);
      return;
    }

    const actionPageKey = page.pageKey;
    try {
      setUpdating(true);
      if (page.issue && !draftIssue) {
        const issuePatch = create(IssueSchema, {
          ...page.issue,
          title,
        });
        const response = await issueServiceClientConnect.updateIssue(
          create(UpdateIssueRequestSchema, {
            issue: issuePatch,
            updateMask: { paths: ["title"] },
          })
        );
        if (pageKeyRef.current !== actionPageKey) return;
        patchState({ issue: response });
      } else {
        const planPatch = create(PlanSchema, {
          ...page.plan,
          title,
        });
        const response = await planServiceClientConnect.updatePlan(
          create(UpdatePlanRequestSchema, {
            plan: planPatch,
            updateMask: { paths: ["title"] },
          })
        );
        if (pageKeyRef.current !== actionPageKey) return;
        patchState({ plan: response });
      }
    } catch (error) {
      if (pageKeyRef.current !== actionPageKey) return;
      setTitle(currentTitle);
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: String(error),
      });
    } finally {
      if (pageKeyRef.current === actionPageKey) {
        setUpdating(false);
        setEditingTitle(false);
        setEditing("title", false);
      }
    }
  };

  const updatePlanState = async (state: State) => {
    const actionPageKey = page.pageKey;
    try {
      const planPatch = clone(PlanSchema, page.plan);
      planPatch.state = state;
      const updated = await planServiceClientConnect.updatePlan(
        create(UpdatePlanRequestSchema, {
          plan: planPatch,
          updateMask: { paths: ["state"] },
        })
      );
      if (pageKeyRef.current !== actionPageKey) return;
      if (draftIssue && page.issue) {
        const issue = clone(IssueSchema, page.issue);
        issue.status =
          state === State.DELETED ? IssueStatus.CANCELED : IssueStatus.OPEN;
        patchState({ plan: updated, issue });
      } else {
        patchState({ plan: updated });
      }
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch (error) {
      if (pageKeyRef.current !== actionPageKey) return;
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    }
  };

  const updateIssueStatus = async (status: IssueStatus) => {
    const issue = page.issue;
    if (!issue) return;
    const actionPageKey = page.pageKey;
    try {
      await issueServiceClientConnect.batchUpdateIssuesStatus(
        create(BatchUpdateIssuesStatusRequestSchema, {
          parent: project.name,
          issues: [issue.name],
          status,
        })
      );
      if (pageKeyRef.current !== actionPageKey) return;
      // Closing / reopening records a system comment — refresh page state and the
      // issue comments so the review timeline reflects it (like issue detail).
      await Promise.all([
        page.refreshState(),
        useAppStore.getState().listIssueComments(
          create(ListIssueCommentsRequestSchema, {
            parent: issue.name,
            pageSize: 1000,
          })
        ),
      ]);
      if (pageKeyRef.current !== actionPageKey) return;
      // Land on the review section so the close/reopen system comment and the
      // updated status are visible (consistent with the other header advances).
      focusPlanPhase("review", page.expandPhase);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch (error) {
      if (pageKeyRef.current !== actionPageKey) return;
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    }
  };

  // Secondary lifecycle actions collapse into a "..." overflow menu beside the
  // primary slot, matching the issue detail page — the slot keeps one action or
  // status, everything else lives in the menu.
  const secondaryActions: {
    key: string;
    label: string;
    onSelect: () => void;
  }[] = [];
  if (showClosePlan) {
    secondaryActions.push({
      key: "close-plan",
      label: t("common.close"),
      onSelect: () => {
        if (window.confirm(t("plan.state.close-confirm"))) {
          void updatePlanState(State.DELETED);
        }
      },
    });
  }
  if (showReopenPlan) {
    secondaryActions.push({
      key: "reopen-plan",
      label: t("common.reopen"),
      onSelect: () => {
        if (window.confirm(t("plan.state.reopen-confirm"))) {
          void updatePlanState(State.ACTIVE);
        }
      },
    });
  }
  if (showCloseIssue) {
    secondaryActions.push({
      key: "close-issue",
      label: t("issue.batch-transition.close"),
      onSelect: () => {
        if (window.confirm(t("plan.state.close-review-confirm"))) {
          void updateIssueStatus(IssueStatus.CANCELED);
        }
      },
    });
  }
  if (showReopenIssue) {
    secondaryActions.push({
      key: "reopen-issue",
      label: t("issue.batch-transition.reopen"),
      onSelect: () => {
        if (window.confirm(t("plan.state.reopen-review-confirm"))) {
          void updateIssueStatus(IssueStatus.OPEN);
        }
      },
    });
  }

  // With no primary in the slot (terminal / none), surface the first secondary
  // action directly (e.g. Reopen) rather than hiding it; the rest stay in the
  // overflow menu.
  const slotHasPrimary = slotHasPrimaryControl(lifecycle);
  const promotedAction =
    !slotHasPrimary && secondaryActions.length > 0
      ? secondaryActions[0]
      : undefined;
  const overflowActions = promotedAction
    ? secondaryActions.slice(1)
    : secondaryActions;

  const createSheets = async (actionPageKey: string) => {
    for (const spec of page.plan.specs) {
      let config = null;
      if (spec.config?.case === "changeDatabaseConfig")
        config = spec.config.value;
      else if (spec.config?.case === "exportDataConfig")
        config = spec.config.value;
      if (!config) continue;
      const uid = extractSheetUID(config.sheet);
      if (uid.startsWith("-")) {
        const local = getLocalSheetByName(config.sheet);
        const createdSheet = await useAppStore
          .getState()
          .createSheet(project.name, local);
        if (pageKeyRef.current !== actionPageKey) return false;
        removeLocalSheet(config.sheet);
        config.sheet = createdSheet.name;
      }
    }
    return true;
  };

  const handleCreatePlan = async () => {
    if (createPlanBlockingReasons.length > 0 || !canCreateDraftReview) {
      return;
    }
    const actionPageKey = page.pageKey;
    try {
      setUpdating(true);
      if (!(await createSheets(actionPageKey))) return;
      const { plan } = await createPlanWithDraftReview({
        createIssue: (request) =>
          issueServiceClientConnect.createIssue(request),
        createPlan: (request) => planServiceClientConnect.createPlan(request),
        creator: `users/${currentUser.email}`,
        labels: page.creationIssueLabels,
        parent: project.name,
        plan: page.plan,
      });
      if (pageKeyRef.current !== actionPageKey) return;
      page.bypassLeaveGuardOnce();
      await router.replace({
        name: PROJECT_V1_ROUTE_PLAN_DETAIL,
        params: {
          projectId: extractProjectResourceName(plan.name),
          planId: extractPlanUID(plan.name),
        },
      });
    } catch (error) {
      if (pageKeyRef.current !== actionPageKey) return;
      if (error instanceof DraftReviewIssueCreationError) {
        page.bypassLeaveGuardOnce();
        await router.replace({
          name: PROJECT_V1_ROUTE_PLAN_DETAIL,
          params: {
            projectId: extractProjectResourceName(error.plan.name),
            planId: extractPlanUID(error.plan.name),
          },
        });
      }
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(
          error instanceof DraftReviewIssueCreationError ? error.cause : error
        ),
      });
    } finally {
      if (pageKeyRef.current === actionPageKey) {
        setUpdating(false);
      }
    }
  };

  const createIssueBlockingErrors = useMemo(
    () =>
      getCreateIssueBlockingErrors({
        emptySpecCount: emptySpecIdSet.size,
        plan: page.plan,
        project,
        t,
      }),
    [emptySpecIdSet.size, page.plan, project, t]
  );
  const createDisabledReason =
    createPermissionReason ?? createPlanBlockingReasons[0];
  const showChecksWarning = useMemo(
    () => hasChecksWarning(page.plan),
    [page.plan]
  );
  const createIssueConfirmErrors = useMemo(
    () =>
      getCreateIssueConfirmErrors({
        blockingErrors: createIssueBlockingErrors,
        project,
        selectedLabelCount: selectedLabels.length,
        t,
      }),
    [createIssueBlockingErrors, project, selectedLabels.length, t]
  );

  const resetReviewPopoverDraft = () => {
    setSelectedLabels(page.issue?.labels ?? []);
    setChecksWarningAcknowledged(false);
  };

  const handleReviewPopoverOpenChange = (open: boolean) => {
    setShowReviewPopover(open);
    if (!open) {
      resetReviewPopoverDraft();
    }
  };

  const handleSubmitDraftReview = async () => {
    if (
      !page.issue?.draft ||
      !canUpdateIssue ||
      createIssueConfirmErrors.length > 0
    ) {
      return;
    }
    const actionPageKey = page.pageKey;
    try {
      setSubmittingReview(true);
      const submittedIssue = await submitDraftReview({
        issue: page.issue,
        labels: selectedLabels,
        updateIssue: (request) =>
          issueServiceClientConnect.updateIssue(request),
      });
      if (pageKeyRef.current !== actionPageKey) return;
      handleReviewPopoverOpenChange(false);
      patchState({ issue: submittedIssue });
      await page.refreshState();
      if (pageKeyRef.current !== actionPageKey) return;
      focusPlanPhase("review", page.expandPhase);
    } catch (error) {
      if (pageKeyRef.current !== actionPageKey) return;
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      if (pageKeyRef.current === actionPageKey) {
        setSubmittingReview(false);
      }
    }
  };

  return (
    <div className="px-2 py-2 sm:px-4">
      <div className="flex flex-row items-center justify-between gap-2">
        <div className="flex min-w-0 flex-1 items-center gap-x-2">
          {/* Terminal status (Closed / Deployed) sits at the far left, before
              the title — a state badge, not an action. */}
          <PlanLifecycleStamp state={lifecycle} />
          <input
            ref={titleInputRef}
            className={cn(
              "h-9 min-w-0 flex-1 bg-transparent text-xl! font-bold text-main outline-hidden",
              editingTitle
                ? "border border-control-border px-3"
                : "border border-transparent px-0",
              !allowTitleEdit && "cursor-default"
            )}
            disabled={!allowTitleEdit || updating}
            maxLength={200}
            onBlur={() => void saveTitle()}
            onChange={(e) => {
              setTitle(e.target.value);
              if (page.isCreating) {
                patchState({
                  plan: { ...page.plan, title: e.target.value },
                });
              }
            }}
            onFocus={() => {
              if (!allowTitleEdit) return;
              setEditingTitle(true);
              setEditing("title", true);
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.nativeEvent.isComposing) {
                e.currentTarget.blur();
              }
            }}
            placeholder={t("common.untitled")}
            value={title}
          />
        </div>

        <div className="flex shrink-0 items-center gap-x-2">
          {/* Lifecycle slot: one primary advance/status per state. Create and
              ready-for-review stay here (coupled to the title/create flow); all
              other states render through PlanLifecycleSlot. */}
          {lifecycle.kind === "create" ? (
            <Button
              disabled={
                updating ||
                !canCreateDraftReview ||
                createPlanBlockingReasons.length > 0
              }
              onClick={() => void handleCreatePlan()}
              title={createDisabledReason}
            >
              {updating && <Loader2 className="mr-2 size-4 animate-spin" />}
              {t("common.create")}
            </Button>
          ) : lifecycle.kind === "ready-for-review" ? (
            <Popover
              open={showReviewPopover}
              onOpenChange={handleReviewPopoverOpenChange}
            >
              <PopoverTrigger
                render={
                  <Button
                    disabled={submitDisabled}
                    title={submitDisabledReason}
                  />
                }
              >
                {t("plan.ready-for-review")}
              </PopoverTrigger>
              <PopoverContent
                align="end"
                className="w-[min(28rem,calc(100vw-2rem))] px-4 py-4"
              >
                <ReadyForReviewPopoverContent
                  checksWarningAcknowledged={checksWarningAcknowledged}
                  confirmErrors={createIssueConfirmErrors}
                  forceIssueLabels={project.forceIssueLabels}
                  issueLabels={project.issueLabels ?? []}
                  onCancel={() => handleReviewPopoverOpenChange(false)}
                  onChecksWarningAcknowledgedChange={
                    setChecksWarningAcknowledged
                  }
                  onConfirm={() => void handleSubmitDraftReview()}
                  onSelectedLabelsChange={setSelectedLabels}
                  selectedLabels={selectedLabels}
                  showChecksWarning={showChecksWarning}
                  submitting={submittingReview}
                />
              </PopoverContent>
            </Popover>
          ) : (
            <PlanLifecycleSlot state={lifecycle} />
          )}
          {/* Secondary actions trail the lifecycle slot in a "..." overflow menu
              and never compete with it for the primary position — except when the
              slot has no primary, where the first one (e.g. Reopen) surfaces
              directly. */}
          {promotedAction && (
            <Button onClick={promotedAction.onSelect} appearance="outline">
              {promotedAction.label}
            </Button>
          )}
          {overflowActions.length > 0 && (
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button
                    aria-label={t("common.more")}
                    className="px-2"
                    appearance="secondary"
                  />
                }
              >
                <EllipsisVertical className="size-4" />
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                {overflowActions.map((action) => (
                  <DropdownMenuItem key={action.key} onClick={action.onSelect}>
                    {action.label}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      </div>
    </div>
  );
}

function ReadyForReviewPopoverContent({
  checksWarningAcknowledged,
  confirmErrors,
  forceIssueLabels,
  issueLabels,
  onCancel,
  onChecksWarningAcknowledgedChange,
  onConfirm,
  onSelectedLabelsChange,
  selectedLabels,
  showChecksWarning,
  submitting,
}: {
  checksWarningAcknowledged: boolean;
  confirmErrors: string[];
  forceIssueLabels: boolean;
  issueLabels: IssueLabel[];
  onCancel: () => void;
  onChecksWarningAcknowledgedChange?: (checked: boolean) => void;
  onConfirm: () => void;
  onSelectedLabelsChange: (labels: string[]) => void;
  selectedLabels: string[];
  showChecksWarning: boolean;
  submitting: boolean;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-y-4">
      {showChecksWarning && (
        <Alert variant="warning" description={t("issue.checks-warning-hint")} />
      )}
      <IssueLabelSelect
        labels={issueLabels}
        onChange={onSelectedLabelsChange}
        required={forceIssueLabels}
        selected={selectedLabels}
      />
      {showChecksWarning && (
        <label className="flex items-center gap-x-2 text-sm text-control">
          <Checkbox
            checked={checksWarningAcknowledged}
            onCheckedChange={(checked) =>
              onChecksWarningAcknowledgedChange?.(checked)
            }
          />
          <span>
            {t("issue.action-anyway", {
              action: t("common.confirm"),
            })}
          </span>
        </label>
      )}
      {confirmErrors.length > 0 && (
        <Alert
          variant="error"
          description={
            <span className="whitespace-pre-line">
              {confirmErrors.join("\n")}
            </span>
          }
        />
      )}
      <div className="flex justify-start gap-x-2 pt-1">
        <Button
          disabled={
            confirmErrors.length > 0 ||
            (showChecksWarning && !checksWarningAcknowledged) ||
            submitting
          }
          onClick={onConfirm}
          size="sm"
        >
          {submitting && <Loader2 className="size-4 animate-spin" />}
          {t("common.confirm")}
        </Button>
        <Button onClick={onCancel} size="sm" appearance="secondary">
          {t("common.cancel")}
        </Button>
      </div>
    </div>
  );
}
