// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, expect, test, vi } from "vitest";
import { PositionSchema } from "@/types/proto-es/v1/common_pb";
import {
  type IssueComment,
  IssueComment_ThreadState,
  IssueCommentSchema,
  ListIssueCommentsRequestSchema,
  StatementAnchorSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { createIssueCommentSlice } from "./issueComment";

const mocks = vi.hoisted(() => ({
  createIssueComment: vi.fn(),
  listIssueComments: vi.fn(),
  updateIssueComment: vi.fn(),
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

test("a thread load pages the timeline and fetches replies per root chunk", async () => {
  const store = createStore();
  const parent = "projects/p/issues/101";
  const rootA = create(IssueCommentSchema, {
    name: parent + "/issueComments/a",
    threadState: IssueComment_ThreadState.OPEN,
  });
  const rootB = create(IssueCommentSchema, {
    name: parent + "/issueComments/b",
    threadState: IssueComment_ThreadState.RESOLVED,
  });
  const general = create(IssueCommentSchema, {
    name: parent + "/issueComments/general",
  });
  const replyA = create(IssueCommentSchema, {
    name: parent + "/issueComments/reply-a",
    root: rootA.name,
  });
  mocks.listIssueComments
    .mockResolvedValueOnce({ issueComments: [rootA], nextPageToken: "p2" })
    .mockResolvedValueOnce({
      issueComments: [general, rootB],
      nextPageToken: "",
    })
    .mockResolvedValueOnce({ issueComments: [replyA], nextPageToken: "" });

  const loaded = await store.fetchIssueCommentThreads({ parent });

  expect(mocks.listIssueComments).toHaveBeenCalledTimes(3);
  expect(mocks.listIssueComments.mock.calls[0][0]).toMatchObject({
    parent,
    filter: "",
    pageSize: 1000,
    pageToken: "",
  });
  expect(mocks.listIssueComments.mock.calls[1][0]).toMatchObject({
    filter: "",
    pageToken: "p2",
  });
  expect(mocks.listIssueComments.mock.calls[2][0]).toMatchObject({
    parent,
    filter: `root in ["${rootA.name}", "${rootB.name}"]`,
    pageSize: 1000,
  });
  expect(loaded).toEqual([rootA, general, rootB, replyA]);
  expect(store.getIssueComments(parent)).toEqual(loaded);
});

test("a thread load without roots skips the reply query", async () => {
  const store = createStore();
  const parent = "projects/p/issues/101";
  mocks.listIssueComments.mockResolvedValueOnce({
    issueComments: [create(IssueCommentSchema, { comment: "plain" })],
    nextPageToken: "",
  });
  await store.fetchIssueCommentThreads({ parent });
  expect(mocks.listIssueComments).toHaveBeenCalledTimes(1);
});

test("an anchored comment starts a thread and a root creates a reply", async () => {
  const store = createStore();
  const parent = "projects/p/issues/101";
  const anchor = create(StatementAnchorSchema, {
    spec: "spec-1",
    sheetSha256: "f".repeat(64),
    startPosition: create(PositionSchema, { line: 2, column: 0 }),
    endPosition: create(PositionSchema, { line: 3, column: 0 }),
  });
  const root = create(IssueCommentSchema, {
    name: parent + "/issueComments/root",
    threadState: IssueComment_ThreadState.OPEN,
    statementAnchor: anchor,
  });
  mocks.createIssueComment.mockResolvedValueOnce(root);
  const created = await store.createIssueComment({
    issueName: parent,
    comment: "No LIMIT here",
    statementAnchor: anchor,
  });
  expect(created).toBe(root);
  const rootRequest = mocks.createIssueComment.mock.calls[0][0];
  expect(rootRequest.issueComment.statementAnchor).toMatchObject({
    spec: "spec-1",
    startPosition: { line: 2, column: 0 },
    endPosition: { line: 3, column: 0 },
  });
  expect(rootRequest.issueComment.root).toBeUndefined();

  const reply = create(IssueCommentSchema, {
    name: parent + "/issueComments/reply",
    root: root.name,
  });
  mocks.createIssueComment.mockResolvedValueOnce(reply);
  await store.createIssueComment({
    issueName: parent,
    comment: "Agreed",
    root: root.name,
  });
  const replyRequest = mocks.createIssueComment.mock.calls[1][0];
  expect(replyRequest.issueComment.root).toBe(root.name);
  expect(replyRequest.issueComment.statementAnchor).toBeUndefined();
  expect(store.getIssueComments(parent)).toEqual([root, reply]);
});

test("updates mask only the provided fields", async () => {
  const store = createStore();
  const parent = "projects/p/issues/101";
  const name = parent + "/issueComments/root";
  const resolved = create(IssueCommentSchema, {
    name,
    threadState: IssueComment_ThreadState.RESOLVED,
  });
  mocks.updateIssueComment.mockResolvedValueOnce(resolved);
  await store.updateIssueComment({
    issueCommentName: name,
    threadState: IssueComment_ThreadState.RESOLVED,
  });
  const resolveRequest = mocks.updateIssueComment.mock.calls[0][0];
  expect(resolveRequest.updateMask.paths).toEqual(["thread_state"]);
  expect(resolveRequest.issueComment.threadState).toBe(
    IssueComment_ThreadState.RESOLVED
  );
  expect(resolveRequest.parent).toBe(parent);

  mocks.updateIssueComment.mockResolvedValueOnce(
    create(IssueCommentSchema, { name, comment: "edited" })
  );
  await store.updateIssueComment({ issueCommentName: name, comment: "edited" });
  const editRequest = mocks.updateIssueComment.mock.calls[1][0];
  expect(editRequest.updateMask.paths).toEqual(["comment"]);
  expect(editRequest.issueComment.threadState).toBeUndefined();
});

test.each(["resolve", "create"])(
  "thread refresh preserves a successful local %s",
  async (action) => {
    const store = createStore();
    const parent = "projects/p/issues/101";
    const root = create(IssueCommentSchema, {
      name: parent + "/issueComments/root",
      threadState: IssueComment_ThreadState.OPEN,
    });
    store.issueCommentsByIssue[parent] = [root];
    let release!: (value: {
      issueComments: never[];
      nextPageToken: string;
    }) => void;
    mocks.listIssueComments.mockReset();
    mocks.listIssueComments
      .mockResolvedValueOnce({ issueComments: [root], nextPageToken: "" })
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            release = resolve;
          })
      );
    const pending = store.fetchIssueCommentThreads({ parent });
    await vi.waitFor(() =>
      expect(mocks.listIssueComments).toHaveBeenCalledTimes(2)
    );
    if (action === "resolve") {
      mocks.updateIssueComment.mockResolvedValueOnce(
        create(IssueCommentSchema, {
          ...root,
          threadState: IssueComment_ThreadState.RESOLVED,
        })
      );
      await store.updateIssueComment({
        issueCommentName: root.name,
        threadState: IssueComment_ThreadState.RESOLVED,
      });
    } else {
      mocks.createIssueComment.mockResolvedValueOnce(
        create(IssueCommentSchema, {
          name: parent + "/issueComments/new",
          comment: "New comment",
        })
      );
      await store.createIssueComment({
        issueName: parent,
        comment: "New comment",
      });
    }
    const expected = store.getIssueComments(parent);
    release({ issueComments: [], nextPageToken: "" });
    await pending;
    expect(store.getIssueComments(parent)).toEqual(expected);
  }
);

test("a newer timeline fetch supersedes an older thread fetch", async () => {
  const store = createStore();
  const parent = "projects/p/issues/101";
  const oldRoot = create(IssueCommentSchema, {
    name: parent + "/issueComments/root",
    threadState: IssueComment_ThreadState.OPEN,
  });
  const newRoot = create(IssueCommentSchema, {
    ...oldRoot,
    threadState: IssueComment_ThreadState.RESOLVED,
  });
  let release!: (value: {
    issueComments: never[];
    nextPageToken: string;
  }) => void;
  mocks.listIssueComments.mockReset();
  mocks.listIssueComments
    .mockResolvedValueOnce({ issueComments: [oldRoot], nextPageToken: "" })
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          release = resolve;
        })
    )
    .mockResolvedValueOnce({ issueComments: [newRoot], nextPageToken: "" });
  const older = store.fetchIssueCommentThreads({ parent });
  await vi.waitFor(() =>
    expect(mocks.listIssueComments).toHaveBeenCalledTimes(2)
  );
  await store.fetchIssueCommentTimeline({ parent });
  release({ issueComments: [], nextPageToken: "" });
  await older;
  expect(store.getIssueComments(parent)).toEqual([newRoot]);
});

