import { beforeEach, describe, expect, test, vi } from "vitest";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/app/router/handles";
import {
  applyPreparedIssueCommentDraft,
  consumePreparedIssueCommentDraft,
  ExistingIssueCommentDraftError,
  IssueCommentDraftPermissionError,
  IssueCommentDraftStorageError,
  issueCommentDraftKey,
  markIssueCommentDraftDelivered,
  prepareIssueCommentDraftRoute,
  readPreparedIssueCommentDraft,
  restoreIssueCommentDraft,
  wasIssueCommentDraftDelivered,
} from "./issueCommentDraft";

const ISSUE = "projects/sample/issues/101";
const access = vi.hoisted(() => ({
  state: {
    projectsByName: {} as Record<string, { name: string }>,
    projectPoliciesByName: {} as Record<string, object>,
    workspacePolicy: undefined as object | undefined,
    roles: [] as object[],
    currentUser: undefined as { name: string } | undefined,
  },
  canCreate: vi.fn(() => true),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => access.state },
}));
vi.mock("@/utils/iam/permission", () => ({
  hasProjectPermissionV2: access.canCreate,
}));

beforeEach(() => {
  localStorage.clear();
  sessionStorage.clear();
  access.state.projectsByName = {};
  access.state.projectPoliciesByName = {};
  access.state.workspacePolicy = undefined;
  access.state.roles = [];
  access.state.currentUser = undefined;
  access.canCreate.mockReset().mockReturnValue(true);
});

