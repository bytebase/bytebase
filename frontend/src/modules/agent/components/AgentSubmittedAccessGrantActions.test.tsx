import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/app/router/handles";
import {
  AccessGrantSchema,
  AccessGrant_Status,
} from "@/types/proto-es/v1/access_grant_service_pb";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import { createSubmittedAccessGrant } from "../logic/submittedAccessGrant";
import type { AgentSubmittedAccessGrant } from "../logic/types";
import { AgentSubmittedAccessGrantActions } from "./AgentSubmittedAccessGrantActions";

const mocks = vi.hoisted(() => ({
  searchMyAccessGrants: vi.fn(),
  getIssue: vi.fn(),
  push: vi.fn(),
  minimize: vi.fn(),
  agent: {} as Record<string, unknown>,
  app: {} as Record<string, unknown>,
  route: {} as Record<string, unknown>,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, values?: { time?: string }) =>
      values?.time ? `${key}: ${values.time}` : key,
  }),
}));
vi.mock("@/api", () => ({
  accessGrantServiceClientConnect: {
    searchMyAccessGrants: mocks.searchMyAccessGrants,
  },
  issueServiceClientConnect: { getIssue: mocks.getIssue },
}));
vi.mock("@/app/router", () => ({
  router: {
    push: mocks.push,
    get currentRoute() {
      return { value: mocks.route };
    },
  },
}));
vi.mock("../store/agent", () => {
  const useAgentStore = <T,>(selector: (state: typeof mocks.agent) => T) =>
    selector(mocks.agent);
  useAgentStore.getState = () => mocks.agent;
  return { useAgentStore };
});
vi.mock("@/stores/app", () => {
  const useAppStore = <T,>(selector: (state: typeof mocks.app) => T) =>
    selector(mocks.app);
  useAppStore.getState = () => mocks.app;
  return { useAppStore };
});
const grant = create(AccessGrantSchema, {
  name: "projects/demo/accessGrants/42",
  creator: "users/member@example.com",
  status: AccessGrant_Status.PENDING,
  issue: "projects/demo/issues/7",
  targets: ["instances/sample/databases/app"],
  query: "SELECT id FROM sample",
  expiration: { case: "ttl", value: { seconds: 3600n, nanos: 0 } },
});

let receipt: AgentSubmittedAccessGrant;

const show = () =>
  render(
    <AgentSubmittedAccessGrantActions
      chatId="chat-1"
      messageId="message-1"
      receipt={receipt}
    />
  );
const checkStatus = () =>
  fireEvent.click(
    screen.getByRole("button", { name: "agent.submitted-access-grant.check-status" })
  );

