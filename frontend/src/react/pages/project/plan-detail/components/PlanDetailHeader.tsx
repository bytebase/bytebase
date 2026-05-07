import { clone, create } from "@bufbuild/protobuf";
import { Ban, ChevronUp, Loader2, Menu, Plus } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect, planServiceClientConnect } from "@/connect";
import {
  type IssueLabel,
  IssueLabelSelect,
} from "@/react/components/IssueLabelSelect";
import { MarkdownEditor } from "@/react/components/MarkdownEditor";
import { Button } from "@/react/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
} from "@/router/dashboard/projectV1";
import { pushNotification, useSheetV1Store } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  CreateIssueRequestSchema,
  Issue_Type,
  IssueSchema,
  IssueStatus,
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
import { usePlanDetailContext } from "../context/PlanDetailContext";
import { usePlanDetailSpecValidation } from "../hooks/usePlanDetailSpecValidation";
import {
  getCreateIssueBlockingErrors,
  getCreateIssueConfirmErrors,
  hasChecksWarning,
  shouldStayOnPlanDetailPage,
} from "../utils/header";
import { getLocalSheetByName, removeLocalSheet } from "../utils/localSheet";

export function PlanDetailHeader() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { patchState, setEditing } = page;
  const sheetStore = useSheetV1Store();
  const currentUser = page.currentUser;
  const project = page.project;
  const [title, setTitle] = useState(page.plan.title);
  const [description, setDescription] = useState(page.plan.description);
  const [editingTitle, setEditingTitle] = useState(false);
  const [editingDescription, setEditingDescription] = useState(false);
  const [showFullDescription, setShowFullDescription] = useState(false);
  const [updating, setUpdating] = useState(false);
  const [showReviewPopover, setShowReviewPopover] = useState(false);
  const [selectedLabels, setSelectedLabels] = useState<string[]>([]);
  const [checksWarningAcknowledged, setChecksWarningAcknowledged] =
    useState(false);
  const [submittingReview, setSubmittingReview] = useState(false);
  const { emptySpecIdSet } = usePlanDetailSpecValidation(page.plan.specs ?? []);

  useEffect(() => {
    const nextTitle = page.issue?.title ?? page.plan.title;
    setTitle((prev) => (prev === nextTitle ? prev : nextTitle));
  }, [page.issue?.title, page.plan.title]);

  useEffect(() => {
    setDescription((prev) =>
      prev === page.plan.description ? prev : page.plan.description
    );
  }, [page.plan.description]);

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

  const allowDescriptionEdit = useMemo(() => {
    if (page.readonly) return false;
    if (page.isCreating) return true;
    if (!page.issue && page.plan.hasRollout) return false;
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

  const showSubmitForReview =
    !!page.plan.name &&
    !page.isCreating &&
    !page.plan.issue &&
    page.plan.state === State.ACTIVE;
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

  const saveDescription = async () => {
    if (page.isCreating) {
      patchState({
        plan: {
          ...page.plan,
          description,
        },
      });
      setEditingDescription(false);
      setEditing("description", false);
      return;
    }

    let saved = false;
    try {
      setUpdating(true);
      const planPatch = create(PlanSchema, {
        ...page.plan,
        description,
      });
      const response = await planServiceClientConnect.updatePlan(
        create(UpdatePlanRequestSchema, {
          plan: planPatch,
          updateMask: { paths: ["description"] },
        })
      );
      patchState({ plan: response });
      saved = true;
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: String(error),
      });
    } finally {
      setUpdating(false);
      if (saved) {
        setEditingDescription(false);
        setEditing("description", false);
      }
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
        const createdSheet = await sheetStore.createSheet(project.name, local);
        removeLocalSheet(config.sheet);
        config.sheet = createdSheet.name;
      }
    }
  };

  const handleCreatePlan = async () => {
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
        checksWarningAcknowledged,
        project,
        selectedLabelCount: selectedLabels.length,
        showChecksWarning,
        t,
      }),
    [
      checksWarningAcknowledged,
      createIssueBlockingErrors,
      project,
      selectedLabels.length,
      showChecksWarning,
      t,
    ]
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

  const isDescriptionLong =
    (description?.length ?? 0) > 150 ||
    (description?.split("\n").length ?? 0) > 3;

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
            type: page.plan.specs.some(
              (spec) => spec.config?.case === "exportDataConfig"
            )
              ? Issue_Type.DATABASE_EXPORT
              : Issue_Type.DATABASE_CHANGE,
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
        {page.plan.state === State.DELETED && (
          <span className="inline-flex shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 text-sm text-control">
            <Ban className="h-4 w-4" />
            {t("common.closed")}
          </span>
        )}
        <div className="min-w-0 flex-1">
          <input
            className={cn(
              "h-9 w-full bg-transparent text-[18px] font-bold text-main outline-hidden",
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
          {page.isCreating ? (
            <Button
              disabled={
                updating || !page.plan.title.trim() || emptySpecIdSet.size > 0
              }
              onClick={() => void handleCreatePlan()}
            >
              {updating && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {t("common.create")}
            </Button>
          ) : (
            <>
              {showSubmitForReview && (
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
              )}
              {showClosePlan && (
                <Button
                  onClick={() => {
                    if (window.confirm(t("plan.state.close-confirm"))) {
                      void updatePlanState(State.DELETED);
                    }
                  }}
                  variant="outline"
                >
                  {t("common.close")}
                </Button>
              )}
              {showReopenPlan && (
                <Button
                  onClick={() => {
                    if (window.confirm(t("plan.state.reopen-confirm"))) {
                      void updatePlanState(State.ACTIVE);
                    }
                  }}
                  variant="outline"
                >
                  {t("common.reopen")}
                </Button>
              )}
            </>
          )}
          {page.sidebarMode === "MOBILE" && (
            <Button
              onClick={() => page.setMobileSidebarOpen(true)}
              size="icon"
              variant="ghost"
            >
              <Menu className="h-5 w-5" />
            </Button>
          )}
        </div>
      </div>

      <div className="min-w-0">
        {editingDescription ? (
          <div className="py-2">
            <div className="flex items-center justify-between">
              <span className="text-base font-medium">
                {t("common.description")}
              </span>
              <div className="flex items-center gap-2">
                {!page.isCreating ? (
                  <Button
                    onClick={() => void saveDescription()}
                    size="xs"
                    variant="outline"
                  >
                    {t("common.save")}
                  </Button>
                ) : (
                  <Button
                    onClick={() => {
                      setDescription(page.plan.description);
                      setEditingDescription(false);
                      setEditing("description", false);
                    }}
                    size="xs"
                    variant="ghost"
                  >
                    <ChevronUp className="mr-1 h-4 w-4" />
                    {t("common.collapse")}
                  </Button>
                )}
                {!page.isCreating && (
                  <Button
                    onClick={() => {
                      setDescription(page.plan.description);
                      setEditingDescription(false);
                      setEditing("description", false);
                    }}
                    size="xs"
                    variant="ghost"
                  >
                    {t("common.cancel")}
                  </Button>
                )}
              </div>
            </div>
            <textarea
              className="mt-2 min-h-28 w-full rounded-sm border border-control-border px-3 py-2 text-sm outline-hidden"
              onChange={(e) => {
                setDescription(e.target.value);
                if (page.isCreating) {
                  patchState({
                    plan: { ...page.plan, description: e.target.value },
                  });
                }
              }}
              value={description}
            />
          </div>
        ) : description ? (
          <div className="mt-1">
            <div
              aria-disabled={!allowDescriptionEdit}
              className={cn(
                "relative w-full rounded-md border border-transparent px-2 py-1 text-left text-sm text-control-light transition-all duration-200",
                !showFullDescription && "max-h-[4.5rem] overflow-hidden",
                allowDescriptionEdit && "cursor-pointer hover:border-gray-200"
              )}
              onClick={() => {
                if (!allowDescriptionEdit) return;
                setEditingDescription(true);
                setEditing("description", true);
              }}
              onKeyDown={(event) => {
                if (!allowDescriptionEdit) return;
                if (event.key !== "Enter" && event.key !== " ") return;
                event.preventDefault();
                setEditingDescription(true);
                setEditing("description", true);
              }}
              role={allowDescriptionEdit ? "button" : undefined}
              tabIndex={allowDescriptionEdit ? 0 : undefined}
            >
              <div className="pointer-events-none">
                <MarkdownEditor content={description} mode="preview" />
              </div>
              {!showFullDescription && isDescriptionLong && (
                <div className="pointer-events-none absolute bottom-0 left-0 right-0 h-6 bg-gradient-to-t from-white to-transparent" />
              )}
            </div>
            {isDescriptionLong && (
              <button
                className="mt-1 px-2 text-xs text-control-placeholder hover:text-control"
                onClick={(event) => {
                  event.stopPropagation();
                  setShowFullDescription((value) => !value);
                }}
                type="button"
              >
                {showFullDescription
                  ? t("common.show-less")
                  : t("common.show-more")}
              </button>
            )}
          </div>
        ) : allowDescriptionEdit ? (
          <Button
            onClick={() => {
              setEditingDescription(true);
              setEditing("description", true);
            }}
            size="xs"
            variant="ghost"
            className="italic opacity-60"
          >
            <Plus className="mr-1 h-4 w-4" />
            {t("plan.description.placeholder")}
          </Button>
        ) : null}
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
          <input
            checked={checksWarningAcknowledged}
            className="accent-accent"
            onChange={(event) =>
              onChecksWarningAcknowledgedChange(event.target.checked)
            }
            type="checkbox"
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
          disabled={confirmErrors.length > 0 || submitting}
          onClick={onConfirm}
          size="sm"
        >
          {submitting && <Loader2 className="h-4 w-4 animate-spin" />}
          {t("common.confirm")}
        </Button>
        <Button onClick={onCancel} size="sm" variant="ghost">
          {t("common.cancel")}
        </Button>
      </div>
    </div>
  );
}