test("a failed newer fetch hands the cache back to an older fetch still in flight", async () => {
  const store = createStore();
  const parent = "projects/p/issues/100";
  const root = create(IssueCommentSchema, {
    name: parent + "/issueComments/root",
    threadState: IssueComment_ThreadState.OPEN,
  });
  let release!: (value: {
    issueComments: never[];
    nextPageToken: string;
  }) => void;
  mocks.listIssueComments.mockReset();
  mocks.listIssueComments
    // Older load: timeline page, then its reply query is held.
    .mockResolvedValueOnce({ issueComments: [root], nextPageToken: "" })
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          release = resolve;
        })
    )
    // Newer load fails on its first page.
    .mockRejectedValueOnce(new Error("network"));
  const older = store.fetchIssueCommentThreads({ parent });
  await vi.waitFor(() =>
    expect(mocks.listIssueComments).toHaveBeenCalledTimes(2)
  );
  await expect(store.fetchIssueCommentThreads({ parent })).rejects.toThrow(
    "network"
  );
  release({ issueComments: [], nextPageToken: "" });
  await older;
  expect(store.getIssueComments(parent)).toEqual([root]);
});

test.each([
  ["timeline", "resolve"],
  ["timeline", "create"],
  ["threads", "resolve"],
  ["threads", "create"],
])(
  "a fallback %s fetch preserves a successful local %s",
  async (kind, action) => {
    const store = createStore();
    const parent = "projects/p/issues/101";
    const root = create(IssueCommentSchema, {
      name: parent + "/issueComments/root",
      threadState: IssueComment_ThreadState.OPEN,
    });
    store.issueCommentsByIssue[parent] = [root];
    let finishOlder!: (value: {
      issueComments: (typeof root)[];
      nextPageToken: string;
    }) => void;
    let failNewer!: (error: Error) => void;
    mocks.listIssueComments.mockReset();
    if (kind === "threads") {
      mocks.listIssueComments.mockResolvedValueOnce({
        issueComments: [root],
        nextPageToken: "",
      });
    }
    mocks.listIssueComments
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            finishOlder = resolve;
          })
      )
      .mockImplementationOnce(
        () =>
          new Promise((_resolve, reject) => {
            failNewer = reject;
          })
      );
    const fetch = () =>
      kind === "threads"
        ? store.fetchIssueCommentThreads({ parent })
        : store.fetchIssueCommentTimeline({ parent });
    const older = fetch();
    await vi.waitFor(() => expect(finishOlder).toBeDefined());
    const newer = fetch();
    if (action === "resolve") {
      mocks.updateIssueComment.mockResolvedValueOnce(
        create(IssueCommentSchema, {
          ...root,
          threadState: IssueComment_ThreadState.RESOLVED,
        })
      );
      await store.updateIssueComment({
        issueCommentName: root.name,
        threadState: IssueComment_ThreadState.RESOLVED,
      });
    } else {
      mocks.createIssueComment.mockResolvedValueOnce(
        create(IssueCommentSchema, {
          name: parent + "/issueComments/new",
          comment: "New comment",
        })
      );
      await store.createIssueComment({
        issueName: parent,
        comment: "New comment",
      });
    }
    const expected = store.getIssueComments(parent);
    const failure = expect(newer).rejects.toThrow("network");
    failNewer(new Error("network"));
    await failure;
    finishOlder({
      issueComments: kind === "threads" ? [] : [root],
      nextPageToken: "",
    });
    await older;
    expect(store.getIssueComments(parent)).toEqual(expected);
  }
);

