import { create } from "@bufbuild/protobuf";
import {
  CheckCircle2,
  Loader2,
  Pencil,
  Play,
  Plus,
  ThumbsUp,
  Trash2,
} from "lucide-react";
import {
  Fragment,
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/connect";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import { MarkdownEditor } from "@/react/components/MarkdownEditor";
import { ReadonlyDiffMonaco } from "@/react/components/monaco";
import { UserAvatar } from "@/react/components/UserAvatar";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { buildPlanDeployRouteFromPlanName } from "@/router/dashboard/projectV1RouteHelpers";
import {
  extractUserEmail,
  getIssueCommentType,
  IssueCommentType,
  pushNotification,
  useCurrentUserV1,
  useIssueCommentStore,
  useProjectV1Store,
  useSheetV1Store,
  useUserStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { getTimeForPbTimestampProtoEs, unknownUser } from "@/types";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import {
  type IssueComment,
  IssueComment_Approval_Status,
  IssueSchema,
  IssueStatus,
  ListIssueCommentsRequestSchema,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  extractPlanUID,
  extractProjectResourceName,
  getSpecDisplayInfo,
} from "@/utils";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";
import {
  enablePriorBackupOfSpec,
  sheetNameOfSpec,
  targetsOfSpec,
} from "@/utils/v1/issue/plan";
import { getSheetStatement } from "@/utils/v1/sheet";
import { useIssueDetailContext } from "../context/IssueDetailContext";
import { isDatabaseChangeDoneRolloutComment } from "../utils/activity";
import {
  diffEntryKey,
  diffPlanSpecsForEvent,
  type SpecDiffEntry,
} from "../utils/diffPlanSpecs";

function useIssueRefTransform(projectName: string | undefined) {
  const { t } = useTranslation();
  return useCallback(
    (raw: string) =>
      raw
        .split(/(#\d+)\b/)
        .map((part) => {
          if (!part.startsWith("#")) {
            return part;
          }
          const id = Number.parseInt(part.slice(1), 10);
          if (!Number.isNaN(id) && id > 0 && projectName) {
            const projectId = extractProjectResourceName(projectName);
            const url = `${window.location.origin}/projects/${projectId}/issues/${id}`;
            return `[${t("common.issue")} #${id}](${url})`;
          }
          return part;
        })
        .join(""),
    [projectName, t]
  );
}

export function IssueDetailCommentList() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const { setEditing } = page;
  const projectStore = useProjectV1Store();
  const userStore = useUserStore();
  const issueCommentStore = useIssueCommentStore();
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const routeHash = useVueState(() => router.currentRoute.value.hash);
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));
  const issueName = page.issue?.name || page.plan?.issue || "";
  const issueComments = useVueState(() =>
    issueName ? issueCommentStore.getIssueComments(issueName) : []
  );
  const issueUpdateKey = `${page.issue?.updateTime?.seconds ?? ""}:${page.issue?.updateTime?.nanos ?? ""}`;
  const [activeCommentName, setActiveCommentName] = useState<string>();
  const [editContent, setEditContent] = useState("");
  const [newComment, setNewComment] = useState("");
  const [isRefreshing, setIsRefreshing] = useState(false);
  const newCommentTransform = useIssueRefTransform(project?.name);
  const allowCreateComment = Boolean(
    project && hasProjectPermissionV2(project, "bb.issueComments.create")
  );
  const activeComment = useMemo(
    () => issueComments.find((comment) => comment.name === activeCommentName),
    [activeCommentName, issueComments]
  );
  const allowUpdateComment = Boolean(
    activeComment && editContent && editContent !== activeComment.comment
  );

  useEffect(() => {
    setEditing("comment-row", Boolean(activeCommentName));
    return () => {
      setEditing("comment-row", false);
    };
  }, [activeCommentName, setEditing]);

  useEffect(() => {
    if (!issueName) {
      return;
    }
    let canceled = false;
    const run = async () => {
      try {
        setIsRefreshing(true);
        await issueCommentStore.listIssueComments(
          create(ListIssueCommentsRequestSchema, {
            parent: issueName,
            pageSize: 1000,
          })
        );
      } finally {
        if (!canceled) {
          setIsRefreshing(false);
        }
      }
    };
    void run();
    return () => {
      canceled = true;
    };
  }, [issueCommentStore, issueName, issueUpdateKey]);

  useEffect(() => {
    if (!routeHash.match(/^#activity(\d+)/)) {
      return;
    }
    const elem =
      document.querySelector(routeHash) || document.querySelector("#activity");
    const timer = window.setTimeout(() => elem?.scrollIntoView());
    return () => window.clearTimeout(timer);
  }, [routeHash]);

  useEffect(() => {
    const identifiers = new Set<string>();
    if (page.issue?.creator) {
      identifiers.add(page.issue.creator);
    }
    for (const comment of issueComments) {
      if (comment.creator) {
        identifiers.add(comment.creator);
      }
    }
    if (identifiers.size === 0) {
      return;
    }
    void userStore.batchGetOrFetchUsers([...identifiers]);
  }, [issueComments, page.issue?.creator, userStore]);

  const refreshIssueComments = async () => {
    if (!issueName) {
      return;
    }
    await issueCommentStore.listIssueComments(
      create(ListIssueCommentsRequestSchema, {
        parent: issueName,
        pageSize: 1000,
      })
    );
  };

  const allowEditComment = (comment: IssueComment): boolean => {
    if (!project) {
      return false;
    }
    const commentType = getIssueCommentType(comment);
    const isEditable =
      commentType === IssueCommentType.USER_COMMENT ||
      (commentType === IssueCommentType.APPROVAL && comment.comment !== "");
    if (!isEditable) {
      return false;
    }
    if (currentUser.email === extractUserEmail(comment.creator)) {
      return true;
    }
    return hasProjectPermissionV2(project, "bb.issueComments.update");
  };

  const startEditComment = (comment: IssueComment) => {
    setActiveCommentName(comment.name);
    setEditContent(comment.comment);
  };

  const cancelEditComment = () => {
    setActiveCommentName(undefined);
    setEditContent("");
  };

  const saveEditComment = async () => {
    if (!activeComment || !allowUpdateComment) {
      return;
    }
    await issueCommentStore.updateIssueComment({
      issueCommentName: activeComment.name,
      comment: editContent,
    });
    cancelEditComment();
    await refreshIssueComments();
  };

  const createComment = async () => {
    if (!issueName || !newComment) {
      return;
    }
    await issueCommentStore.createIssueComment({
      issueName,
      comment: newComment,
    });
    setNewComment("");
    await refreshIssueComments();
  };

  return (
    <div className="flex flex-col">
      <ul>
        <IssueDescriptionCommentRow
          issueComments={issueComments}
          onRefresh={refreshIssueComments}
        />
        {issueComments.map((item, index) => {
          const isEditing = activeCommentName === item.name;
          return (
            <IssueCommentRow
              key={item.name}
              isLast={index === issueComments.length - 1}
              issueComment={item}
              comment={
                item.comment ? (
                  <EditableMarkdownContent
                    allowSave={allowUpdateComment}
                    content={item.comment}
                    editContent={editContent}
                    isEditing={isEditing}
                    onCancel={cancelEditComment}
                    onChange={setEditContent}
                    onSave={() => {
                      void saveEditComment();
                    }}
                    placeholder={t("issue.no-description-provided")}
                    projectName={project?.name}
                  />
                ) : null
              }
              subjectSuffix={
                allowEditComment(item) && !activeCommentName ? (
                  <Button
                    className="text-gray-500 hover:text-gray-700"
                    onClick={() => startEditComment(item)}
                    size="xs"
                    variant="ghost"
                  >
                    <Pencil className="h-3.5 w-3.5" />
                  </Button>
                ) : null
              }
            />
          );
        })}
      </ul>

      {!activeCommentName && allowCreateComment && (
        <div className="mt-2">
          <div className="flex gap-3">
            <div className="shrink-0 pt-1">
              <UserAvatar
                size="sm"
                title={currentUser.title || currentUser.email}
              />
            </div>
            <div className="min-w-0 flex-1">
              <h3 className="sr-only" id="issue-comment-editor">
                {t("common.comment")}
              </h3>
              <MarkdownEditor
                content={newComment}
                onChange={setNewComment}
                onSubmit={() => {
                  void createComment();
                }}
                placeholder={t("issue.leave-a-comment")}
                transform={newCommentTransform}
              />
              <div className="mt-3 flex items-center justify-end">
                <Button
                  disabled={newComment.length === 0}
                  onClick={() => {
                    void createComment();
                  }}
                  size="sm"
                  type="button"
                >
                  {t("common.comment")}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
      {isRefreshing && issueComments.length === 0 && (
        <div className="mt-3 flex items-center justify-center text-sm text-control-light">
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          {t("common.loading")}
        </div>
      )}
    </div>
  );
}

function IssueDescriptionCommentRow({
  issueComments,
  onRefresh,
}: {
  issueComments: IssueComment[];
  onRefresh: () => Promise<void>;
}) {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const { setEditing } = page;
  const projectStore = useProjectV1Store();
  const userStore = useUserStore();
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));
  const creator = useVueState(
    () =>
      userStore.getUserByIdentifier(page.issue?.creator || "") ??
      unknownUser(page.issue?.creator || "")
  );
  const [isEditing, setIsEditing] = useState(false);
  const [editContent, setEditContent] = useState(page.issue?.description || "");
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (!isEditing) {
      setEditContent(page.issue?.description || "");
    }
  }, [isEditing, page.issue?.description]);

  useEffect(() => {
    setEditing("issue-description", isEditing);
    return () => {
      setEditing("issue-description", false);
    };
  }, [isEditing, setEditing]);

  const allowEdit = Boolean(
    project && hasProjectPermissionV2(project, "bb.issues.update")
  );
  const allowSave = editContent !== (page.issue?.description || "");

  const saveEdit = async () => {
    if (!allowSave || !page.issue) {
      return;
    }
    try {
      setIsSaving(true);
      const issuePatch = create(IssueSchema, {
        name: page.issue.name,
        description: editContent,
      });
      const request = create(UpdateIssueRequestSchema, {
        issue: issuePatch,
        updateMask: { paths: ["description"] },
      });
      const response = await issueServiceClientConnect.updateIssue(request);
      page.patchState({ issue: response });
      setIsEditing(false);
      await onRefresh();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <li>
      <div className="relative pb-4">
        {issueComments.length > 0 && (
          <span
            aria-hidden="true"
            className="absolute left-4 -ml-px h-full w-0.5 bg-block-border"
          />
        )}
        <div className="relative flex items-start">
          <div className="relative">
            <div className="bg-white pt-1.5" />
            <UserAvatar
              className="h-7 w-7 text-[0.8rem] font-medium"
              size="sm"
              title={creator.title || creator.email}
            />
            <div className="absolute -bottom-1 -right-1 flex h-4 w-4 items-center justify-center rounded-full bg-control-bg ring-2 ring-white">
              <Plus className="h-4 w-4 text-control" />
            </div>
          </div>

          <div className="ml-3 min-w-0 flex-1">
            <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
              <div className="bg-gray-50 px-3 py-2">
                <div className="flex items-center justify-between">
                  <div className="flex min-w-0 flex-wrap items-center gap-x-2 text-sm">
                    <CommentCreator creator={creator} />
                    <span className="wrap-break-word min-w-0 text-gray-600">
                      {t("activity.sentence.created-issue")}
                    </span>
                    {page.issue?.createTime && (
                      <HumanizeTs
                        className="text-gray-500"
                        ts={
                          getTimeForPbTimestampProtoEs(
                            page.issue.createTime,
                            0
                          ) / 1000
                        }
                      />
                    )}
                  </div>
                  {allowEdit && !isEditing && (
                    <Button
                      onClick={() => setIsEditing(true)}
                      size="xs"
                      variant="ghost"
                    >
                      <Pencil className="h-3.5 w-3.5" />
                    </Button>
                  )}
                </div>
              </div>
              <div className="border-t border-gray-200 px-4 py-3 text-sm text-gray-700">
                <EditableMarkdownContent
                  allowSave={allowSave}
                  content={page.issue?.description || ""}
                  editContent={editContent}
                  isEditing={isEditing}
                  isSaving={isSaving}
                  onCancel={() => {
                    setEditContent(page.issue?.description || "");
                    setIsEditing(false);
                  }}
                  onChange={setEditContent}
                  onSave={() => {
                    void saveEdit();
                  }}
                  placeholder={t("issue.no-description-provided")}
                  projectName={project?.name}
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </li>
  );
}

function IssueCommentRow({
  comment,
  isLast,
  issueComment,
  subjectSuffix,
}: {
  comment?: ReactNode;
  isLast: boolean;
  issueComment: IssueComment;
  subjectSuffix?: ReactNode;
}) {
  return (
    <li>
      <div className="relative pb-4" id={issueComment.name}>
        {!isLast && (
          <span
            aria-hidden="true"
            className="absolute left-4 -ml-px h-full w-0.5 bg-gray-200"
          />
        )}
        <div className="relative flex items-start">
          <div className="pt-1.5">
            <CommentActionIcon issueComment={issueComment} />
          </div>
          <div className="min-w-0 flex-1">
            <div
              className={cn(
                "overflow-hidden rounded-lg border",
                comment
                  ? "ml-3 border-gray-200 bg-white"
                  : "ml-1 border-transparent"
              )}
            >
              <div className={cn("px-3 py-2", comment && "bg-gray-50")}>
                <div className="flex items-center justify-between">
                  <div className="flex min-w-0 flex-wrap items-center gap-x-2 text-sm">
                    <CommentActionHeader issueComment={issueComment} />
                  </div>
                  {subjectSuffix}
                </div>
              </div>
              {comment && (
                <div className="wrap-break-word border-t border-gray-200 px-4 py-3 text-sm whitespace-pre-wrap text-gray-700">
                  {comment}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </li>
  );
}

function CommentActionHeader({ issueComment }: { issueComment: IssueComment }) {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const userStore = useUserStore();
  const creator = useVueState(
    () =>
      userStore.getUserByIdentifier(issueComment.creator) ??
      unknownUser(issueComment.creator)
  );
  const createdTs = getTimeForPbTimestampProtoEs(issueComment.createTime, 0);
  const updatedTs = getTimeForPbTimestampProtoEs(issueComment.updateTime, 0);
  const isEdited =
    createdTs !== updatedTs &&
    getIssueCommentType(issueComment) === IssueCommentType.USER_COMMENT;

  return (
    <>
      {!isDatabaseChangeDoneRolloutComment(
        page.issue,
        page.plan,
        issueComment
      ) && <CommentCreator creator={creator} />}
      <CommentActionSentence issueComment={issueComment} />
      {issueComment.createTime && (
        <HumanizeTs className="text-gray-500" ts={createdTs / 1000} />
      )}
      {isEdited && (
        <span className="text-xs text-gray-500">({t("common.edited")})</span>
      )}
    </>
  );
}

function CommentCreator({ creator }: { creator: User }) {
  return (
    <span className="font-medium text-main">
      {creator.title || creator.email}
    </span>
  );
}

function CommentActionSentence({
  issueComment,
}: {
  issueComment: IssueComment;
}) {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const commentType = getIssueCommentType(issueComment);

  if (
    commentType === IssueCommentType.APPROVAL &&
    issueComment.event.case === "approval"
  ) {
    const { status } = issueComment.event.value;
    if (status === IssueComment_Approval_Status.APPROVED) {
      return (
        <span className="wrap-break-word min-w-0 text-gray-600">
          {t("custom-approval.issue-review.approved-issue")}
        </span>
      );
    }
    if (status === IssueComment_Approval_Status.REJECTED) {
      return (
        <span className="wrap-break-word min-w-0 text-gray-600">
          {t("custom-approval.issue-review.rejected-issue")}
        </span>
      );
    }
    if (status === IssueComment_Approval_Status.PENDING) {
      return (
        <span className="wrap-break-word min-w-0 text-gray-600">
          {t("custom-approval.issue-review.re-requested-review")}
        </span>
      );
    }
  }

  if (
    commentType === IssueCommentType.ISSUE_UPDATE &&
    issueComment.event.case === "issueUpdate"
  ) {
    const {
      fromDescription,
      fromLabels,
      fromStatus,
      fromTitle,
      toDescription,
      toLabels,
      toStatus,
      toTitle,
    } = issueComment.event.value;
    if (fromTitle !== undefined && toTitle !== undefined) {
      return (
        <span className="wrap-break-word min-w-0 text-gray-600">
          {t("activity.sentence.changed-from-to", {
            name: t("issue.issue-name").toLowerCase(),
            newValue: toTitle,
            oldValue: fromTitle,
          })}
        </span>
      );
    }
    if (fromDescription !== undefined && toDescription !== undefined) {
      return (
        <span className="wrap-break-word min-w-0 text-gray-600">
          {t("activity.sentence.changed-description")}
        </span>
      );
    }
    if (fromStatus !== undefined && toStatus !== undefined) {
      if (toStatus === IssueStatus.DONE) {
        if (
          isDatabaseChangeDoneRolloutComment(
            page.issue,
            page.plan,
            issueComment
          )
        ) {
          const planUID = page.plan ? extractPlanUID(page.plan.name) : "";
          const planHref = page.plan
            ? router.resolve(buildPlanDeployRouteFromPlanName(page.plan.name))
                .href
            : "";
          const sentence =
            page.issue?.approvalStatus === ApprovalStatus.APPROVED
              ? planUID
                ? t("activity.sentence.review-done-rollout-created-for-plan")
                : t("activity.sentence.review-done-rollout-created")
              : planUID
                ? t("activity.sentence.review-skipped-rollout-created-for-plan")
                : t("activity.sentence.review-skipped-rollout-created");

          return (
            <span className="wrap-break-word min-w-0 text-gray-600">
              {sentence}
              {planUID && planHref && (
                <>
                  {" "}
                  <a
                    className="font-medium text-accent hover:underline"
                    href={planHref}
                  >
                    #{planUID}
                  </a>
                </>
              )}
            </span>
          );
        }
        return (
          <span className="wrap-break-word min-w-0 text-gray-600">
            {t("activity.sentence.resolved-issue")}
          </span>
        );
      }
      if (toStatus === IssueStatus.CANCELED) {
        return (
          <span className="wrap-break-word min-w-0 text-gray-600">
            {t("activity.sentence.canceled-issue")}
          </span>
        );
      }
      if (toStatus === IssueStatus.OPEN) {
        return (
          <span className="wrap-break-word min-w-0 text-gray-600">
            {t("activity.sentence.reopened-issue")}
          </span>
        );
      }
    }
    if (fromLabels.length !== 0 || toLabels.length !== 0) {
      return (
        <span className="wrap-break-word min-w-0 text-gray-600">
          {t("activity.sentence.changed-labels")}
        </span>
      );
    }
  }

  if (
    commentType === IssueCommentType.PLAN_UPDATE &&
    issueComment.event.case === "planUpdate"
  ) {
    const entries = diffPlanSpecsForEvent(issueComment.event.value);
    if (entries.length === 0) return null;
    if (entries.length === 1) {
      return <SpecDiffRow entry={entries[0]} />;
    }
    return (
      <div className="flex flex-col gap-1">
        {entries.map((entry) => (
          <SpecDiffRow key={diffEntryKey(entry)} entry={entry} />
        ))}
      </div>
    );
  }

  return <span className="wrap-break-word min-w-0 text-gray-600" />;
}

function SpecDiffRow({ entry }: { entry: SpecDiffEntry }) {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const planName = page.plan?.name ?? "";

  if (entry.kind === "added") {
    return (
      <SpecChangeRow specRef={specResourceName(planName, entry.spec)}>
        {t("activity.sentence.added-spec")}
      </SpecChangeRow>
    );
  }

  if (entry.kind === "removed") {
    return (
      <SpecChangeRow specRef={specResourceName(planName, entry.spec)}>
        {t("activity.sentence.removed-spec")}
      </SpecChangeRow>
    );
  }

  // updated
  const fragments: ReactNode[] = [];
  let trailing: ReactNode = null;
  if (entry.sheetChanged) {
    const fromSheet = sheetNameOfSpec(entry.from);
    const toSheet = sheetNameOfSpec(entry.to);
    fragments.push(
      <span key="sheet">{t("activity.sentence.modified-sql-of")}</span>
    );
    trailing = (
      <IssueDetailStatementUpdateButton
        newSheet={toSheet}
        oldSheet={fromSheet}
      />
    );
  }
  if (entry.targetsChanged) {
    const fromTargets = targetsOfSpec(entry.from);
    const toTargets = targetsOfSpec(entry.to);
    const fromSet = new Set(fromTargets);
    const toSet = new Set(toTargets);
    const added = toTargets.filter((x) => !fromSet.has(x));
    const removed = fromTargets.filter((x) => !toSet.has(x));
    const diffText = [
      added.length > 0 ? `+${added.join(", ")}` : null,
      removed.length > 0 ? `-${removed.join(", ")}` : null,
    ]
      .filter(Boolean)
      .join("  ");
    fragments.push(
      <span key="targets" className="inline-flex items-center gap-1">
        {t("activity.sentence.changed-targets-of")}{" "}
        <span className="text-xs">{diffText}</span>
      </span>
    );
  }
  if (entry.priorBackupChanged) {
    const flipped = enablePriorBackupOfSpec(entry.to);
    fragments.push(
      <span key="backup">
        {flipped
          ? t("activity.sentence.enabled-prior-backup-on")
          : t("activity.sentence.disabled-prior-backup-on")}
      </span>
    );
  }
  if (fragments.length === 0 && entry.otherChanged) {
    // Unknown attribute change — generic fallback with a JSON-diff toggle.
    return (
      <SpecChangeRow specRef={specResourceName(planName, entry.to)}>
        <span>{t("common.updated")}</span>
        <details className="ml-2 text-xs">
          <summary className="cursor-pointer text-control-light">
            {t("common.detail")}
          </summary>
          <pre className="mt-1 max-h-48 overflow-auto rounded bg-control-bg p-2">
            {JSON.stringify({ from: entry.from, to: entry.to }, null, 2)}
          </pre>
        </details>
      </SpecChangeRow>
    );
  }

  return (
    <SpecChangeRow
      specRef={specResourceName(planName, entry.to)}
      trailing={trailing}
    >
      {joinFragments(fragments, t("common.and"))}
    </SpecChangeRow>
  );
}

function specResourceName(planName: string, spec: Plan_Spec): string {
  return planName ? `${planName}/specs/${spec.id}` : `specs/${spec.id}`;
}

function SpecChangeRow({
  specRef,
  children,
  trailing,
}: {
  specRef: string;
  children: ReactNode;
  trailing?: ReactNode;
}) {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const specs = page.plan?.specs ?? [];
  const specInfo = getSpecDisplayInfo(specs, specRef);
  const specIdFromRef = specRef.match(/\/specs\/([^/]+)$/)?.[1] ?? "";
  const specId = specInfo?.specId ?? specIdFromRef;
  const specIdShort = specId.slice(0, 8);
  // Only link to specs that still exist in the live plan — otherwise the spec
  // view would silently bounce back to specs[0].
  const href = specInfo?.specId
    ? router.resolve({
        query: {
          ...router.currentRoute.value.query,
          spec: specInfo.specId,
        },
      }).href
    : null;
  const onClick = href
    ? (e: React.MouseEvent) => {
        // Allow modifier-clicks (open in new tab) to use the native href.
        if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
        e.preventDefault();
        void router.push({
          query: {
            ...router.currentRoute.value.query,
            spec: specInfo!.specId,
          },
        });
      }
    : undefined;

  const chip = (
    <span className="inline-flex items-center gap-1">
      {t("plan.spec.change")}
      {specIdShort !== "" ? (
        <span className="rounded-full bg-control-bg px-1.5 py-0.5 text-xs text-main">
          {specIdShort}
        </span>
      ) : null}
    </span>
  );

  return (
    <span className="inline-flex items-center gap-1 whitespace-nowrap text-gray-600">
      {children}{" "}
      {href != null ? (
        <a
          className="inline-flex items-center gap-1 hover:underline"
          href={href}
          onClick={onClick}
        >
          {chip}
        </a>
      ) : (
        chip
      )}
      {trailing != null ? <> {trailing}</> : null}
    </span>
  );
}

function joinFragments(fragments: ReactNode[], separator: string): ReactNode {
  if (fragments.length === 0) return null;
  if (fragments.length === 1) return fragments[0];
  return fragments.map((f, i) => (
    <Fragment key={i}>
      {i > 0 ? <span> {separator} </span> : null}
      {f}
    </Fragment>
  ));
}

function CommentActionIcon({ issueComment }: { issueComment: IssueComment }) {
  const page = useIssueDetailContext();
  const userStore = useUserStore();
  const user = useVueState(
    () =>
      userStore.getUserByIdentifier(issueComment.creator) ??
      unknownUser(issueComment.creator)
  );
  const commentType = getIssueCommentType(issueComment);

  if (
    commentType === IssueCommentType.APPROVAL &&
    issueComment.event.case === "approval"
  ) {
    const { status } = issueComment.event.value;
    if (status === IssueComment_Approval_Status.APPROVED) {
      return (
        <CommentIconBadge
          className="bg-success text-white"
          icon={<ThumbsUp className="h-4 w-4" />}
        />
      );
    }
    if (status === IssueComment_Approval_Status.REJECTED) {
      return (
        <CommentIconBadge
          className="bg-warning text-white"
          icon={<Pencil className="h-4 w-4" />}
        />
      );
    }
    if (status === IssueComment_Approval_Status.PENDING) {
      return (
        <CommentIconBadge
          className="bg-control-bg text-control"
          icon={<Play className="ml-px h-4 w-4" strokeWidth={3} />}
        />
      );
    }
  }

  if (
    commentType === IssueCommentType.ISSUE_UPDATE &&
    issueComment.event.case === "issueUpdate"
  ) {
    if (
      isDatabaseChangeDoneRolloutComment(page.issue, page.plan, issueComment)
    ) {
      return (
        <CommentIconBadge
          className="bg-success text-white"
          icon={<CheckCircle2 className="size-4" />}
        />
      );
    }

    const { fromLabels, toDescription, toLabels, toTitle } =
      issueComment.event.value;
    if (
      toTitle !== undefined ||
      toDescription !== undefined ||
      toLabels.length !== 0 ||
      fromLabels.length !== 0
    ) {
      return (
        <CommentIconBadge
          className="bg-control-bg text-control"
          icon={<Pencil className="h-4 w-4" />}
        />
      );
    }
  }

  if (
    commentType === IssueCommentType.PLAN_UPDATE &&
    issueComment.event.case === "planUpdate"
  ) {
    const entries = diffPlanSpecsForEvent(issueComment.event.value);
    const allAdded =
      entries.length > 0 && entries.every((e) => e.kind === "added");
    const allRemoved =
      entries.length > 0 && entries.every((e) => e.kind === "removed");
    if (allAdded) {
      return (
        <CommentIconBadge
          className="bg-success text-white"
          icon={<Plus className="h-4 w-4" />}
        />
      );
    }
    if (allRemoved) {
      return (
        <CommentIconBadge
          className="bg-error text-white"
          icon={<Trash2 className="h-4 w-4" />}
        />
      );
    }
    return (
      <CommentIconBadge
        className="bg-control-bg text-control"
        icon={<Pencil className="h-4 w-4" />}
      />
    );
  }

  return (
    <div className="relative pl-0.5">
      <div className="flex h-7 w-7 items-center justify-center rounded-full bg-white ring-4 ring-white">
        <UserAvatar
          className="h-7 w-7 text-[0.8rem] font-medium"
          size="sm"
          title={user.title || user.email}
        />
      </div>
    </div>
  );
}

function CommentIconBadge({
  className,
  icon,
}: {
  className: string;
  icon: ReactNode;
}) {
  return (
    <div className="relative pl-0.5">
      <div
        className={cn(
          "flex h-7 w-7 items-center justify-center rounded-full ring-4 ring-white",
          className
        )}
      >
        {icon}
      </div>
    </div>
  );
}

function EditableMarkdownContent({
  allowSave,
  content,
  editContent,
  isEditing,
  isSaving = false,
  onCancel,
  onChange,
  onSave,
  placeholder,
  projectName,
}: {
  allowSave: boolean;
  content: string;
  editContent: string;
  isEditing: boolean;
  isSaving?: boolean;
  onCancel: () => void;
  onChange: (value: string) => void;
  onSave: () => void;
  placeholder: string;
  projectName?: string;
}) {
  const { t } = useTranslation();
  const transform = useIssueRefTransform(projectName);

  if (!isEditing && !content) {
    return (
      <p>
        <i className="italic text-gray-400">{placeholder}</i>
      </p>
    );
  }

  return (
    <div>
      <MarkdownEditor
        content={isEditing ? editContent : content}
        maxHeight={Number.MAX_SAFE_INTEGER}
        mode={isEditing ? "editor" : "preview"}
        onChange={onChange}
        onSubmit={onSave}
        transform={transform}
      />
      {isEditing && (
        <div className="mt-2 flex items-center justify-end gap-x-2">
          <Button onClick={onCancel} size="xs" variant="ghost">
            {t("common.cancel")}
          </Button>
          <Button disabled={!allowSave || isSaving} onClick={onSave} size="xs">
            {isSaving && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
            {t("common.save")}
          </Button>
        </div>
      )}
    </div>
  );
}

function IssueDetailStatementUpdateButton({
  newSheet,
  oldSheet,
}: {
  newSheet: string;
  oldSheet: string;
}) {
  const { t } = useTranslation();
  const sheetStore = useSheetV1Store();
  const [open, setOpen] = useState(false);
  const [oldStatement, setOldStatement] = useState("");
  const [newStatement, setNewStatement] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (!open) {
      return;
    }
    let canceled = false;
    const load = async () => {
      setIsLoading(true);
      try {
        // One side may be empty when the sheet was attached (no oldSheet)
        // or cleared (no newSheet). Treat that side as empty content
        // rather than fetching by an empty resource name.
        const fetchStatement = async (sheetName: string) => {
          if (!sheetName) return "";
          const sheet = await sheetStore.getOrFetchSheetByName(sheetName);
          return sheet ? getSheetStatement(sheet) : "";
        };
        const [oldValue, newValue] = await Promise.all([
          fetchStatement(oldSheet),
          fetchStatement(newSheet),
        ]);
        if (!canceled) {
          setOldStatement(oldValue);
          setNewStatement(newValue);
        }
      } finally {
        if (!canceled) {
          setIsLoading(false);
        }
      }
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [newSheet, oldSheet, open, sheetStore]);

  return (
    <>
      <button
        className="inline-flex items-center text-accent hover:underline"
        onClick={() => setOpen(true)}
        type="button"
      >
        {t("common.view-details")}
      </button>
      <Dialog onOpenChange={setOpen} open={open}>
        <DialogContent className="max-w-none border-0 p-0 sm:w-[calc(100vw-9rem)] 2xl:max-w-none">
          <div className="px-6 pt-5">
            <DialogTitle>{t("common.detail")}</DialogTitle>
          </div>
          <div className="px-6 pb-6 pt-2">
            {(() => {
              // Fill the dialog instead of letting the editor shrink to
              // content height — short SQL diffs were leaving most of the
              // dialog blank. Cap at a sensible max so very tall viewports
              // don't get an awkwardly-stretched editor.
              const height = Math.min(
                900,
                Math.max(400, window.innerHeight - 240)
              );
              if (isLoading) {
                return (
                  <div
                    className="flex w-full items-center justify-center rounded-md border"
                    style={{ height }}
                  >
                    <Loader2 className="h-6 w-6 animate-spin text-control-light" />
                  </div>
                );
              }
              return (
                <ReadonlyDiffMonaco
                  className="w-full"
                  max={height}
                  min={height}
                  modified={newStatement}
                  original={oldStatement}
                />
              );
            })()}
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
