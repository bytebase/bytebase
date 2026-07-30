import { clone, create } from "@bufbuild/protobuf";
import { EllipsisVertical } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect, planServiceClientConnect } from "@/api";
import { router } from "@/app/router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/app/router/handles";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  createPlanWithDraftReview,
  DraftReviewIssueCreationError,
  submitDraftReview,
} from "@/lib/plan/workflow";
import { invalidateProjectPlansPagedDataCache } from "@/lib/projectPagedDataCache";
import { cn } from "@/lib/utils";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { IssueStatus, State } from "@/types/proto-es/v1/common_pb";
import {
  BatchUpdateIssuesStatusRequestSchema,
  IssueSchema,
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
import { getLocalSheetByName, removeLocalSheet } from "../utils/localSheet";
import {
  getCreatePlanBlockers,
  getSubmitReviewAdvance,
  NO_ADVANCE,
  NO_BLOCKERS,
} from "./lifecycle/advanceState";
import {
  LifecycleAdvanceButton,
  type LifecycleAdvanceProps,
} from "./lifecycle/LifecycleAdvance";
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
  const [submittingReview, setSubmittingReview] = useState(false);
  const { emptySpecIdSet } = usePlanDetailSpecValidation(page.plan.specs ?? []);
  const titleInputRef = useRef<HTMLInputElement>(null);
  const titleAutoFocusedRef = useRef(false);
  const pageKeyRef = useRef(page.pageKey);
  pageKeyRef.current = page.pageKey;
  const missingCreatePermissions = (
    ["bb.plans.create", "bb.issues.create"] as const
  ).filter((permission) => !hasProjectPermissionV2(project, permission));
  const createPermissionReason = missingCreatePermissions.length
    ? t("common.missing-required-permission", {
        permissions: missingCreatePermissions.join(", "),
      })
    : undefined;

  const canUpdatePlan =
    page.plan.creator === currentUser.name ||
    hasProjectPermissionV2(project, "bb.plans.update");

  useEffect(() => {
    setTitle(persistedTitle);
    setEditingTitle(false);
    setUpdating(false);
    setSubmittingReview(false);
    titleAutoFocusedRef.current = false;
    setEditing("title", false);
  }, [page.pageKey]);

  useEffect(() => {
    if (!editingTitle) {
      setTitle((prev) => (prev === persistedTitle ? prev : persistedTitle));
    }
  }, [editingTitle, persistedTitle]);

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
      // This mutation changes the Plan List's Issue-derived review badge but
      // not the Plan resource. Invalidate immediately instead of relying on a
      // subsequent detail refresh observing an Issue update-time change.
      invalidateProjectPlansPagedDataCache(page.projectId);
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

  // Everything standing between the reader and the next lifecycle state, as
  // data. A missing permission is one more entry rather than a separate header
  // state, so the action never disappears or goes dead.
  // Only the state that renders the advance resolves it: in create mode every
  // title keystroke hands out a fresh `page.plan`, and the submit resolver walks
  // the specs for a result create mode never reads.
  const isCreating = lifecycle.kind === "create";
  const isSubmitting = lifecycle.kind === "ready-for-review";
  // Labels come from the metadata row, which persists them on change — the
  // submit path only counts them, so key the memo on the count, not the array.
  const selectedLabelCount = page.issue?.labels?.length ?? 0;

  const createBlockers = useMemo(
    () =>
      isCreating
        ? getCreatePlanBlockers({
            emptySpecCount: emptySpecIdSet.size,
            permissionReason: createPermissionReason,
            title: page.plan.title,
            t,
          })
        : NO_BLOCKERS,
    [
      createPermissionReason,
      emptySpecIdSet.size,
      isCreating,
      page.plan.title,
      t,
    ]
  );

  const submitAdvance = useMemo(
    () =>
      isSubmitting
        ? getSubmitReviewAdvance({
            emptySpecCount: emptySpecIdSet.size,
            isEditing: page.isEditing,
            permissionReason: canUpdateIssue
              ? undefined
              : t("plan.draft-update-permission-required"),
            plan: page.plan,
            project,
            selectedLabelCount,
            t,
          })
        : NO_ADVANCE,
    [
      canUpdateIssue,
      emptySpecIdSet.size,
      isSubmitting,
      page.isEditing,
      page.plan,
      project,
      selectedLabelCount,
      t,
    ]
  );

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
    if (createBlockers.length > 0) {
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

  const handleSubmitDraftReview = async () => {
    if (!page.issue?.draft || submitAdvance.blockers.length > 0) {
      return;
    }
    const actionPageKey = page.pageKey;
    try {
      setSubmittingReview(true);
      const submittedIssue = await submitDraftReview({
        issue: page.issue,
        updateIssue: (request) =>
          issueServiceClientConnect.updateIssue(request),
      });
      if (pageKeyRef.current !== actionPageKey) return;
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

  let advance: LifecycleAdvanceProps | undefined;
  if (isCreating) {
    advance = {
      blockers: createBlockers,
      busy: updating,
      heading: t("plan.cannot-create"),
      onAdvance: () => void handleCreatePlan(),
      // The empty title is the one blocker whose field is already in this row.
      onBlocked: (blockers) => {
        if (blockers.some((blocker) => blocker.id === "title")) {
          titleInputRef.current?.focus();
        }
      },
      verb: t("common.create"),
    };
  } else if (isSubmitting) {
    advance = {
      blockers: submitAdvance.blockers,
      busy: submittingReview,
      decision: submitAdvance.decision,
      heading: t("plan.not-ready-for-review"),
      onAdvance: () => void handleSubmitDraftReview(),
      verb: t("plan.ready-for-review"),
    };
  }

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
          {advance ? (
            // Keyed on the plan so a surface opened for one plan cannot greet
            // the reader on the next.
            <LifecycleAdvanceButton key={page.pageKey} {...advance} />
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