beforeEach(() => {
  vi.clearAllMocks();
  receipt = createSubmittedAccessGrant(grant, Date.now())!;
  mocks.agent = {
    workspaceName: "workspaces/demo",
    userEmail: "member@example.com",
    currentChatId: "chat-1",
    visible: true,
    minimized: false,
    chats: [
      { id: "chat-1", workspaceName: "workspaces/demo", archived: false },
    ],
    messagesByChatId: {
      "chat-1": [
        { id: "message-1", metadata: { submittedAccessGrants: [receipt], accessRequestDraft: {project: "projects/demo", sourceDatabase: grant.targets[0], targets: grant.targets, statement: grant.query} } },
      ],
    },
    getChat: (chatId: string) =>
      (mocks.agent.chats as Array<{ id: string }>).find(({ id }) => id === chatId),
    captureApiScope: () => () => true,
    recordActionObservation: vi.fn().mockReturnValue({}),
    minimize: mocks.minimize,
  };
  mocks.app = {
    workspace: { name: "workspaces/demo" },
    currentUser: {
      email: "member@example.com",
      workspace: "workspaces/demo",
    },
  };
  mocks.route = { name: "initial", params: {}, fullPath: "/initial" };
  mocks.push.mockImplementation(async (route) => {
    mocks.route = { ...route, fullPath: "/destination" };
  });
  mocks.searchMyAccessGrants.mockResolvedValue({ accessGrants: [grant] });
  mocks.getIssue.mockResolvedValue(
    create(IssueSchema, { name: grant.issue, status: IssueStatus.OPEN })
  );
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("AgentSubmittedAccessGrantActions", () => {
  test("shows the submission snapshot without making a request", () => {
    show();
    expect(screen.getByText("Pending")).toBeDefined();
    expect(screen.getByText(/agent.submitted-access-grant.observed-at/)).toBeDefined();
    expect(screen.getByRole("button", { name: "agent.submitted-access-grant.open" })).toBeDefined();
    expect(mocks.searchMyAccessGrants).not.toHaveBeenCalled();
    expect(mocks.getIssue).not.toHaveBeenCalled();
  });

  test("keeps idle and loading icons at the small action size", async () => {
    let resolve: ((value: { accessGrants: typeof grant[] }) => void) | undefined;
    mocks.searchMyAccessGrants.mockImplementation(
      () => new Promise((done) => { resolve = done; })
    );
    show();
    const button = screen.getByRole("button", {
      name: "agent.submitted-access-grant.check-status",
    });
    const expectSmallIcon = () => {
      const icon = button.querySelector("svg")!;
      expect(icon.classList.contains("size-4")).toBe(true);
      expect(icon.classList.contains("shrink-0")).toBe(true);
      expect(button.classList.contains("gap-1.5")).toBe(true);
      return icon;
    };
    expect(expectSmallIcon().classList.contains("animate-spin")).toBe(false);
    checkStatus();
    expect(expectSmallIcon().classList.contains("animate-spin")).toBe(true);
    await act(async () => resolve?.({ accessGrants: [grant] }));
    expect(expectSmallIcon().classList.contains("animate-spin")).toBe(false);
  });

  test("checks the caller's exact grant only after a member click", async () => {
    mocks.getIssue.mockResolvedValue(
      create(IssueSchema, {
        name: grant.issue,
        approvalStatus: ApprovalStatus.REJECTED,
        status: IssueStatus.OPEN,
      })
    );
    show();
    await act(async () => checkStatus());
    expect(mocks.searchMyAccessGrants).toHaveBeenCalledTimes(1);
    expect(mocks.searchMyAccessGrants.mock.calls[0]?.[0]).toMatchObject({
      parent: "projects/demo",
      pageSize: 1,
      filter: 'name == "projects/demo/accessGrants/42"',
    });
    expect(mocks.getIssue.mock.calls[0]?.[0]).toMatchObject({ name: grant.issue });
    expect(screen.getByText("Rejected")).toBeDefined();
    expect(mocks.push).not.toHaveBeenCalled();
  });

  test("uses a fresh result on each explicit check and never checks automatically", async () => {
    show();
    await act(async () => checkStatus());
    mocks.searchMyAccessGrants.mockResolvedValue({
      accessGrants: [create(AccessGrantSchema, { ...grant, status: AccessGrant_Status.REVOKED })],
    });
    await act(async () => checkStatus());
    expect(mocks.searchMyAccessGrants).toHaveBeenCalledTimes(2);
    expect(screen.getByText("Revoked")).toBeDefined();
    expect(mocks.getIssue).toHaveBeenCalledTimes(1);
  });

  test("does not claim Pending when the associated issue cannot be read", async () => {
    mocks.getIssue.mockRejectedValue(new Error("unavailable"));
    show();
    await act(async () => checkStatus());
    expect(screen.getByRole("alert").textContent).toBe("agent.submitted-access-grant.status-unavailable");
    expect(screen.queryByText("Pending")).toBeNull();
    expect(screen.getByRole("button", { name: "agent.submitted-access-grant.open" })).toBeDefined();
  });

  test.each(["missing", "other-name", "other-creator", "other-issue"])(
    "keeps status unverified for a %s grant result",
    async (kind) => {
      mocks.searchMyAccessGrants.mockResolvedValue({
        accessGrants: kind === "missing" ? [] : [create(AccessGrantSchema, {
          ...grant,
          ...(kind === "other-name" ? { name: "projects/demo/accessGrants/43" } : {}),
          ...(kind === "other-creator" ? { creator: "users/other@example.com" } : {}),
          ...(kind === "other-issue" ? { issue: "projects/other/issues/7" } : {}),
        })],
      });
      show();
      await act(async () => checkStatus());
      expect(screen.getByRole("alert").textContent).toBe("agent.submitted-access-grant.status-unavailable");
      expect(mocks.getIssue).not.toHaveBeenCalled();
    }
  );

  test("opens the existing approval issue without any grant or SQL API calls", async () => {
    show();
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "agent.submitted-access-grant.open" })));
    expect(mocks.push).toHaveBeenCalledWith({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: { projectId: "demo", issueId: "7" },
    });
    expect(mocks.searchMyAccessGrants).not.toHaveBeenCalled();
    expect(mocks.getIssue).not.toHaveBeenCalled();
    expect(mocks.minimize).toHaveBeenCalledOnce();
  });

  test("keeps a receipt without an issue in the chat while status checks remain available", async () => {
    const grantWithoutIssue = create(AccessGrantSchema, {
      ...grant,
      status: AccessGrant_Status.ACTIVE,
      issue: "",
      expiration: {
        case: "expireTime",
        value: timestampFromDate(new Date(Date.now() + 3_600_000)),
      },
    });
    receipt.grant = createSubmittedAccessGrant(grantWithoutIssue, receipt.submittedAt)!.grant;
    mocks.searchMyAccessGrants.mockResolvedValue({ accessGrants: [grantWithoutIssue] });
    show();
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "agent.submitted-access-grant.open" })));
    expect(screen.getByRole("alert").textContent).toBe("agent.submitted-access-grant.open-unavailable");
    expect(mocks.push).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    expect(mocks.searchMyAccessGrants).not.toHaveBeenCalled();
    expect(mocks.getIssue).not.toHaveBeenCalled();

    await act(async () => checkStatus());
    expect(mocks.searchMyAccessGrants).toHaveBeenCalledOnce();
    expect(screen.getByText("Active")).toBeDefined();
    expect(screen.queryByRole("alert")).toBeNull();
    expect(mocks.push).not.toHaveBeenCalled();
  });

  test.each(["identity", "archive", "message"])(
    "ignores a late check after the originating %s changes",
    async (kind) => {
      let resolve: ((value: { accessGrants: typeof grant[] }) => void) | undefined;
      mocks.searchMyAccessGrants.mockImplementation(() => new Promise((done) => { resolve = done; }));
      show();
      checkStatus();
      if (kind === "identity") mocks.app.currentUser = { email: "other@example.com", workspace: "workspaces/demo" };
      if (kind === "archive") (mocks.agent.chats as Array<{ archived: boolean }>)[0]!.archived = true;
      if (kind === "message") mocks.agent.messagesByChatId = { "chat-1": [] };
      await act(async () => resolve?.({ accessGrants: [grant] }));
      expect(mocks.getIssue).not.toHaveBeenCalled();
      expect(mocks.push).not.toHaveBeenCalled();
    }
  );

  test("does not start an issue read after the actions have unmounted", async () => {
    let resolve: ((value: { accessGrants: typeof grant[] }) => void) | undefined;
    mocks.searchMyAccessGrants.mockImplementation(() => new Promise((done) => { resolve = done; }));
    const { unmount } = show();
    checkStatus();
    unmount();
    await act(async () => resolve?.({ accessGrants: [grant] }));
    expect(mocks.getIssue).not.toHaveBeenCalled();
  });

  test("allows only one in-flight status check", async () => {
    let resolve: ((value: { accessGrants: typeof grant[] }) => void) | undefined;
    mocks.searchMyAccessGrants.mockImplementation(() => new Promise((done) => { resolve = done; }));
    show();
    checkStatus();
    checkStatus();
    expect(mocks.searchMyAccessGrants).toHaveBeenCalledOnce();
    expect((screen.getByRole("button", { name: "agent.submitted-access-grant.check-status" }) as HTMLButtonElement).disabled).toBe(true);
    await act(async () => resolve?.({ accessGrants: [grant] }));
    expect((screen.getByRole("button", { name: "agent.submitted-access-grant.check-status" }) as HTMLButtonElement).disabled).toBe(false);
  });

  test("reports a denied read without retrying or changing the receipt", async () => {
    mocks.searchMyAccessGrants.mockRejectedValue(new Error("permission denied"));
    const original = JSON.stringify(receipt);
    show();
    await act(async () => checkStatus());
    expect(screen.getByRole("alert").textContent).toBe("agent.submitted-access-grant.status-unavailable");
    expect(mocks.searchMyAccessGrants).toHaveBeenCalledOnce();
    expect(mocks.getIssue).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    expect(JSON.stringify(receipt)).toBe(original);
  });

  test("updates local expiration without making a status request", async () => {
    vi.useFakeTimers();
    receipt.grant = createSubmittedAccessGrant(create(AccessGrantSchema, {
      ...grant,
      status: AccessGrant_Status.ACTIVE,
      expiration: {
        case: "expireTime",
        value: timestampFromDate(new Date(Date.now() + 1000)),
      },
    }), receipt.submittedAt)!.grant;
    show();
    expect(screen.getByText("Active")).toBeDefined();
    await act(async () => vi.advanceTimersByTime(1001));
    expect(screen.getByText("Expired")).toBeDefined();
    expect(mocks.searchMyAccessGrants).not.toHaveBeenCalled();
    expect(mocks.getIssue).not.toHaveBeenCalled();
  });

  test("does not dispatch actions after the app identity changes", async () => {
    show();
    mocks.app.currentUser = { email: "other@example.com", workspace: "workspaces/demo" };
    await act(async () => {
      checkStatus();
      fireEvent.click(screen.getByRole("button", { name: "agent.submitted-access-grant.open" }));
    });
    expect(mocks.searchMyAccessGrants).not.toHaveBeenCalled();
    expect(mocks.push).not.toHaveBeenCalled();
  });

  test("keeps the chat open when navigation does not reach the native request", async () => {
    mocks.push.mockResolvedValue(undefined);
    show();
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "agent.submitted-access-grant.open" })));
    expect(mocks.minimize).not.toHaveBeenCalled();
    expect(screen.getByRole("alert").textContent).toBe("agent.submitted-access-grant.open-unavailable");
  });

  test("does not display an unsupported grant status as a verified status", () => {
    receipt.grant = createSubmittedAccessGrant(create(AccessGrantSchema, {
      ...grant,
      status: AccessGrant_Status.STATUS_UNSPECIFIED,
    }), receipt.submittedAt)!.grant;
    show();
    expect(screen.queryByText("UNKNOWN")).toBeNull();
    expect(screen.getByRole("alert").textContent).toBe("agent.submitted-access-grant.status-unavailable");
    expect(mocks.searchMyAccessGrants).not.toHaveBeenCalled();
  });

  test("ignores a late result after an app scope revision, even when the same actor returns", async () => {
    let sameScope = true;
    mocks.agent.captureApiScope = () => () => sameScope;
    let resolve: ((value: { accessGrants: typeof grant[] }) => void) | undefined;
    mocks.searchMyAccessGrants.mockImplementation(() => new Promise((done) => { resolve = done; }));
    show();
    checkStatus();
    sameScope = false;
    await act(async () => resolve?.({ accessGrants: [grant] }));
    expect(mocks.getIssue).not.toHaveBeenCalled();
    expect((screen.getByRole("button", { name: "agent.submitted-access-grant.check-status" }) as HTMLButtonElement).disabled).toBe(false);
  });
});
