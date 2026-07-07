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
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
} from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  BatchUpdateIssuesStatusRequestSchema,
  CreateIssueRequestSchema,
  Issue_Type,
  IssueSchema,
  IssueStatus,
  ListIssueCommentsRequestSchema,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  CreatePlanRequestSchema,
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  extractIssueUID,
  extractPlanUID,
  extractProjectResourceName,
  extractSheetUID,
  hasProjectPermissionV2,
} from "@/utils";
import { usePlanDetailSpecValidation } from "../hooks/usePlanDetailSpecValidation";
import { focusPlanPhase } from "../shell/focusPhase";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import {
  getCreateIssueBlockingErrors,
  getCreateIssueConfirmErrors,
  getCreatePlanBlockingReasons,
  hasChecksWarning,
  shouldStayOnPlanDetailPage,
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
  const [title, setTitle] = useState(page.plan.title);
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
  const createButtonRef = useRef<HTMLButtonElement>(null);
  const [showCreateErrors, setShowCreateErrors] = useState(false);
  const createPlanBlockingReasons = useMemo(
    () =>
      getCreatePlanBlockingReasons({
        title: page.plan.title,
        emptySpecCount: emptySpecIdSet.size,
        t,
      }),
    [emptySpecIdSet.size, page.plan.title, t]
  );
  const titleMissing = page.plan.title.trim() === "";

  useEffect(() => {
    const nextTitle = page.issue?.title ?? page.plan.title;
    setTitle((prev) => (prev === nextTitle ? prev : nextTitle));
  }, [page.issue?.title, page.plan.title]);

  const allowTitleEdit = useMemo(() => {
    if (page.readonly) return false;
    if (page.isCreating) return true;
    if (!page.issue && page.plan.hasRollout) return false;
    if (page.issue) {
      return hasProjectPermissionV2(project, "bb.issues.update");
    }
    return (
      page.plan.creator === currentUser.name ||
      hasProjectPermissionV2(project, "bb.plans.update")
    );
  }, [
    currentUser.name,
    page.isCreating,
    page.issue,
    page.plan.creator,
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
    !page.plan.issue &&
    !page.plan.hasRollout &&
    page.plan.state === State.ACTIVE;
  const showReopenPlan =
    !page.isCreating &&
    !page.plan.issue &&
    !page.plan.hasRollout &&
    page.plan.state === State.DELETED;

  // Close (cancel) / reopen the review issue from the header, mirroring the issue
  // detail page. Close is offered during the review phase (open issue, no rollout
  // yet); reopen once the review was canceled.
  const canUpdateIssue = hasProjectPermissionV2(project, "bb.issues.update");
  const showCloseIssue =
    !!page.issue &&
    page.issue.status === IssueStatus.OPEN &&
    !page.plan.hasRollout &&
    canUpdateIssue;
  const showReopenIssue =
    !!page.issue &&
    page.issue.status === IssueStatus.CANCELED &&
    canUpdateIssue;

  const submitDisabled = page.isEditing;
  const submitDisabledReason = submitDisabled
    ? t("plan.editor.save-changes-before-continuing")
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
    const currentTitle = page.issue?.title ?? page.plan.title;
    if (trimmed === currentTitle) {
      setTitle(currentTitle);
      setEditingTitle(false);
      setEditing("title", false);
      return;
    }

    try {
      setUpdating(true);
      if (page.issue) {
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
        patchState({ plan: response });
      }
    } catch (error) {
      setTitle(page.issue?.title ?? page.plan.title);
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: String(error),
      });
    } finally {
      setUpdating(false);
      setEditingTitle(false);
      setEditing("title", false);
    }
  };

  const updatePlanState = async (state: State) => {
    try {
      const planPatch = clone(PlanSchema, page.plan);
      planPatch.state = state;
      const updated = await planServiceClientConnect.updatePlan(
        create(UpdatePlanRequestSchema, {
          plan: planPatch,
          updateMask: { paths: ["state"] },
        })
      );
      patchState({ plan: updated });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch (error) {
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
    try {
      await issueServiceClientConnect.batchUpdateIssuesStatus(
        create(BatchUpdateIssuesStatusRequestSchema, {
          parent: project.name,
          issues: [issue.name],
          status,
        })
      );
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
      // Land on the review section so the close/reopen system comment and the
      // updated status are visible (consistent with the other header advances).
      focusPlanPhase("review", page.expandPhase);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch (error) {
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

  const createSheets = async () => {
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
        removeLocalSheet(config.sheet);
        config.sheet = createdSheet.name;
      }
    }
  };

  const handleCreatePlan = async () => {
    if (createPlanBlockingReasons.length > 0) {
      setShowCreateErrors(true);
      return;
    }
    try {
      setUpdating(true);
      await createSheets();
      const createdPlan = await planServiceClientConnect.createPlan(
        create(CreatePlanRequestSchema, {
          parent: project.name,
          plan: page.plan,
        })
      );
      page.bypassLeaveGuardOnce();
      void router.replace({
        name: PROJECT_V1_ROUTE_PLAN_DETAIL,
        params: {
          projectId: extractProjectResourceName(createdPlan.name),
          planId: extractPlanUID(createdPlan.name),
        },
      });
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setUpdating(false);
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
    setSelectedLabels([]);
    setChecksWarningAcknowledged(false);
  };

  const handleReviewPopoverOpenChange = (open: boolean) => {
    setShowReviewPopover(open);
    if (!open) {
      resetReviewPopoverDraft();
    }
  };

  const handleCreateIssue = async () => {
    if (createIssueConfirmErrors.length > 0) return;
    try {
      setSubmittingReview(true);
      const createdIssue = await issueServiceClientConnect.createIssue(
        create(CreateIssueRequestSchema, {
          parent: project.name,
          issue: create(IssueSchema, {
            creator: `users/${currentUser.email}`,
            labels: selectedLabels,
            plan: page.plan.name,
            status: IssueStatus.OPEN,
            type: Issue_Type.DATABASE_CHANGE,
          }),
        })
      );
      handleReviewPopoverOpenChange(false);
      patchState({
        issue: createdIssue,
        plan: {
          ...page.plan,
          issue: createdIssue.name,
        },
      });
      if (shouldStayOnPlanDetailPage(page.plan)) {
        await page.refreshState();
        // Land the user on the review they just opened.
        focusPlanPhase("review", page.expandPhase);
        return;
      }
      void router.push({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        params: {
          projectId: extractProjectResourceName(page.plan.name),
          issueId: extractIssueUID(createdIssue.name),
        },
      });
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setSubmittingReview(false);
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
            <Popover
              modal={false}
              onOpenChange={setShowCreateErrors}
              open={showCreateErrors && createPlanBlockingReasons.length > 0}
            >
              <Button
                disabled={updating}
                onClick={() => void handleCreatePlan()}
                ref={createButtonRef}
              >
                {updating && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                {t("common.create")}
              </Button>
              <PopoverContent
                anchor={createButtonRef}
                className="max-w-xs overflow-hidden border-error/40 p-0"
                initialFocus={titleMissing ? titleInputRef : false}
              >
                <Alert
                  className="rounded-none border-0 shadow-none"
                  description={
                    <ul className="list-disc pl-4">
                      {createPlanBlockingReasons.map((reason) => (
                        <li key={reason}>{reason}</li>
                      ))}
                    </ul>
                  }
                  title={t("plan.cannot-create")}
                  variant="error"
                />
              </PopoverContent>
            </Popover>
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
                  onConfirm={() => void handleCreateIssue()}
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
  onChecksWarningAcknowledgedChange: (checked: boolean) => void;
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
        <div className="rounded-sm border border-warning/30 bg-warning/10 px-3 py-2 text-sm text-warning">
          {t("issue.checks-warning-hint")}
        </div>
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
              onChecksWarningAcknowledgedChange(checked)
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
        <div className="whitespace-pre-line rounded-sm border border-error/30 bg-error/5 px-3 py-2 text-sm text-error">
          {confirmErrors.join("\n")}
        </div>
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
          {submitting && <Loader2 className="h-4 w-4 animate-spin" />}
          {t("common.confirm")}
        </Button>
        <Button onClick={onCancel} size="sm" appearance="secondary">
          {t("common.cancel")}
        </Button>
      </div>
    </div>
  );
}
