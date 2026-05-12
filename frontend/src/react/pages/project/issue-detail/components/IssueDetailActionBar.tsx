import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { first, orderBy } from "lodash-es";
import {
  CalendarX,
  Check,
  ChevronDown,
  EllipsisVertical,
  ExternalLink,
  Loader2,
  MessageCircle,
  X,
} from "lucide-react";
import {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import {
  issueServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/connect";
import { MarkdownEditor } from "@/react/components/MarkdownEditor";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  buildPlanDeployRouteFromPlanName,
  buildPlanDeployRouteFromRolloutName,
} from "@/router/dashboard/projectV1RouteHelpers";
import {
  pushNotification,
  useCurrentUserV1,
  useIssueCommentStore,
  useProjectV1Store,
  useSQLStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  ApproveIssueRequestSchema,
  BatchUpdateIssuesStatusRequestSchema,
  IssueStatus,
  ListIssueCommentsRequestSchema,
  RejectIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  BatchRunTasksRequestSchema,
  CreateRolloutRequestSchema,
  type Rollout,
  TaskRun_ExportArchiveStatus,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  Advice_Level,
  ExportRequestSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import {
  extractPlanUID,
  extractProjectResourceName,
  extractTaskRunUID,
  extractTaskUID,
} from "@/utils";
import { useIssueDetailContext } from "../context/IssueDetailContext";
import { useIssueDetailSpecValidation } from "../hooks/useIssueDetailSpecValidation";
import {
  type ActionContext,
  type ActionDefinition,
  buildIssueDetailActionContext,
  createIssueDetailActions,
  type UnifiedAction,
} from "../utils/actionRegistry";
import { isApprovalCompleted } from "../utils/approval";
import { refreshIssueDetailState } from "../utils/refreshIssueDetailState";
import { IssueDetailTaskRolloutActionPanel } from "./IssueDetailTaskRolloutActionPanel";

export function IssueDetailActionBar() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const projectStore = useProjectV1Store();
  const issueCommentStore = useIssueCommentStore();
  const sqlStore = useSQLStore();
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const [pendingConfirmAction, setPendingConfirmAction] =
    useState<ActionDefinition>();
  const [pendingReviewOpen, setPendingReviewOpen] = useState(false);
  const [pendingRolloutAction, setPendingRolloutAction] = useState<
    "ROLLOUT_START" | "ROLLOUT_CANCEL" | undefined
  >();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const [menuOpen, setMenuOpen] = useState(false);
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));
  const { isSpecEmpty } = useIssueDetailSpecValidation(page.plan?.specs ?? []);

  useClickOutside(menuRef, menuOpen, () => setMenuOpen(false));

  const context = useMemo<ActionContext | undefined>(() => {
    if (!page.plan || !project || !currentUser) {
      return undefined;
    }
    const statusCount = page.plan.planCheckRunStatusCount ?? {};
    const planCheckStatus =
      statusCount.ERROR > 0 || statusCount.FAILED > 0
        ? Advice_Level.ERROR
        : statusCount.WARNING > 0
          ? Advice_Level.WARNING
          : Advice_Level.SUCCESS;

    return buildIssueDetailActionContext({
      plan: page.plan,
      issue: page.issue,
      rollout: page.rollout,
      project,
      currentUser,
      taskRuns: page.taskRuns,
      isCreating: page.isCreating,
      planCheckStatus,
      hasRunningPlanChecks: (statusCount.RUNNING ?? 0) > 0,
      isSpecEmpty,
    });
  }, [
    currentUser,
    page.isCreating,
    page.issue,
    page.plan,
    page.rollout,
    page.taskRuns,
    project,
    isSpecEmpty,
  ]);

  const globalDisabledReason = useMemo(() => {
    return page.isEditing
      ? t("plan.editor.save-changes-before-continuing")
      : undefined;
  }, [page.isEditing, t]);
  const issueDetailActions = useMemo(() => createIssueDetailActions(t), [t]);

  const getCategory = useCallback(
    (action: ActionDefinition) => {
      if (!context) {
        return "secondary";
      }
      return typeof action.category === "function"
        ? action.category(context)
        : action.category;
    },
    [context]
  );

  const visibleActions = useMemo(() => {
    if (!context || page.isCreating) {
      return [];
    }
    return issueDetailActions.filter((action) => action.isVisible(context));
  }, [context, page.isCreating]);

  const primaryAction = useMemo(() => {
    return visibleActions.find((action) => getCategory(action) === "primary");
  }, [getCategory, visibleActions]);

  const secondaryActions = useMemo(() => {
    return visibleActions.filter(
      (action) =>
        getCategory(action) === "secondary" ||
        (getCategory(action) === "primary" && action.id !== primaryAction?.id)
    );
  }, [getCategory, primaryAction?.id, visibleActions]);

  const shouldShowPlanLink = useMemo(() => {
    if (!page.plan || !page.issue) {
      return false;
    }
    if (page.issueType !== "DATABASE_CHANGE") {
      return false;
    }
    return page.plan.hasRollout || isApprovalCompleted(page.issue);
  }, [page.issue, page.issueType, page.plan]);

  const planRoute = useMemo(() => {
    if (!page.plan) {
      return undefined;
    }
    if (page.plan.hasRollout) {
      return buildPlanDeployRouteFromPlanName(page.plan.name);
    }
    return {
      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
      params: {
        projectId: extractProjectResourceName(page.plan.name),
        planId: extractPlanUID(page.plan.name),
      },
    };
  }, [page.plan]);

  const isActionDisabled = useCallback(
    (action: ActionDefinition) => {
      if (!context) {
        return true;
      }
      return page.isEditing || action.isDisabled(context);
    },
    [context, page.isEditing]
  );

  const getDisabledReason = useCallback(
    (action: ActionDefinition) => {
      if (!context) {
        return undefined;
      }
      return globalDisabledReason || action.disabledReason(context);
    },
    [context, globalDisabledReason]
  );

  const refreshIssueComments = useCallback(async () => {
    if (!page.issue?.name) {
      return;
    }
    await issueCommentStore.listIssueComments(
      create(ListIssueCommentsRequestSchema, {
        parent: page.issue.name,
        pageSize: 1000,
      })
    );
  }, [issueCommentStore, page.issue?.name]);

  const handleRefreshIssueDetailState = useCallback(async () => {
    await refreshIssueDetailState(page);
  }, [page]);

  const handleExportDownload = useCallback(async () => {
    if (!page.rollout) {
      return;
    }
    try {
      setIsSubmitting(true);
      const content = await sqlStore.exportData(
        create(ExportRequestSchema, {
          name: `${page.rollout.name}/stages/-`,
        })
      );
      const buffer = content.buffer.slice(
        content.byteOffset,
        content.byteOffset + content.byteLength
      ) as ArrayBuffer;
      const blob = new Blob([buffer], {
        type: "application/zip",
      });
      const url = window.URL.createObjectURL(blob);
      const filename = `export-data-${dayjs(new Date()).format(
        "YYYY-MM-DDTHH-mm-ss"
      )}.zip`;
      const link = document.createElement("a");
      link.download = filename;
      link.href = url;
      link.click();
      await handleRefreshIssueDetailState();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setIsSubmitting(false);
    }
  }, [handleRefreshIssueDetailState, page.rollout, sqlStore, t]);

  const handleCreateRollout = useCallback(
    async (options?: { runAllTasks?: boolean }) => {
      if (!page.plan) {
        return;
      }
      const createdRollout = await rolloutServiceClientConnect.createRollout(
        create(CreateRolloutRequestSchema, {
          parent: page.plan.name,
        })
      );

      if (options?.runAllTasks) {
        for (const stage of createdRollout.stages) {
          await rolloutServiceClientConnect.batchRunTasks(
            create(BatchRunTasksRequestSchema, {
              parent: stage.name,
              tasks: stage.tasks.map((task) => task.name),
            })
          );
        }
      }

      await handleRefreshIssueDetailState();

      if (createdRollout.stages.length > 0) {
        void router.push(
          buildPlanDeployRouteFromRolloutName(createdRollout.name)
        );
      }
    },
    [handleRefreshIssueDetailState, page.plan]
  );

  const handleIssueStatusAction = useCallback(
    async (action: "ISSUE_STATUS_CLOSE" | "ISSUE_STATUS_REOPEN") => {
      if (!page.issue || !project) {
        return;
      }
      const nextStatus =
        action === "ISSUE_STATUS_CLOSE"
          ? IssueStatus.CANCELED
          : IssueStatus.OPEN;
      await issueServiceClientConnect.batchUpdateIssuesStatus(
        create(BatchUpdateIssuesStatusRequestSchema, {
          parent: project.name,
          issues: [page.issue.name],
          status: nextStatus,
        })
      );
      await Promise.all([
        handleRefreshIssueDetailState(),
        refreshIssueComments(),
      ]);
    },
    [handleRefreshIssueDetailState, page.issue, project, refreshIssueComments]
  );

  const confirmLabel =
    pendingConfirmAction && context ? pendingConfirmAction.label(context) : "";
  const confirmContent = pendingConfirmAction
    ? pendingConfirmAction.id === "ISSUE_STATUS_CLOSE"
      ? t("issue.status-transition.modal.close")
      : pendingConfirmAction.id === "ISSUE_STATUS_REOPEN"
        ? t("issue.status-transition.modal.reopen")
        : ""
    : "";
  const exportExpired =
    primaryAction?.id === "EXPORT_DOWNLOAD" &&
    isExportExpired(page.rollout, page.taskRuns);

  const executeAction = useCallback(
    async (action: UnifiedAction) => {
      if (!context) {
        return;
      }
      if (action === "EXPORT_DOWNLOAD") {
        await handleExportDownload();
        return;
      }
      if (action === "ISSUE_REVIEW") {
        setPendingReviewOpen(true);
        return;
      }
      if (action === "ISSUE_STATUS_CLOSE" || action === "ISSUE_STATUS_REOPEN") {
        const definition = visibleActions.find((item) => item.id === action);
        setPendingConfirmAction(definition);
        return;
      }
      if (action === "ROLLOUT_START") {
        if (context.hasDeferredRollout && !page.rollout) {
          try {
            setIsSubmitting(true);
            await handleCreateRollout({ runAllTasks: true });
          } catch (error) {
            pushNotification({
              module: "bytebase",
              style: "CRITICAL",
              title: t("common.failed"),
              description: String(error),
            });
          } finally {
            setIsSubmitting(false);
          }
          return;
        }
        setPendingRolloutAction("ROLLOUT_START");
        return;
      }
      if (action === "ROLLOUT_CANCEL") {
        setPendingRolloutAction("ROLLOUT_CANCEL");
      }
    },
    [
      context,
      handleCreateRollout,
      handleExportDownload,
      page.rollout,
      t,
      visibleActions,
    ]
  );

  const confirmAction = useCallback(async () => {
    if (!pendingConfirmAction) {
      return;
    }
    try {
      setIsSubmitting(true);
      if (
        pendingConfirmAction.id === "ISSUE_STATUS_CLOSE" ||
        pendingConfirmAction.id === "ISSUE_STATUS_REOPEN"
      ) {
        await handleIssueStatusAction(pendingConfirmAction.id);
      }
      setPendingConfirmAction(undefined);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setIsSubmitting(false);
    }
  }, [handleIssueStatusAction, pendingConfirmAction, t]);

  if (!context || !page.plan) {
    return null;
  }

  return (
    <>
      <div className="flex items-center gap-x-2">
        {shouldShowPlanLink && planRoute && (
          <Button
            className="gap-x-1"
            onClick={() => {
              void router.push(planRoute);
            }}
            variant="outline"
          >
            <span>#{extractPlanUID(page.plan.name)}</span>
            <span>{t("common.plan")}</span>
            <ExternalLink className="size-3.5" />
          </Button>
        )}

        {primaryAction &&
          (primaryAction.id === "ISSUE_REVIEW" ? (
            <IssueDetailReviewTrigger
              action={primaryAction}
              context={context}
              disabled={isSubmitting || isActionDisabled(primaryAction)}
              disabledReason={getDisabledReason(primaryAction)}
              loading={isSubmitting}
              onExecute={executeAction}
            >
              <IssueDetailReviewPopover
                context={context}
                mobile={page.sidebarMode === "MOBILE"}
                onOpenChange={setPendingReviewOpen}
                onRefreshIssueComments={refreshIssueComments}
                onRefreshState={handleRefreshIssueDetailState}
                open={pendingReviewOpen}
              />
            </IssueDetailReviewTrigger>
          ) : (
            <IssueDetailActionButton
              action={primaryAction}
              context={context}
              disabled={isSubmitting || isActionDisabled(primaryAction)}
              disabledReason={getDisabledReason(primaryAction)}
              exportExpired={exportExpired}
              loading={isSubmitting && primaryAction.id === "EXPORT_DOWNLOAD"}
              onExecute={executeAction}
            />
          ))}

        {secondaryActions.length > 0 && (
          <div className="relative" ref={menuRef}>
            <Tooltip
              content={
                primaryAction ? getDisabledReason(primaryAction) : undefined
              }
            >
              <span className="inline-flex">
                <Button
                  aria-label={t("common.more")}
                  className="px-1"
                  disabled={Boolean(
                    primaryAction && isActionDisabled(primaryAction)
                  )}
                  onClick={() => setMenuOpen((open) => !open)}
                  variant="ghost"
                >
                  <EllipsisVertical className="h-4 w-4" />
                </Button>
              </span>
            </Tooltip>
            {menuOpen && (
              <div
                className={cn(
                  "absolute right-0 top-full mt-1 min-w-44 overflow-hidden rounded-sm border border-control-border bg-white py-1 shadow-lg",
                  LAYER_SURFACE_CLASS
                )}
              >
                {secondaryActions.map((action) => {
                  const disabled = isActionDisabled(action) || isSubmitting;
                  return (
                    <Tooltip
                      key={action.id}
                      content={disabled ? getDisabledReason(action) : undefined}
                      side="left"
                    >
                      <span className="block">
                        <button
                          className={cn(
                            "flex w-full items-center px-3 py-2 text-left text-sm",
                            disabled
                              ? "cursor-not-allowed text-control-placeholder"
                              : "hover:bg-control-bg"
                          )}
                          disabled={disabled}
                          onClick={() => {
                            setMenuOpen(false);
                            void executeAction(action.id);
                          }}
                          type="button"
                        >
                          {action.label(context)}
                        </button>
                      </span>
                    </Tooltip>
                  );
                })}
              </div>
            )}
          </div>
        )}
      </div>

      <IssueDetailConfirmDialog
        busy={isSubmitting}
        content={confirmContent}
        label={confirmLabel}
        onConfirm={() => {
          void confirmAction();
        }}
        open={Boolean(pendingConfirmAction)}
        onOpenChange={(open) => {
          if (!open) {
            setPendingConfirmAction(undefined);
          }
        }}
      />

      <IssueDetailTaskRolloutActionPanel
        action={
          pendingRolloutAction === "ROLLOUT_START"
            ? "RUN"
            : pendingRolloutAction === "ROLLOUT_CANCEL"
              ? "CANCEL"
              : undefined
        }
        onConfirm={handleRefreshIssueDetailState}
        open={Boolean(pendingRolloutAction && page.rollout)}
        onOpenChange={(open) => {
          if (!open) {
            setPendingRolloutAction(undefined);
          }
        }}
        target={{ type: "tasks", stage: page.rollout?.stages[0] }}
      />
    </>
  );
}

