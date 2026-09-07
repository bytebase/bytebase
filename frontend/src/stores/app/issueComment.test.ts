import { create } from "@bufbuild/protobuf";
import { beforeEach, expect, test, vi } from "vitest";
import {
  IssueCommentSchema,
  ListIssueCommentsRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { createIssueCommentSlice } from "./issueComment";

const mocks = vi.hoisted(() => ({
  createIssueComment: vi.fn(),
  listIssueComments: vi.fn(),
}));
vi.mock("@/api", () => ({
  issueServiceClientConnect: mocks,
}));

const createStore = () => {
  const state: Record<string, unknown> = {};
  const set = (updater: unknown) => {
    Object.assign(
      state,
      typeof updater === "function" ? updater(state) : updater
    );
  };
  Object.assign(
    state,
    createIssueCommentSlice(set as never, (() => state) as never, {} as never)
  );
  return state as ReturnType<typeof createIssueCommentSlice>;
};

beforeEach(() => {
  vi.clearAllMocks();
});

test("the frontend composer still creates a general comment", async () => {
  const store = createStore();
  const parent = "projects/p/issues/101";
  const comment = create(IssueCommentSchema, {
    name: parent + "/issueComments/comment",
    comment: "Looks good",
  });
  mocks.createIssueComment.mockResolvedValue(comment);
  await store.createIssueComment({ issueName: parent, comment: "Looks good" });
  const request = mocks.createIssueComment.mock.calls[0][0];
  expect(request.parent).toBe(parent);
  expect(request.issueComment.comment).toBe("Looks good");
  expect(request.issueComment.root).toBeUndefined();
  expect(request.issueComment.threadState).toBeUndefined();
  expect(request.issueComment.statementAnchor).toBeUndefined();
  expect(store.getIssueComments(parent)).toEqual([comment]);
});

test("timeline loading caches the requested page and preserves other issues", async () => {
  const store = createStore();
  const parent = "projects/p/issues/101";
  const other = "projects/p/issues/102";
  const otherComments = [
    create(IssueCommentSchema, { comment: "other issue" }),
  ];
  mocks.listIssueComments.mockResolvedValueOnce({
    issueComments: otherComments,
    nextPageToken: "",
  });
  await store.fetchIssueCommentTimeline({ parent: other });

  const timeline = [
    create(IssueCommentSchema, { name: parent + "/issueComments/root" }),
  ];
  mocks.listIssueComments.mockResolvedValueOnce({
    issueComments: timeline,
    nextPageToken: "next",
  });
  const response = await store.fetchIssueCommentTimeline({
    parent,
    pageSize: 50,
    pageToken: "page",
  });
  expect(mocks.listIssueComments).toHaveBeenLastCalledWith(
    expect.objectContaining({
      parent,
      filter: "",
      pageSize: 50,
      pageToken: "page",
    })
  );
  expect(response).toEqual({ issueComments: timeline, nextPageToken: "next" });
  expect(store.getIssueComments(parent)).toEqual(timeline);
  expect(store.getIssueComments(other)).toEqual(otherComments);

  mocks.listIssueComments.mockResolvedValueOnce({
    issueComments: [],
    nextPageToken: "",
  });
  await store.fetchIssueCommentTimeline({ parent });
  expect(store.getIssueComments(parent)).toEqual([]);
  expect(store.getIssueComments(other)).toEqual(otherComments);
});

test.each([
  "",
  "root == null",
  "(root == null)",
  "root == (null)",
  "((root) == (null))",
  "root == null // timeline",
  'root == "projects/p/issues/101/issueComments/root"',
  'root in ["projects/p/issues/101/issueComments/root"]',
  "root in []",
])("arbitrary queries never change the timeline cache: %s", async (filter) => {
  const store = createStore();
  const parent = "projects/p/issues/101";
  const cached = [create(IssueCommentSchema, { comment: "cached timeline" })];
  mocks.listIssueComments.mockResolvedValueOnce({
    issueComments: cached,
    nextPageToken: "",
  });
  await store.fetchIssueCommentTimeline({ parent });

  for (const issueComments of [
    [],
    [create(IssueCommentSchema, { comment: "query result" })],
  ]) {
    const result = { issueComments, nextPageToken: "next" };
    mocks.listIssueComments.mockResolvedValueOnce(result);
    const response = await store.listIssueComments(
      create(ListIssueCommentsRequestSchema, {
        parent,
        filter,
        pageSize: 10,
        pageToken: "page",
      })
    );
    expect(response).toEqual(result);
    expect(mocks.listIssueComments).toHaveBeenLastCalledWith(
      expect.objectContaining({
        parent,
        filter,
        pageSize: 10,
        pageToken: "page",
      })
    );
    expect(store.getIssueComments(parent)).toEqual(cached);
  }
});

test("failed timeline refresh preserves the cached content and propagates the error", async () => {
  const store = createStore();
  const parent = "projects/p/issues/101";
  const cached = [create(IssueCommentSchema, { comment: "cached timeline" })];
  mocks.listIssueComments.mockResolvedValueOnce({
    issueComments: cached,
    nextPageToken: "",
  });
  await store.fetchIssueCommentTimeline({ parent });
  const error = new Error("unavailable");
  mocks.listIssueComments.mockRejectedValueOnce(error);
  await expect(store.fetchIssueCommentTimeline({ parent })).rejects.toBe(error);
  expect(store.getIssueComments(parent)).toEqual(cached);
});
