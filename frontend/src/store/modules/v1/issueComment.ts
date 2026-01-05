import { create } from "@bufbuild/protobuf";
import { defineStore } from "pinia";
import { reactive } from "vue";
import { issueServiceClientConnect } from "@/connect";
import type {
  IssueComment,
  ListIssueCommentsRequest,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  CreateIssueCommentRequestSchema,
  IssueCommentSchema,
  ListIssueCommentsRequestSchema,
  UpdateIssueCommentRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  getProjectIdIssueIdIssueCommentId,
  issueNamePrefix,
  projectNamePrefix,
} from "./common";

export enum IssueCommentType {
  USER_COMMENT = "USER_COMMENT",
  APPROVAL = "APPROVAL",
  ISSUE_UPDATE = "ISSUE_UPDATE",
  PLAN_SPEC_UPDATE = "PLAN_SPEC_UPDATE",
}

export const getIssueCommentType = (
  issueComment: IssueComment
): IssueCommentType => {
  if (issueComment.event?.case === "approval") {
    return IssueCommentType.APPROVAL;
  } else if (issueComment.event?.case === "issueUpdate") {
    return IssueCommentType.ISSUE_UPDATE;
  } else if (issueComment.event?.case === "planSpecUpdate") {
    return IssueCommentType.PLAN_SPEC_UPDATE;
  }
  return IssueCommentType.USER_COMMENT;
};

export const useIssueCommentStore = defineStore("issue_comment", () => {
  // issueCommentMap is a map of issueName to IssueComment[].
  const issueCommentMap = reactive(new Map<string, IssueComment[]>());

  const listIssueComments = async (request: ListIssueCommentsRequest) => {
    const connectRequest = create(ListIssueCommentsRequestSchema, {
      parent: request.parent,
      pageSize: request.pageSize,
      pageToken: request.pageToken,
    });
    const resp =
      await issueServiceClientConnect.listIssueComments(connectRequest);
    const issueComments = resp.issueComments;
    issueCommentMap.set(request.parent, issueComments);

    return {
      nextPageToken: resp.nextPageToken,
      issueComments,
    };
  };

  const createIssueComment = async ({
    issueName,
    comment,
  }: {
    issueName: string;
    comment: string;
  }) => {
    const request = create(CreateIssueCommentRequestSchema, {
      parent: issueName,
      issueComment: create(IssueCommentSchema, {
        comment,
      }),
    });
    const newIssueComment =
      await issueServiceClientConnect.createIssueComment(request);
    const issueComment = newIssueComment;
    issueCommentMap.set(issueName, [
      ...(issueCommentMap.get(issueName) ?? []),
      issueComment,
    ]);
  };

  const updateIssueComment = async ({
    issueCommentName,
    comment,
  }: {
    issueCommentName: string;
    comment: string;
  }) => {
    const { projectId, issueId } =
      getProjectIdIssueIdIssueCommentId(issueCommentName);
    const parent = `${projectNamePrefix}${projectId}/${issueNamePrefix}${issueId}`;
    const request = create(UpdateIssueCommentRequestSchema, {
      parent: parent,
      issueComment: create(IssueCommentSchema, {
        name: issueCommentName,
        comment,
      }),
      updateMask: { paths: ["comment"] },
    });
    await issueServiceClientConnect.updateIssueComment(request);
    issueCommentMap.set(
      parent,
      (issueCommentMap.get(parent) ?? []).map((issueComment) => {
        if (issueComment.name === issueCommentName) {
          return {
            ...issueComment,
            comment,
          };
        }
        return issueComment;
      })
    );
  };

  const getIssueComments = (issueName: string): IssueComment[] => {
    return issueCommentMap.get(issueName) ?? [];
  };

  return {
    listIssueComments,
    createIssueComment,
    updateIssueComment,
    getIssueComments,
  };
});
