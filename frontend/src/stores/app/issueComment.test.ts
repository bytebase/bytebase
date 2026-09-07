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

test("existing timelines request the unfiltered list across pages", async () => {
  const store = createStore();
  const parent = "projects/p/issues/101";
  const timeline = [
    create(IssueCommentSchema, { name: parent + "/issueComments/root" }),
  ];
  mocks.listIssueComments.mockResolvedValue({
    issueComments: timeline,
    nextPageToken: "next",
  });
  const response = await store.listIssueComments(
    create(ListIssueCommentsRequestSchema, {
      parent,
      pageSize: 50,
      pageToken: "page",
    })
  );
  expect(mocks.listIssueComments).toHaveBeenCalledWith(
    expect.objectContaining({
      parent,
      filter: "",
      pageSize: 50,
      pageToken: "page",
    })
  );
  expect(response.nextPageToken).toBe("next");
  expect(store.getIssueComments(parent)).toEqual(timeline);

  // The explicit timeline filter refreshes the cache like the empty one.
  const refreshed = [
    ...timeline,
    create(IssueCommentSchema, { name: parent + "/issueComments/event" }),
  ];
  mocks.listIssueComments.mockResolvedValue({
    issueComments: refreshed,
    nextPageToken: "",
  });
  await store.listIssueComments(
    create(ListIssueCommentsRequestSchema, { parent, filter: "root == null" })
  );
  expect(store.getIssueComments(parent)).toEqual(refreshed);

  // Reading one thread's replies must not replace the cached timeline.
  mocks.listIssueComments.mockResolvedValue({
    issueComments: [
      create(IssueCommentSchema, { name: parent + "/issueComments/reply" }),
    ],
    nextPageToken: "",
  });
  await store.listIssueComments(
    create(ListIssueCommentsRequestSchema, {
      parent,
      filter: `root == "${parent}/issueComments/root"`,
    })
  );
  expect(store.getIssueComments(parent)).toEqual(refreshed);
});
