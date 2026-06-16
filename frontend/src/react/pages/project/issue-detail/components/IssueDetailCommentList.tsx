import { create } from "@bufbuild/protobuf";
import { Loader2, Pencil, Plus } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/connect";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import {
  CommentCreator,
  canEditIssueComment,
  IssueCommentRow,
} from "@/react/components/issue-activity/IssueCommentActivity";
import { MarkdownEditor } from "@/react/components/MarkdownEditor";
import { UserAvatar } from "@/react/components/UserAvatar";
import { Button } from "@/react/components/ui/button";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import { useCurrentRoute } from "@/react/router";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { getTimeForPbTimestampProtoEs, unknownUser } from "@/types";
import {
  type IssueComment,
  IssueSchema,
  ListIssueCommentsRequestSchema,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { extractProjectResourceName } from "@/utils";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";
import { useIssueDetailContext } from "../context/IssueDetailContext";

// Stable empty reference for the no-issue branch of the comments selector.
const EMPTY_ISSUE_COMMENTS: IssueComment[] = [];

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
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  const batchGetOrFetchUsers = useAppStore(
    (state) => state.batchGetOrFetchUsers
  );
  const currentUser = useCurrentUser();
  const routeHash = useCurrentRoute().hash;
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useProjectByName(projectName);
  void projectsByName;
  const issueName = page.issue?.name || page.plan?.issue || "";
  // `getIssueComments` returns a stable empty array on miss, so reading it
  // inside the selector won't loop.
  const issueComments = useAppStore((state) =>
    issueName ? state.getIssueComments(issueName) : EMPTY_ISSUE_COMMENTS
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
        await useAppStore.getState().listIssueComments(
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
  }, [issueName, issueUpdateKey]);

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
    void batchGetOrFetchUsers([...identifiers]);
  }, [batchGetOrFetchUsers, issueComments, page.issue?.creator]);

  const refreshIssueComments = async () => {
    if (!issueName) {
      return;
    }
    await useAppStore.getState().listIssueComments(
      create(ListIssueCommentsRequestSchema, {
        parent: issueName,
        pageSize: 1000,
      })
    );
  };

  const allowEditComment = (comment: IssueComment): boolean =>
    canEditIssueComment(comment, currentUser.email, project);

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
    await useAppStore.getState().updateIssueComment({
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
    await useAppStore.getState().createIssueComment({
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
              body={
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
              comment={item}
              isLast={index === issueComments.length - 1}
              issue={page.issue}
              plan={page.plan}
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
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useProjectByName(projectName);
  void projectsByName;
  const creatorUser = useAppStore((state) =>
    state.getUserByIdentifier(page.issue?.creator || "")
  );
  const creator = creatorUser ?? unknownUser(page.issue?.creator || "");
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
      <div className="relative pb-3">
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
