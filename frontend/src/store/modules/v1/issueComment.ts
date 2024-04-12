import { defineStore } from "pinia";
import { reactive } from "vue";
import { issueServiceClient } from "@/grpcweb";
import type { ActivityIssueCommentCreatePayload } from "@/types";
import type {
  IssueComment,
  ListIssueCommentsRequest,
} from "@/types/proto/v1/issue_service";

export enum IssueCommentType {
  USER_COMMENT = "USER_COMMENT",
  APPROVAL = "APPROVAL",
  ISSUE_UPDATE = "ISSUE_UPDATE",
  STAGE_END = "STAGE_END",
  TASK_UPDATE = "TASK_UPDATE",
  TASK_PRIOR_BACKUP = "TASK_PRIOR_BACKUP",
}

export interface ComposedIssueComment extends IssueComment {
  type: IssueCommentType;
}

const composeIssueComment = (
  issueComment: IssueComment
): ComposedIssueComment => {
  let type = IssueCommentType.USER_COMMENT;
  if (issueComment.approval !== undefined) {
    type = IssueCommentType.APPROVAL;
  } else if (issueComment.issueUpdate !== undefined) {
    type = IssueCommentType.ISSUE_UPDATE;
  } else if (issueComment.stageEnd !== undefined) {
    type = IssueCommentType.STAGE_END;
  } else if (issueComment.taskUpdate !== undefined) {
    type = IssueCommentType.TASK_UPDATE;
  } else if (issueComment.taskPriorBackup !== undefined) {
    type = IssueCommentType.TASK_PRIOR_BACKUP;
  }
  return {
    ...issueComment,
    type,
  };
};

export const useIssueCommentStore = defineStore("issue_comment", () => {
  // issueCommentMap is a map of issueName to ComposedIssueComment[].
  const issueCommentMap = reactive(new Map<string, ComposedIssueComment[]>());

  const listIssueComments = async (request: ListIssueCommentsRequest) => {
    const resp = await issueServiceClient.listIssueComments(request);
    issueCommentMap.set(
      request.parent,
      resp.issueComments.map(composeIssueComment)
    );

    return {
      nextPageToken: resp.nextPageToken,
      issueComments: resp.issueComments,
    };
  };

  const createIssueComment = async ({
    issueName,
    comment,
    payload,
  }: {
    issueName: string;
    comment: string;
    payload?: ActivityIssueCommentCreatePayload;
  }) => {
    const issueComment = await issueServiceClient.createIssueComment({
      parent: issueName,
      issueComment: {
        comment,
        payload: JSON.stringify(payload ?? {}),
      },
    });
    issueCommentMap.set(issueName, [
      ...(issueCommentMap.get(issueName) ?? []),
      composeIssueComment(issueComment),
    ]);
  };

  const updateIssueComment = async ({
    issueName,
    commentId,
    comment,
  }: {
    issueName: string;
    commentId: string;
    comment: string;
  }) => {
    await issueServiceClient.updateIssueComment({
      parent: issueName,
      issueComment: {
        uid: commentId,
        comment,
      },
      updateMask: ["comment"],
    });
    issueCommentMap.set(
      issueName,
      (issueCommentMap.get(issueName) ?? []).map((issueComment) => {
        if (issueComment.uid === commentId) {
          return {
            ...issueComment,
            comment,
          };
        }
        return issueComment;
      })
    );
  };

  const getIssueComments = (issueName: string): ComposedIssueComment[] => {
    return issueCommentMap.get(issueName) ?? [];
  };

  return {
    listIssueComments,
    createIssueComment,
    updateIssueComment,
    getIssueComments,
  };
});