test("an older success is kept as a fallback and promoted when the newer fetch fails", async () => {
  const store = createStore();
  const parent = "projects/p/issues/100";
  const root = create(IssueCommentSchema, {
    name: parent + "/issueComments/root",
  });
  let releaseOlder!: (value: {
    issueComments: IssueComment[];
    nextPageToken: string;
  }) => void;
  let failNewer!: (error: Error) => void;
  mocks.listIssueComments.mockReset();
  mocks.listIssueComments
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          releaseOlder = resolve;
        })
    )
    .mockImplementationOnce(
      () =>
        new Promise((_, reject) => {
          failNewer = reject;
        })
    );
  const older = store.fetchIssueCommentTimeline({ parent });
  const newer = store.fetchIssueCommentTimeline({ parent });
  await vi.waitFor(() =>
    expect(mocks.listIssueComments).toHaveBeenCalledTimes(2)
  );
  // The older result lands first: held back, not written, not discarded.
  releaseOlder({ issueComments: [root], nextPageToken: "" });
  await older;
  expect(store.getIssueComments(parent)).toEqual([]);
  failNewer(new Error("network"));
  await expect(newer).rejects.toThrow("network");
  expect(store.getIssueComments(parent)).toEqual([root]);
});

test("mutations invalidate only their issue, including colliding IDs across projects", async () => {
  const store = createStore();
  const parent = "projects/p/issues/101";
  const other = "projects/q/issues/101";
  const oldRoot = create(IssueCommentSchema, {
    name: parent + "/issueComments/root",
    threadState: IssueComment_ThreadState.OPEN,
  });
  const otherRoot = create(IssueCommentSchema, {
    name: other + "/issueComments/root",
    comment: "Other project",
  });
  expect(parent.split("/").pop()).toBe(other.split("/").pop());
  store.issueCommentsByIssue[parent] = [oldRoot];
  let finishTarget!: (value: {
    issueComments: (typeof oldRoot)[];
    nextPageToken: string;
  }) => void;
  let finishOther!: typeof finishTarget;
  mocks.listIssueComments.mockReset();
  mocks.listIssueComments
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishTarget = resolve;
        })
    )
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishOther = resolve;
        })
    );
  const targetFetch = store.fetchIssueCommentTimeline({ parent });
  const otherFetch = store.fetchIssueCommentTimeline({ parent: other });
  const updated = create(IssueCommentSchema, {
    ...oldRoot,
    threadState: IssueComment_ThreadState.RESOLVED,
  });
  mocks.updateIssueComment.mockResolvedValueOnce(updated);
  await store.updateIssueComment({
    issueCommentName: oldRoot.name,
    threadState: IssueComment_ThreadState.RESOLVED,
  });
  finishTarget({ issueComments: [oldRoot], nextPageToken: "" });
  finishOther({ issueComments: [otherRoot], nextPageToken: "" });
  await Promise.all([targetFetch, otherFetch]);
  expect(store.getIssueComments(parent)).toEqual([updated]);
  expect(store.getIssueComments(other)).toEqual([otherRoot]);
});

