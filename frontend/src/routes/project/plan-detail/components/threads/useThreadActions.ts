import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import {
  IssueComment_ThreadState,
  type StatementAnchor,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";

// Resolve and Reopen go through UpdateIssueComment, which the server gates
// on the update permission alone; authoring the root grants nothing there,
// so the UI offers the actions only to users the server will accept.
export function canSettleThread(project: Project | undefined): boolean {
  return Boolean(
    project && hasProjectPermissionV2(project, "bb.issueComments.update")
  );
}

export function canReplyToThread(project: Project | undefined): boolean {
  return Boolean(
    project && hasProjectPermissionV2(project, "bb.issueComments.create")
  );
}

// Store calls shared by the timeline card, the editor card, and the inline
// composer. Failures surface as notifications; callers keep their drafts.
export function useThreadActions(issueName: string) {
  const { t } = useTranslation();
  const [pending, setPending] = useState(false);

  const run = useCallback(
    async <T>(
      action: () => Promise<T>,
      failureTitle?: string
    ): Promise<T | undefined> => {
      try {
        setPending(true);
        return await action();
      } catch (error) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: failureTitle ?? t("common.failed"),
          description: String(error),
        });
        return undefined;
      } finally {
        setPending(false);
      }
    },
    [t]
  );

  const publish = useCallback(
    (comment: string, statementAnchor: StatementAnchor) =>
      run(() =>
        useAppStore
          .getState()
          .createIssueComment({ issueName, comment, statementAnchor })
      ),
    [issueName, run]
  );

  const reply = useCallback(
    (root: string, comment: string) =>
      run(() =>
        useAppStore.getState().createIssueComment({ issueName, comment, root })
      ),
    [issueName, run]
  );

  const edit = useCallback(
    (issueCommentName: string, comment: string) =>
      run(() =>
        useAppStore.getState().updateIssueComment({ issueCommentName, comment })
      ),
    [run]
  );

  const setThreadState = useCallback(
    (
      issueCommentName: string,
      threadState: IssueComment_ThreadState,
      failureTitle?: string
    ) =>
      run(
        () =>
          useAppStore
            .getState()
            .updateIssueComment({ issueCommentName, threadState }),
        failureTitle
      ),
    [run]
  );

  const resolve = useCallback(
    (root: string, failureTitle?: string) =>
      setThreadState(root, IssueComment_ThreadState.RESOLVED, failureTitle),
    [setThreadState]
  );
  const reopen = useCallback(
    (root: string, failureTitle?: string) =>
      setThreadState(root, IssueComment_ThreadState.OPEN, failureTitle),
    [setThreadState]
  );

  return { edit, pending, publish, reopen, reply, resolve };
}
