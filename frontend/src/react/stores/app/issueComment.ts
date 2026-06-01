import { create as createProto } from "@bufbuild/protobuf";
import { issueServiceClientConnect } from "@/connect";
import {
  getProjectIdIssueIdIssueCommentId,
  issueNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import {
  CreateIssueCommentRequestSchema,
  type IssueComment,
  IssueCommentSchema,
  ListIssueCommentsRequestSchema,
  UpdateIssueCommentRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type { AppSliceCreator, IssueCommentSlice } from "./types";

export enum IssueCommentType {
  USER_COMMENT = "USER_COMMENT",
  APPROVAL = "APPROVAL",
  ISSUE_UPDATE = "ISSUE_UPDATE",
  PLAN_UPDATE = "PLAN_UPDATE",
}

export const getIssueCommentType = (
  issueComment: IssueComment
): IssueCommentType => {
  if (issueComment.event?.case === "approval") {
    return IssueCommentType.APPROVAL;
  } else if (issueComment.event?.case === "issueUpdate") {
    return IssueCommentType.ISSUE_UPDATE;
  } else if (issueComment.event?.case === "planUpdate") {
    return IssueCommentType.PLAN_UPDATE;
  }
  return IssueCommentType.USER_COMMENT;
};

// Stable empty reference so `getIssueComments` can be read inside a reactive
// selector without producing a fresh array (which would loop forever).
const EMPTY_COMMENTS: IssueComment[] = [];

export const createIssueCommentSlice: AppSliceCreator<IssueCommentSlice> = (
  set,
  get
) => ({
  issueCommentsByIssue: {},

  listIssueComments: async (request) => {
    const resp = await issueServiceClientConnect.listIssueComments(
      createProto(ListIssueCommentsRequestSchema, {
        parent: request.parent,
        pageSize: request.pageSize,
        pageToken: request.pageToken,
      })
    );
    set((state) => ({
      issueCommentsByIssue: {
        ...state.issueCommentsByIssue,
        [request.parent]: resp.issueComments,
      },
    }));
    return {
      nextPageToken: resp.nextPageToken,
      issueComments: resp.issueComments,
    };
  },

  createIssueComment: async ({ issueName, comment }) => {
    const newIssueComment = await issueServiceClientConnect.createIssueComment(
      createProto(CreateIssueCommentRequestSchema, {
        parent: issueName,
        issueComment: createProto(IssueCommentSchema, { comment }),
      })
    );
    set((state) => ({
      issueCommentsByIssue: {
        ...state.issueCommentsByIssue,
        [issueName]: [
          ...(state.issueCommentsByIssue[issueName] ?? []),
          newIssueComment,
        ],
      },
    }));
  },

  updateIssueComment: async ({ issueCommentName, comment }) => {
    const { projectId, issueId } =
      getProjectIdIssueIdIssueCommentId(issueCommentName);
    const parent = `${projectNamePrefix}${projectId}/${issueNamePrefix}${issueId}`;
    await issueServiceClientConnect.updateIssueComment(
      createProto(UpdateIssueCommentRequestSchema, {
        parent,
        issueComment: createProto(IssueCommentSchema, {
          name: issueCommentName,
          comment,
        }),
        updateMask: { paths: ["comment"] },
      })
    );
    set((state) => ({
      issueCommentsByIssue: {
        ...state.issueCommentsByIssue,
        [parent]: (state.issueCommentsByIssue[parent] ?? []).map(
          (issueComment) =>
            issueComment.name === issueCommentName
              ? { ...issueComment, comment }
              : issueComment
        ),
      },
    }));
  },

  getIssueComments: (issueName) =>
    get().issueCommentsByIssue[issueName] ?? EMPTY_COMMENTS,
});