test("a failed mutation does not discard an in-flight refresh", async () => {
  const store = createStore();
  const parent = "projects/p/issues/101";
  const root = create(IssueCommentSchema, {
    name: parent + "/issueComments/root",
    threadState: IssueComment_ThreadState.OPEN,
  });
  let release!: (value: {
    issueComments: (typeof root)[];
    nextPageToken: string;
  }) => void;
  mocks.listIssueComments.mockReset();
  mocks.listIssueComments.mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        release = resolve;
      })
  );
  const pending = store.fetchIssueCommentTimeline({ parent });
  mocks.updateIssueComment.mockRejectedValueOnce(new Error("Write failed"));
  await expect(
    store.updateIssueComment({
      issueCommentName: root.name,
      threadState: IssueComment_ThreadState.RESOLVED,
    })
  ).rejects.toThrow("Write failed");
  release({ issueComments: [root], nextPageToken: "" });
  await pending;
  expect(store.getIssueComments(parent)).toEqual([root]);
});

test.each(["timeline", "threads"])(
  "first %s load preserves fetched history and concurrent creation",
  async (kind) => {
    const store = createStore();
    const parent = "projects/p/issues/101";
    const old = create(IssueCommentSchema, {
      name: parent + "/issueComments/old",
      threadState: IssueComment_ThreadState.OPEN,
    });
    const added = create(IssueCommentSchema, {
      name: parent + "/issueComments/new",
      comment: "new",
    });
    const reply = create(IssueCommentSchema, {
      name: parent + "/issueComments/reply",
      root: old.name,
    });
    let finish!: (value: {
      issueComments: (typeof old)[];
      nextPageToken: string;
    }) => void;
    mocks.listIssueComments.mockReset();
    if (kind === "threads")
      mocks.listIssueComments.mockResolvedValueOnce({
        issueComments: [old],
        nextPageToken: "",
      });
    mocks.listIssueComments.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        })
    );
    const pending =
      kind === "threads"
        ? store.fetchIssueCommentThreads({ parent })
        : store.fetchIssueCommentTimeline({ parent });
    await vi.waitFor(() =>
      expect(mocks.listIssueComments).toHaveBeenCalledTimes(
        kind === "threads" ? 2 : 1
      )
    );
    mocks.createIssueComment.mockResolvedValueOnce(added);
    await store.createIssueComment({ issueName: parent, comment: "new" });
    if (kind === "threads") {
      // The reply read can already include a just-created reply: do not duplicate it.
      mocks.createIssueComment.mockResolvedValueOnce(reply);
      await store.createIssueComment({
        issueName: parent,
        root: old.name,
        comment: "reply",
      });
    }
    finish({
      issueComments: kind === "threads" ? [reply] : [old, added],
      nextPageToken: "",
    });
    await pending;
    expect(store.getIssueComments(parent)).toEqual(
      kind === "threads" ? [old, reply, added] : [old, added]
    );

    // Local write protection ends with the load; later server changes must win.
    const remote = create(IssueCommentSchema, {
      ...added,
      comment: "edited elsewhere",
    });
    mocks.listIssueComments.mockResolvedValueOnce({
      issueComments: [old, remote],
      nextPageToken: "",
    });
    await store.fetchIssueCommentTimeline({ parent });
    expect(store.getIssueComments(parent)).toEqual([old, remote]);
  }
);

