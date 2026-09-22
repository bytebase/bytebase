import { create as createProto } from "@bufbuild/protobuf";
import { issueServiceClientConnect } from "@/api";
import {
  getProjectIdIssueIdIssueCommentId,
  issueNamePrefix,
  projectNamePrefix,
} from "@/stores/modules/v1/common";
import {
  CreateIssueCommentRequestSchema,
  type IssueComment,
  IssueCommentSchema,
  ListIssueCommentsRequestSchema,
  UpdateIssueCommentRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { inlineThreadsEnabled } from "@/utils/featureGates";
import { celStringList } from "@/utils/v1/celLiteral";
import type { AppSliceCreator, IssueCommentSlice } from "./types";

export enum IssueCommentType {
  USER_COMMENT = "USER_COMMENT",
  APPROVAL = "APPROVAL",
  ISSUE_UPDATE = "ISSUE_UPDATE",
  PLAN_UPDATE = "PLAN_UPDATE",
  REVIEW_SUBMISSION = "REVIEW_SUBMISSION",
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
  } else if (issueComment.event?.case === "reviewSubmission") {
    return IssueCommentType.REVIEW_SUBMISSION;
  }
  return IssueCommentType.USER_COMMENT;
};

// A thread root carries a thread state; a reply names its root. General
// comments and events carry neither.
export const isThreadRoot = (comment: IssueComment): boolean =>
  comment.threadState !== undefined && !comment.root;

export const isThreadReply = (
  comment: IssueComment
): comment is IssueComment & { root: string } => Boolean(comment.root);

// The server caps a page at 1000 entries; a full load pages until the token
// runs out, per the design's "client fetches all comments" contract.
const MAX_PAGE_SIZE = 1000;
// Roots per reply query. Keeps the CEL filter short; volumes are small.
const REPLY_QUERY_ROOTS = 50;

// Stable empty reference so `getIssueComments` can be read inside a reactive
// selector without producing a fresh array (which would loop forever).
const EMPTY_COMMENTS: IssueComment[] = [];