describe("prepareIssueCommentDraftRoute", () => {
  test("stores an unsent comment and opens the exact Issue review editor", () => {
    const route = prepareIssueCommentDraftRoute(
      ISSUE,
      "**Security review**\n\nConfirm the grant duration."
    );

    expect(route).toMatchObject({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: { projectId: "sample", issueId: "101" },
      query: { phase: "review" },
      hash: "#issue-comment-editor",
    });
    const key = route.query?.agentCommentDraft as string;
    expect(key).toEqual(expect.any(String));
    expect(localStorage.getItem(issueCommentDraftKey(ISSUE))).toBeNull();
    expect(readPreparedIssueCommentDraft(key, ISSUE)).toBe(
      "**Security review**\n\nConfirm the grant duration."
    );
    expect(sessionStorage.getItem(key)).not.toBeNull();
    expect(consumePreparedIssueCommentDraft(key, ISSUE)).toBe(
      "**Security review**\n\nConfirm the grant duration."
    );
    expect(consumePreparedIssueCommentDraft(key, ISSUE)).toBeUndefined();
  });

  test("records delivery only after the exact prepared comment is displayed", () => {
    const route = prepareIssueCommentDraftRoute(ISSUE, "Agent review");
    const key = route.query?.agentCommentDraft as string;

    expect(wasIssueCommentDraftDelivered(key)).toBe(false);
    markIssueCommentDraftDelivered(key, ISSUE, "Different comment");
    markIssueCommentDraftDelivered(
      key,
      "projects/sample/issues/102",
      "Agent review"
    );
    expect(wasIssueCommentDraftDelivered(key)).toBe(false);
    markIssueCommentDraftDelivered(key, ISSUE, "Agent review");
    expect(wasIssueCommentDraftDelivered(key)).toBe(true);
  });

  test("does not overwrite an existing unsent comment", () => {
    localStorage.setItem(issueCommentDraftKey(ISSUE), "My unfinished review");

    expect(() =>
      prepareIssueCommentDraftRoute(ISSUE, "Agent suggestion")
    ).toThrow(ExistingIssueCommentDraftError);
    expect(localStorage.getItem(issueCommentDraftKey(ISSUE))).toBe(
      "My unfinished review"
    );
  });

  test.each([
    ["projects/-/issues/101", "Review"],
    ["projects/sample/issues/-", "Review"],
    ["projects/other/issues/101/extra", "Review"],
    [ISSUE, "  \n  "],
  ])("rejects invalid target or empty comment", (issue, comment) => {
    expect(() => prepareIssueCommentDraftRoute(issue, comment)).toThrow();
    expect(sessionStorage.length).toBe(0);
  });

  test("does not return a prepared draft for another Issue", () => {
    const route = prepareIssueCommentDraftRoute(ISSUE, "Review");
    const key = route.query?.agentCommentDraft as string;

    expect(
      consumePreparedIssueCommentDraft(key, "projects/sample/issues/102")
    ).toBeUndefined();
    expect(consumePreparedIssueCommentDraft(key, ISSUE)).toBe("Review");
  });

  test("applies a one-time draft only after it is persisted", () => {
    const route = prepareIssueCommentDraftRoute(ISSUE, "Agent review");
    const key = route.query?.agentCommentDraft as string;

    expect(applyPreparedIssueCommentDraft(key, ISSUE)).toBe("Agent review");
    expect(localStorage.getItem(issueCommentDraftKey(ISSUE))).toBe(
      "Agent review"
    );
    expect(sessionStorage.getItem(key)).toBeNull();
    expect(applyPreparedIssueCommentDraft(key, ISSUE)).toBeUndefined();
  });

  test("keeps a prepared draft when restoring it cannot persist", () => {
    const route = prepareIssueCommentDraftRoute(ISSUE, "Agent review");
    const key = route.query?.agentCommentDraft as string;
    const setItem = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new Error("Storage unavailable");
      });
    try {
      expect(restoreIssueCommentDraft(ISSUE, key)).toBe("Agent review");
      expect(sessionStorage.getItem(key)).not.toBeNull();
    } finally {
      setItem.mockRestore();
    }
    expect(restoreIssueCommentDraft(ISSUE, key)).toBe("Agent review");
  });

  test("keeps the prepared draft if another comment or storage failure blocks applying it", () => {
    const route = prepareIssueCommentDraftRoute(ISSUE, "Agent review");
    const key = route.query?.agentCommentDraft as string;
    localStorage.setItem(issueCommentDraftKey(ISSUE), "My review");

    expect(applyPreparedIssueCommentDraft(key, ISSUE)).toBeUndefined();
    expect(localStorage.getItem(issueCommentDraftKey(ISSUE))).toBe("My review");
    expect(sessionStorage.getItem(key)).not.toBeNull();

    localStorage.removeItem(issueCommentDraftKey(ISSUE));
    const setItem = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new Error("Storage disabled");
      });
    try {
      expect(() => applyPreparedIssueCommentDraft(key, ISSUE)).toThrow();
      expect(sessionStorage.getItem(key)).not.toBeNull();
    } finally {
      setItem.mockRestore();
    }
  });

  test("reports storage failure without navigating to an empty editor", () => {
    const setItem = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new Error("Storage disabled");
      });
    try {
      expect(() => prepareIssueCommentDraftRoute(ISSUE, "Review")).toThrow(
        IssueCommentDraftStorageError
      );
    } finally {
      setItem.mockRestore();
    }
  });

  test("does not offer a draft when a loaded Project denies commenting", () => {
    access.state.projectsByName = {
      "projects/sample": { name: "projects/sample" },
    };
    access.state.projectPoliciesByName = { "projects/sample": {} };
    access.state.workspacePolicy = {};
    access.state.roles = [{}];
    access.state.currentUser = { name: "users/1" };
    access.canCreate.mockReturnValue(false);

    expect(() => prepareIssueCommentDraftRoute(ISSUE, "Review")).toThrow(
      IssueCommentDraftPermissionError
    );
    expect(access.canCreate).toHaveBeenCalledOnce();
    expect(sessionStorage.length).toBe(0);
  });

  test("fails open when the Project permission state is incomplete", () => {
    access.state.projectsByName = {
      "projects/sample": { name: "projects/sample" },
    };
    access.canCreate.mockReturnValue(false);

    expect(prepareIssueCommentDraftRoute(ISSUE, "Review").name).toBe(
      PROJECT_V1_ROUTE_ISSUE_DETAIL
    );
    expect(access.canCreate).not.toHaveBeenCalled();
  });
});