test("a release build loads the timeline but never the reply pass", async () => {
  vi.resetModules();
  vi.doMock("@/utils/featureGates", () => ({
    inlineThreadsEnabled: () => false,
  }));
  const { createIssueCommentSlice: createGatedSlice } = await import(
    "./issueComment"
  );
  const state: Record<string, unknown> = {};
  const set = (updater: unknown) => {
    Object.assign(
      state,
      typeof updater === "function" ? updater(state) : updater
    );
  };
  Object.assign(
    state,
    createGatedSlice(set as never, (() => state) as never, {} as never)
  );
  const store = state as ReturnType<typeof createIssueCommentSlice>;

  const parent = "projects/p/issues/1";
  const root = create(IssueCommentSchema, {
    name: parent + "/issueComments/root",
    threadState: IssueComment_ThreadState.OPEN,
  });
  const general = create(IssueCommentSchema, {
    name: parent + "/issueComments/general",
  });
  mocks.listIssueComments.mockResolvedValueOnce({
    issueComments: [root, general],
    nextPageToken: "",
  });

  const loaded = await store.fetchIssueCommentThreads({ parent });

  // One call: the timeline. No `root in [...]` follow-up.
  expect(mocks.listIssueComments).toHaveBeenCalledTimes(1);
  expect(mocks.listIssueComments.mock.calls[0][0]).toMatchObject({
    parent,
    filter: "",
  });
  // The plain timeline still reaches every comment surface, thread roots
  // included — they simply render as ordinary comments.
  expect(loaded).toEqual([root, general]);
  expect(store.getIssueComments(parent)).toEqual(loaded);

  vi.doUnmock("@/utils/featureGates");
  vi.resetModules();
});