function IssueDetailReviewTrigger({
  action,
  children,
  context,
  disabled,
  disabledReason,
  loading,
  onExecute,
}: {
  action: ActionDefinition;
  children: ReactNode;
  context: ActionContext;
  disabled: boolean;
  disabledReason?: string;
  loading?: boolean;
  onExecute: (action: UnifiedAction) => Promise<void>;
}) {
  const button = (
    <Button
      className="gap-x-1.5"
      disabled={disabled}
      onClick={() => {
        void onExecute(action.id);
      }}
      variant={action.buttonType === "default" ? "outline" : "default"}
    >
      {loading && <Loader2 className="h-4 w-4 animate-spin" />}
      <span>{action.label(context)}</span>
      <ChevronDown className="h-4 w-4" />
    </Button>
  );

  return (
    <div className="relative inline-flex">
      <Tooltip content={disabled ? disabledReason : undefined}>
        <span className="inline-flex">{button}</span>
      </Tooltip>
      {children}
    </div>
  );
}

function IssueDetailActionButton({
  action,
  context,
  disabled,
  disabledReason,
  exportExpired = false,
  loading = false,
  onExecute,
}: {
  action: ActionDefinition;
  context: ActionContext;
  disabled: boolean;
  disabledReason?: string;
  exportExpired?: boolean;
  loading?: boolean;
  onExecute: (action: UnifiedAction) => Promise<void>;
}) {
  const { t } = useTranslation();

  if (exportExpired) {
    return (
      <Tooltip content={t("issue.data-export.download-tooltip")}>
        <div className="flex items-center gap-2 text-sm textlabel leading-8">
          <CalendarX className="h-5 w-5" />
          {t("issue.data-export.file-expired")}
        </div>
      </Tooltip>
    );
  }

  const button = (
    <Button
      className={cn(
        action.buttonType === "success" &&
          "bg-success text-white hover:bg-success/90",
        action.id === "ISSUE_REVIEW" && "gap-x-1.5"
      )}
      disabled={disabled}
      onClick={() => {
        void onExecute(action.id);
      }}
      variant={action.buttonType === "default" ? "outline" : "default"}
    >
      {loading && <Loader2 className="h-4 w-4 animate-spin" />}
      <span>{action.label(context)}</span>
      {action.id === "ISSUE_REVIEW" && <ChevronDown className="h-4 w-4" />}
    </Button>
  );

  return (
    <Tooltip content={disabled ? disabledReason : undefined}>
      <span className="inline-flex">{button}</span>
    </Tooltip>
  );
}