export const createIssueCommentSlice: AppSliceCreator<IssueCommentSlice> = (
  set,
  get
) => {
  // Loads per issue, oldest first. Only the newest load may replace the
  // cache: a load that succeeds while it is newest retires every older one,
  // so an older response never overwrites a newer one. An older load that
  // finishes first keeps its result as a fallback; if the newer load then
  // fails, the fallback is promoted rather than leaving the cache stale.
  // Successful writes are recorded on every active load, so whichever
  // result lands preserves them.
  interface Load {
    writes: Map<string, IssueComment>;
    result?: IssueComment[];
  }
  const activeLoads = new Map<string, Load[]>();
  const loadsOf = (parent: string) => activeLoads.get(parent) ?? [];
  const recordWrite = (parent: string, comment: IssueComment) => {
    for (const load of loadsOf(parent)) load.writes.set(comment.name, comment);
  };
  const beginLoad = (parent: string) => {
    const load: Load = { writes: new Map() };
    activeLoads.set(parent, [...loadsOf(parent), load]);
    return load.writes;
  };
  const applyLoad = (parent: string, load: Load, comments: IssueComment[]) => {
    activeLoads.delete(parent);
    const merged = new Map(comments.map((comment) => [comment.name, comment]));
    for (const [name, comment] of load.writes) merged.set(name, comment);
    set((state) => ({
      issueCommentsByIssue: {
        ...state.issueCommentsByIssue,
        [parent]: [...merged.values()],
      },
    }));
  };
  // Runs after a load leaves: the newest remaining load wins if it already
  // finished; otherwise the cache waits for it.
  const settleLoads = (parent: string) => {
    const newest = loadsOf(parent).at(-1);
    if (newest?.result) applyLoad(parent, newest, newest.result);
  };
  const withdrawLoad = (parent: string, writes: Map<string, IssueComment>) => {
    const loads = loadsOf(parent).filter((load) => load.writes !== writes);
    if (loads.length === 0) activeLoads.delete(parent);
    else activeLoads.set(parent, loads);
    settleLoads(parent);
  };
  const cacheLoadedComments = (
    parent: string,
    writes: Map<string, IssueComment>,
    comments: IssueComment[]
  ) => {
    const load = loadsOf(parent).find((l) => l.writes === writes);
    // Already retired by a newer success.
    if (!load) return;
    load.result = comments;
    settleLoads(parent);
  };

  return {
    issueCommentsByIssue: {},

    listIssueComments: async (request) => {
      const resp = await issueServiceClientConnect.listIssueComments(
        createProto(ListIssueCommentsRequestSchema, {
          parent: request.parent,
          pageSize: request.pageSize,
          pageToken: request.pageToken,
          filter: request.filter,
        })
      );
      return {
        nextPageToken: resp.nextPageToken,
        issueComments: resp.issueComments,
      };
    },

    fetchIssueCommentTimeline: async ({ parent, pageSize, pageToken }) => {
      const writes = beginLoad(parent);
      try {
        const resp = await get().listIssueComments(
          createProto(ListIssueCommentsRequestSchema, {
            parent,
            pageSize,
            pageToken,
          })
        );
        cacheLoadedComments(parent, writes, resp.issueComments);
        return resp;
      } catch (error) {
        withdrawLoad(parent, writes);
        throw error;
      }
    },

    fetchIssueCommentThreads: async ({ parent }) => {
      const writes = beginLoad(parent);
      try {
        const listAll = async (filter: string): Promise<IssueComment[]> => {
          const all: IssueComment[] = [];
          let pageToken = "";
          do {
            const resp = await get().listIssueComments(
              createProto(ListIssueCommentsRequestSchema, {
                parent,
                pageSize: MAX_PAGE_SIZE,
                pageToken,
                filter,
              })
            );
            all.push(...resp.issueComments);
            pageToken = resp.nextPageToken;
          } while (pageToken);
          return all;
        };

        // The timeline load is what every comment surface reads, threads or
        // not, so it runs unconditionally; only the reply pass is gated.
        const timeline = await listAll("");
        const rootNames = inlineThreadsEnabled()
          ? timeline.filter(isThreadRoot).map((c) => c.name)
          : [];
        const replies: IssueComment[] = [];
        for (let i = 0; i < rootNames.length; i += REPLY_QUERY_ROOTS) {
          const chunk = rootNames.slice(i, i + REPLY_QUERY_ROOTS);
          replies.push(...(await listAll(`root in ${celStringList(chunk)}`)));
        }
        const issueComments = [...timeline, ...replies];
        cacheLoadedComments(parent, writes, issueComments);
        return issueComments;
      } catch (error) {
        withdrawLoad(parent, writes);
        throw error;
      }
    },

    createIssueComment: async ({
      issueName,
      comment,
      root,
      statementAnchor,
    }) => {
      const newIssueComment =
        await issueServiceClientConnect.createIssueComment(
          createProto(CreateIssueCommentRequestSchema, {
            parent: issueName,
            issueComment: createProto(IssueCommentSchema, {
              comment,
              root,
              statementAnchor,
            }),
          })
        );
      recordWrite(issueName, newIssueComment);
      set((state) => ({
        issueCommentsByIssue: {
          ...state.issueCommentsByIssue,
          [issueName]: [
            ...(state.issueCommentsByIssue[issueName] ?? []),
            newIssueComment,
          ],
        },
      }));
      return newIssueComment;
    },

    updateIssueComment: async ({ issueCommentName, comment, threadState }) => {
      const { projectId, issueId } =
        getProjectIdIssueIdIssueCommentId(issueCommentName);
      const parent = `${projectNamePrefix}${projectId}/${issueNamePrefix}${issueId}`;
      const paths: string[] = [];
      if (comment !== undefined) paths.push("comment");
      if (threadState !== undefined) paths.push("thread_state");
      const updatedIssueComment =
        await issueServiceClientConnect.updateIssueComment(
          createProto(UpdateIssueCommentRequestSchema, {
            parent,
            issueComment: createProto(IssueCommentSchema, {
              name: issueCommentName,
              comment,
              threadState,
            }),
            updateMask: { paths },
          })
        );
      recordWrite(parent, updatedIssueComment);
      set((state) => ({
        issueCommentsByIssue: {
          ...state.issueCommentsByIssue,
          [parent]: (state.issueCommentsByIssue[parent] ?? []).map(
            (issueComment) =>
              issueComment.name === issueCommentName
                ? updatedIssueComment
                : issueComment
          ),
        },
      }));
      return updatedIssueComment;
    },

    getIssueComments: (issueName) =>
      get().issueCommentsByIssue[issueName] ?? EMPTY_COMMENTS,
  };
};