function IssueDetailConfirmDialog({
  busy,
  content,
  label,
  onConfirm,
  open,
  onOpenChange,
}: {
  busy: boolean;
  content: string;
  label: string;
  onConfirm: () => void;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { t } = useTranslation();
  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-w-md p-6">
        {open && (
          <>
            <DialogTitle>{label}</DialogTitle>
            <div className="mt-3 text-sm text-control-light">{content}</div>
            <div className="mt-6 flex items-center justify-end gap-x-2">
              <Button onClick={() => onOpenChange(false)} variant="ghost">
                {t("common.cancel")}
              </Button>
              <Button disabled={busy} onClick={onConfirm}>
                {busy && <Loader2 className="h-4 w-4 animate-spin" />}
                {label}
              </Button>
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

function IssueDetailReviewPopover({
  context,
  mobile,
  onOpenChange,
  onRefreshIssueComments,
  onRefreshState,
  open,
}: {
  context: ActionContext;
  mobile: boolean;
  onOpenChange: (open: boolean) => void;
  onRefreshIssueComments: () => Promise<void>;
  onRefreshState: () => Promise<void>;
  open: boolean;
}) {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const issueCommentStore = useIssueCommentStore();
  const popoverRef = useRef<HTMLDivElement>(null);
  const [loading, setLoading] = useState(false);
  const [comment, setComment] = useState("");
  const [selectedAction, setSelectedAction] = useState<
    "COMMENT" | "APPROVE" | "REJECT"
  >("COMMENT");
  const issue = page.issue;
  const submitDisabled =
    loading || (selectedAction === "COMMENT" && comment.trim().length === 0);

  useEffect(() => {
    if (!open) {
      setComment("");
      setSelectedAction("COMMENT");
    }
  }, [open]);

  useClickOutside(popoverRef, open && !mobile, () => onOpenChange(false));

  const handleSubmit = useCallback(async () => {
    if (!issue) {
      return;
    }

    try {
      setLoading(true);
      if (selectedAction === "APPROVE") {
        const response = await issueServiceClientConnect.approveIssue(
          create(ApproveIssueRequestSchema, {
            comment,
            name: issue.name,
          })
        );
        page.patchState({ issue: response });
      } else if (selectedAction === "REJECT") {
        const response = await issueServiceClientConnect.rejectIssue(
          create(RejectIssueRequestSchema, {
            comment,
            name: issue.name,
          })
        );
        page.patchState({ issue: response });
      } else {
        await issueCommentStore.createIssueComment({
          issueName: issue.name,
          comment,
        });
      }

      await Promise.all([onRefreshState(), onRefreshIssueComments()]);
      onOpenChange(false);

      if (
        selectedAction === "APPROVE" &&
        page.plan &&
        !page.plan.specs.some(
          (spec) =>
            spec.config.case === "createDatabaseConfig" ||
            spec.config.case === "exportDataConfig"
        ) &&
        page.plan.hasRollout
      ) {
        void router.push(buildPlanDeployRouteFromPlanName(page.plan.name));
      } else if (selectedAction !== "COMMENT" && issue) {
        void router.push({
          name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
          params: {
            issueId: page.issueId,
            projectId: extractProjectResourceName(issue.name),
          },
          hash: "#issue-comment-editor",
        });
      }
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setLoading(false);
    }
  }, [
    comment,
    issue,
    issueCommentStore,
    onOpenChange,
    onRefreshIssueComments,
    onRefreshState,
    page.issueId,
    page.patchState,
    page.plan,
    selectedAction,
    t,
  ]);

  if (!open) {
    return null;
  }

  const content = (
    <div className="flex flex-col gap-y-3">
      <MarkdownEditor
        content={comment}
        onChange={setComment}
        onSubmit={() => {
          void handleSubmit();
        }}
      />

      <div className="flex flex-col gap-y-2.5">
        <IssueDetailReviewOption
          description={t("issue.review.comment-description")}
          icon={<MessageCircle className="size-4 text-control" />}
          label={t("common.comment")}
          onSelect={() => setSelectedAction("COMMENT")}
          selected={selectedAction === "COMMENT"}
        />
        {context.permissions.isApprovalCandidate && (
          <IssueDetailReviewOption
            description={t("issue.review.approve-description")}
            icon={<Check className="size-4 text-success" />}
            label={t("common.approve")}
            onSelect={() => setSelectedAction("APPROVE")}
            selected={selectedAction === "APPROVE"}
          />
        )}
        {context.permissions.isApprovalCandidate && (
          <IssueDetailReviewOption
            description={t("issue.review.reject-description")}
            icon={<X className="size-4 text-error" />}
            label={t("common.reject")}
            onSelect={() => setSelectedAction("REJECT")}
            selected={selectedAction === "REJECT"}
          />
        )}
      </div>

      <div className="flex items-center justify-start gap-x-2 pt-1">
        <Button
          disabled={submitDisabled}
          onClick={() => {
            void handleSubmit();
          }}
          size="sm"
        >
          {loading && <Loader2 className="h-4 w-4 animate-spin" />}
          {t("common.submit")}
        </Button>
        <Button onClick={() => onOpenChange(false)} size="sm" variant="ghost">
          {t("common.cancel")}
        </Button>
      </div>
    </div>
  );

  if (mobile) {
    return (
      <Sheet onOpenChange={onOpenChange} open={open}>
        <SheetContent
          className="w-[calc(100vw-2rem)] max-w-[32rem]"
          width="standard"
        >
          <SheetHeader>
            <SheetTitle>{t("issue.review.self")}</SheetTitle>
          </SheetHeader>
          <SheetBody className="py-4">{content}</SheetBody>
        </SheetContent>
      </Sheet>
    );
  }

  return (
    <div
      className={cn(
        "absolute right-0 top-full mt-2 w-[min(34rem,calc(100vw-2rem))] rounded-sm border border-control-border bg-white px-4 py-4 shadow-lg",
        LAYER_SURFACE_CLASS
      )}
      ref={popoverRef}
    >
      {content}
    </div>
  );
}

function IssueDetailReviewOption({
  description,
  icon,
  label,
  onSelect,
  selected,
}: {
  description?: string;
  icon?: ReactNode;
  label: string;
  onSelect: () => void;
  selected: boolean;
}) {
  return (
    <label
      className={cn(
        "flex cursor-pointer items-start gap-3 text-left transition-colors",
        selected ? "text-main" : "text-control"
      )}
    >
      <input
        checked={selected}
        className="mt-1 h-4 w-4 accent-accent"
        onChange={onSelect}
        type="radio"
      />
      {icon && <span className="mt-1 shrink-0">{icon}</span>}
      <span className="flex flex-col">
        <span className="text-sm font-medium leading-6">{label}</span>
        {description && (
          <span className="text-xs text-control-light">{description}</span>
        )}
      </span>
    </label>
  );
}

function isExportExpired(
  rollout: Rollout | undefined,
  taskRuns: {
    name: string;
    exportArchiveStatus: TaskRun_ExportArchiveStatus;
  }[]
) {
  if (!rollout) {
    return false;
  }
  const exportTaskRuns =
    rollout.stages
      .flatMap((stage) => stage.tasks)
      .map((task) => {
        const taskRunsForTask = taskRuns.filter(
          (taskRun) =>
            extractTaskUID(taskRun.name) === extractTaskUID(task.name)
        );
        return first(
          orderBy(
            taskRunsForTask,
            (taskRun) => Number(extractTaskRunUID(taskRun.name)),
            "desc"
          )
        );
      })
      .filter(Boolean) ?? [];

  return exportTaskRuns.every(
    (taskRun) =>
      !!taskRun &&
      taskRun.exportArchiveStatus === TaskRun_ExportArchiveStatus.EXPORTED
  );
}
