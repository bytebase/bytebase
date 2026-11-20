import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { lazyExtractDomRefSuggestions, lazyPrepareDomAction } from "../../dom";
import {
  type AgentDomContext,
  registerAgentDomOwner,
} from "../../dom/interactionRegistry";
import type { PreparedApiCall } from "./callApi";

const mocks = vi.hoisted(() => ({
  checkPreparedApiPermission: vi.fn(),
  dispatchPreparedApiCall: vi.fn(),
  validateIssueDecisionPermission: vi.fn(),
  workspace: "workspaces/example",
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      currentUser: { name: "users/1" },
      workspaceResourceName: () => mocks.workspace,
    }),
  },
}));

vi.mock("./apiPermissionPreflight", () => ({
  checkPreparedApiPermission: mocks.checkPreparedApiPermission,
}));

vi.mock("./callApi", () => ({
  dispatchPreparedApiCall: mocks.dispatchPreparedApiCall,
}));
vi.mock("./issueDecisionPermission", () => ({
  validateIssueDecisionPermission: mocks.validateIssueDecisionPermission,
}));

import {
  createPendingApiCall,
  getPendingApiCallPreview,
  resolvePendingApiCall,
} from "./pendingApiCall";

const makeCall = (): PreparedApiCall => ({
  operationId: "bytebase.v1.ProjectService.UpdateProject",
  path: "/bytebase.v1.ProjectService/UpdateProject",
  body: JSON.stringify({ project: { name: "projects/demo", title: "Before" } }),
});

const disposers: (() => void)[] = [];
afterEach(() => {
  for (const dispose of disposers.splice(0)) dispose();
  document.body.replaceChildren();
});
async function domFixture(body?: Record<string, unknown>) {
  const ctx: AgentDomContext = {
    actor: "users/1",
    workspace: "workspaces/example",
    chatId: "chat",
    runId: "run",
    runMode: "default",
    pageKey: "/db",
    pageGeneration: 1,
    isCurrent: () => true,
  };
  const snapshot = {
    pageKey: ctx.pageKey,
    resourceName: "instances/i/databases/db",
    projectName: "projects/p",
    revision: "old",
  };
  const button = document.createElement("button");
  button.textContent = "Save";
  document.body.append(button);
  const validateFresh = vi.fn(async () => true),
    refresh = vi.fn(async () => {});
  const call = {
    operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
    path: "/bytebase.v1.DatabaseService/UpdateDatabase",
    body: JSON.stringify(
      body ?? {
        database: { name: snapshot.resourceName, labels: { tier: "test" } },
        updateMask: "labels",
      }
    ),
  };
  const dispose = registerAgentDomOwner({
    ownerId: "labels",
    actor: ctx.actor,
    workspace: ctx.workspace,
    getSnapshot: () => snapshot,
    refresh,
    getBinding: (el, verb) =>
      el === button && verb === "click"
        ? {
            id: "save",
            verb,
            effect: "persist",
            eligible: () => true,
            validateFresh,
            prepare: async () => ({
              call,
              preview: { kind: "database-labels", changes: [] },
            }),
          }
        : undefined,
  });
  disposers.push(dispose);
  const [ref] = await lazyExtractDomRefSuggestions(ctx);
  const prepared = await lazyPrepareDomAction(
    { type: "click", ref: ref.ref },
    ctx
  );
  if (prepared.kind !== "confirm-api")
    throw new Error("expected a real registered origin");
  const identity = {
    chatId: ctx.chatId,
    toolCallId: "original-dom",
    userName: ctx.actor,
  };
  const mint = () =>
    createPendingApiCall({
      ...identity,
      call: prepared.call,
      domOrigin: prepared.origin,
      domContext: ctx,
    });
  return {
    ctx,
    snapshot,
    button,
    dispose,
    refresh,
    validateFresh,
    prepared,
    identity,
    mint,
  };
}

describe("pendingApiCall", () => {
  test("a dismissed stale preview reports not called rather than user denial", async () => {
    const f = await domFixture(),
      { actionId } = f.mint();
    f.snapshot.revision = "new";
    expect(
      JSON.parse(
        await resolvePendingApiCall({
          ...f.identity,
          actionId,
          domContext: f.ctx,
          approved: false,
        })
      )
    ).toMatchObject({
      called: false,
      error: expect.stringMatching(/^Not called:/),
    });
    expect(f.validateFresh).not.toHaveBeenCalled();
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
  });
  test.each(["expiry", "revision", "unchanged"])(
    "rechecks %s at the real fetch boundary and sends only the approved request",
    async (change) => {
      const f = await domFixture(),
        { actionId, expiresAt } = f.mint();
      const sender =
        await vi.importActual<typeof import("./callApi")>("./callApi");
      const fetch = vi.fn(async () => new Response("{}", { status: 200 }));
      vi.stubGlobal("fetch", fetch);
      mocks.dispatchPreparedApiCall.mockImplementationOnce(
        (...args: Parameters<typeof sender.dispatchPreparedApiCall>) => {
          if (change === "expiry")
            vi.spyOn(Date, "now").mockReturnValue(expiresAt);
          if (change === "revision") f.snapshot.revision = "new";
          if (change === "unchanged")
            f.button.setAttribute("title", "unrelated");
          return sender.dispatchPreparedApiCall(...args);
        }
      );
      try {
        const args = {
          ...f.identity,
          actionId,
          domContext: f.ctx,
          approved: true,
        };
        const result = JSON.parse(await resolvePendingApiCall(args));
        expect(mocks.dispatchPreparedApiCall).toHaveBeenCalledOnce();
        expect(fetch).toHaveBeenCalledTimes(change === "unchanged" ? 1 : 0);
        expect(result).toMatchObject(
          change === "unchanged" ? { status: 200 } : { called: false }
        );
        if (change === "unchanged")
          expect(fetch).toHaveBeenCalledExactlyOnceWith(
            expect.stringContaining(f.prepared.call.path),
            expect.objectContaining({ body: f.prepared.call.body })
          );
      } finally {
        vi.restoreAllMocks();
        vi.unstubAllGlobals();
      }
    }
  );
  test.each(["owner gone", "new account", "response failed"])(
    "skips owner refresh when %s without discarding the known response",
    async (change) => {
      const f = await domFixture(),
        { actionId } = f.mint();
      let current = true;
      const status = change === "response failed" ? 403 : 200;
      mocks.dispatchPreparedApiCall.mockImplementationOnce(async () => {
        if (change === "owner gone") f.dispose();
        if (change === "new account") current = false;
        return JSON.stringify({ status, response: { known: true } });
      });
      const args = {
        ...f.identity,
        actionId,
        domContext: { ...f.ctx, isCurrent: () => current },
        approved: true,
      };
      expect(JSON.parse(await resolvePendingApiCall(args))).toEqual({
        status,
        response: { known: true },
      });
      expect(f.refresh).not.toHaveBeenCalled();
      expect(mocks.dispatchPreparedApiCall).toHaveBeenCalledTimes(1);
    }
  );
  test.each([
    "no context",
    "run",
    "actor",
    "workspace",
    "chat",
    "page",
    "generation",
    "revision",
    "removed element",
    "unmounted owner",
    "cancelled",
    "revoked",
    "owner race",
    "IAM race",
    "expired during IAM",
    "denied",
  ])("never sends a DOM request after %s", async (reason) => {
    const f = await domFixture();
    const { actionId, expiresAt } = f.mint();
    const controller = new AbortController();
    let domContext: AgentDomContext | undefined = {
      ...f.ctx,
      signal: controller.signal,
    };
    if (reason === "no context") domContext = undefined;
    if (reason === "run") domContext = { ...f.ctx, runId: "different" };
    if (reason === "actor") domContext = { ...f.ctx, actor: "users/other" };
    if (reason === "workspace")
      domContext = { ...f.ctx, workspace: "workspaces/other" };
    if (reason === "chat") domContext = { ...f.ctx, chatId: "other" };
    if (reason === "page") domContext = { ...f.ctx, pageKey: "/other" };
    if (reason === "generation") domContext = { ...f.ctx, pageGeneration: 2 };
    if (reason === "revision") f.snapshot.revision = "new";
    if (reason === "removed element")
      f.button.replaceWith(f.button.cloneNode(true));
    if (reason === "unmounted owner") f.dispose();
    if (reason === "cancelled") controller.abort();
    if (reason === "revoked") f.validateFresh.mockResolvedValue(false);
    if (reason === "owner race")
      f.validateFresh.mockImplementationOnce(async () => {
        f.snapshot.revision = "new";
        return true;
      });
    if (reason === "IAM race")
      mocks.checkPreparedApiPermission.mockImplementationOnce(async () => {
        f.snapshot.revision = "new";
        return { kind: "allowed" };
      });
    if (reason === "expired during IAM") {
      vi.useFakeTimers();
      mocks.checkPreparedApiPermission.mockImplementationOnce(async () => {
        vi.setSystemTime(expiresAt);
        return { kind: "allowed" };
      });
    }
    const args = {
      ...f.identity,
      actionId,
      domContext,
      signal: controller.signal,
      approved: reason !== "denied",
    };
    expect(JSON.parse(await resolvePendingApiCall(args))).toHaveProperty(
      "error"
    );
    expect(JSON.parse(await resolvePendingApiCall(args)).error).toMatch(
      /not found|already used/i
    );
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
    expect(f.refresh).not.toHaveBeenCalled();
  });
  test.each([
    { database: { name: "instances/i/databases/other" }, updateMask: "labels" },
    {
      database: { name: "instances/i/databases/db" },
      updateMask: "environment",
    },
    {
      database: { name: "instances/i/databases/db" },
      updateMask: "labels,environment",
    },
  ])(
    "refuses a mismatched database target or mask at creation",
    async (body) => {
      const f = await domFixture(body);
      expect(f.mint).toThrow();
    }
  );
  test("refuses a substituted frozen call with the same target and mask", async () => {
    const f = await domFixture();
    expect(() =>
      createPendingApiCall({
        ...f.identity,
        call: {
          ...f.prepared.call,
          body: f.prepared.call.body.replace("test", "other"),
        },
        domOrigin: f.prepared.origin,
        domContext: f.ctx,
      })
    ).toThrow();
  });
  test("does_not_replay_successful_write_when_refresh_fails", async () => {
    const f = await domFixture();
    const { actionId } = f.mint();
    mocks.dispatchPreparedApiCall.mockImplementationOnce(async () => {
      f.snapshot.revision = "newer draft";
      return '{"status":200,"response":{"known":true}}';
    });
    f.refresh.mockRejectedValue(new Error("private data"));
    const args = {
      ...f.identity,
      actionId,
      domContext: { ...f.ctx },
      approved: true,
    };
    expect(JSON.parse(await resolvePendingApiCall(args))).toEqual({
      status: 200,
      response: { known: true },
      domRefresh: "failed",
    });
    expect(JSON.parse(await resolvePendingApiCall(args)).error).toMatch(/used/);
    expect(f.refresh).toHaveBeenCalledOnce();
    expect(mocks.dispatchPreparedApiCall).toHaveBeenCalledTimes(1);
  });
  test.each(["valid", "malformed"])(
    "consumes an approved %s diagnostic action without dispatch",
    async (kind) => {
      const { actionId } = createPendingApiCall({
        chatId: "diagnostic",
        toolCallId: "pending",
        userName: "users/1",
        call: makeCall(),
      });
      const capability = {
        kind: "diagnostic" as const,
        resource: {
          kind: "plan" as const,
          name: "projects/demo/plans/1",
          project: kind === "valid" ? "projects/demo" : "projects/other",
          label: "Plan",
        },
      };
      const result = await resolvePendingApiCall({
        actionId,
        chatId: "diagnostic",
        toolCallId: "pending",
        userName: "users/1",
        approved: true,
        capability,
      });
      expect(JSON.parse(result).error).toMatch(/diagnostic/i);
      expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
      expect(mocks.checkPreparedApiPermission).not.toHaveBeenCalled();
      const replay = await resolvePendingApiCall({
        actionId,
        chatId: "diagnostic",
        toolCallId: "pending",
        userName: "users/1",
        approved: true,
      });
      expect(JSON.parse(replay).error).toMatch(/not found|used/i);
    }
  );

  beforeEach(() => {
    vi.useRealTimers();
    mocks.workspace = "workspaces/example";
    mocks.checkPreparedApiPermission.mockReset();
    mocks.checkPreparedApiPermission.mockResolvedValue({ kind: "allowed" });
    mocks.dispatchPreparedApiCall.mockReset();
    mocks.dispatchPreparedApiCall.mockResolvedValue(
      JSON.stringify({ status: 200, response: {} })
    );
    mocks.validateIssueDecisionPermission.mockReset();
    mocks.validateIssueDecisionPermission.mockResolvedValue(true);
  });

  test("does not use a DOM permission grant in a different signed-in workspace", async () => {
    const f = await domFixture();
    const { actionId } = f.mint();
    mocks.workspace = "workspaces/other";
    const identity = { ...f.identity, domContext: f.ctx, actionId };
    expect(getPendingApiCallPreview(identity)).toBeUndefined();
    const result = await resolvePendingApiCall({
      ...identity,
      approved: true,
    });
    expect(JSON.parse(result)).toMatchObject({
      called: false,
      error: expect.stringMatching(/^Not called:/),
    });
    expect(mocks.checkPreparedApiPermission).not.toHaveBeenCalled();
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("rechecks access after approval before sending the target request", async () => {
    const { actionId } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-revoked",
      userName: "users/1",
      call: makeCall(),
    });
    mocks.checkPreparedApiPermission.mockResolvedValue({
      kind: "denied",
      result: JSON.stringify({
        status: 403,
        error: "Not called: bb.projects.update is not granted.",
      }),
    });
    const result = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-revoked",
      userName: "users/1",
      approved: true,
    });
    expect(JSON.parse(result)).toMatchObject({ status: 403 });
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("does not create a permission request without a signed-in principal", () => {
    expect(() =>
      createPendingApiCall({
        chatId: "chat-1",
        toolCallId: "tool-1",
        userName: "",
        call: makeCall(),
      })
    ).toThrow(/user/i);
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("does not dispatch before approval and uses the prepared snapshot once", async () => {
    const call = makeCall();
    const expected = { ...call };
    const { actionId } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call,
    });
    (call as { body: string }).body = JSON.stringify({ changed: true });

    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
    const result = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });

    expect(JSON.parse(result)).toEqual({ status: 200, response: {} });
    expect(mocks.dispatchPreparedApiCall).toHaveBeenCalledExactlyOnceWith(
      expected
    );
    await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    expect(mocks.dispatchPreparedApiCall).toHaveBeenCalledTimes(1);
  });

  test("previews only the matching, unexpired action without consuming it", async () => {
    vi.useFakeTimers();
    const call = makeCall();
    const { actionId, expiresAt } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call,
    });

    expect(
      getPendingApiCallPreview({
        actionId,
        chatId: "chat-1",
        toolCallId: "tool-1",
        userName: "users/1",
      })
    ).toEqual({
      operationId: call.operationId,
      body: call.body,
      expiresAt,
    });
    expect(
      getPendingApiCallPreview({
        actionId,
        chatId: "chat-2",
        toolCallId: "tool-1",
        userName: "users/1",
      })
    ).toBeUndefined();
    expect(
      getPendingApiCallPreview({
        actionId,
        chatId: "chat-1",
        toolCallId: "tool-1",
        userName: "users/2",
      })
    ).toBeUndefined();
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();

    vi.setSystemTime(expiresAt);
    expect(
      getPendingApiCallPreview({
        actionId,
        chatId: "chat-1",
        toolCallId: "tool-1",
        userName: "users/1",
      })
    ).toBeUndefined();
    vi.useRealTimers();
  });

  test("rejects a different chat, tool, or user without consuming the approval", async () => {
    const { actionId } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call: makeCall(),
    });

    const wrongChat = await resolvePendingApiCall({
      actionId,
      chatId: "chat-2",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    const wrongTool = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-2",
      userName: "users/1",
      approved: true,
    });
    const wrongUser = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/2",
      approved: true,
    });
    expect(JSON.parse(wrongChat).error).toMatch(/not found|does not match/i);
    expect(JSON.parse(wrongTool).error).toMatch(/not found|does not match/i);
    expect(JSON.parse(wrongUser).error).toMatch(/not found|does not match/i);
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();

    await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    expect(mocks.dispatchPreparedApiCall).toHaveBeenCalledTimes(1);
  });

  test("denial consumes the action without dispatching", async () => {
    const { actionId } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call: makeCall(),
    });

    const denied = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: false,
    });
    expect(JSON.parse(denied).error).toMatch(/denied/i);
    await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("shows Issue consequence and revalidates its state before approval", async () => {
    const call = {
      operationId: "bytebase.v1.IssueService.ApproveIssue",
      path: "/bytebase.v1.IssueService/ApproveIssue",
      body: JSON.stringify({ name: "projects/demo/issues/12" }),
    };
    const issueDecision = {
      issueName: "projects/demo/issues/12",
      issueTitle: "Change account table",
      impact: "final-approval-with-rollout" as const,
      action: "approve" as const,
      snapshot: "before",
    };
    const { actionId, expiresAt } = createPendingApiCall({
      chatId: "chat-issue",
      toolCallId: "tool-issue",
      userName: "users/1",
      call,
      issueDecision,
    });

    expect(
      getPendingApiCallPreview({
        actionId,
        chatId: "chat-issue",
        toolCallId: "tool-issue",
        userName: "users/1",
      })
    ).toEqual({
      operationId: call.operationId,
      body: call.body,
      expiresAt,
      issueDecision: {
        issueName: issueDecision.issueName,
        issueTitle: issueDecision.issueTitle,
        impact: issueDecision.impact,
      },
    });

    mocks.validateIssueDecisionPermission.mockResolvedValueOnce(false);
    const stale = await resolvePendingApiCall({
      actionId,
      chatId: "chat-issue",
      toolCallId: "tool-issue",
      userName: "users/1",
      approved: true,
    });
    expect(JSON.parse(stale).error).toMatch(/changed|unavailable/i);
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
    expect(mocks.validateIssueDecisionPermission).toHaveBeenCalledWith(
      issueDecision
    );
    expect(
      mocks.checkPreparedApiPermission.mock.invocationCallOrder[0]
    ).toBeLessThan(
      mocks.validateIssueDecisionPermission.mock.invocationCallOrder[0]
    );
  });

  test("dispatches a current Issue decision once and does not revalidate denial", async () => {
    const call = {
      operationId: "bytebase.v1.IssueService.RejectIssue",
      path: "/bytebase.v1.IssueService/RejectIssue",
      body: JSON.stringify({ name: "projects/demo/issues/12" }),
    };
    const issueDecision = {
      issueName: "projects/demo/issues/12",
      issueTitle: "Change account table",
      impact: "reject" as const,
      action: "reject" as const,
      snapshot: "before",
    };
    const denied = createPendingApiCall({
      chatId: "chat-issue",
      toolCallId: "tool-deny",
      userName: "users/1",
      call,
      issueDecision,
    });
    await resolvePendingApiCall({
      actionId: denied.actionId,
      chatId: "chat-issue",
      toolCallId: "tool-deny",
      userName: "users/1",
      approved: false,
    });
    expect(mocks.validateIssueDecisionPermission).not.toHaveBeenCalled();

    const allowed = createPendingApiCall({
      chatId: "chat-issue",
      toolCallId: "tool-allow",
      userName: "users/1",
      call,
      issueDecision,
    });
    await resolvePendingApiCall({
      actionId: allowed.actionId,
      chatId: "chat-issue",
      toolCallId: "tool-allow",
      userName: "users/1",
      approved: true,
    });
    expect(mocks.validateIssueDecisionPermission).toHaveBeenCalledOnce();
    expect(mocks.dispatchPreparedApiCall).toHaveBeenCalledExactlyOnceWith(call);
  });

  test("does not dispatch when the approved request was stopped before execution", async () => {
    const { actionId } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call: makeCall(),
    });
    const controller = new AbortController();
    controller.abort();

    const result = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
      signal: controller.signal,
    });

    expect(JSON.parse(result).error).toMatch(/stopped/i);
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
    expect(
      getPendingApiCallPreview({
        actionId,
        chatId: "chat-1",
        toolCallId: "tool-1",
        userName: "users/1",
      })
    ).toBeUndefined();
  });

  test("does not dispatch when stopped during Issue decision validation", async () => {
    let finishValidation!: (valid: boolean) => void;
    mocks.validateIssueDecisionPermission.mockImplementationOnce(
      () => new Promise<boolean>((resolve) => (finishValidation = resolve))
    );
    const { actionId } = createPendingApiCall({
      chatId: "chat-issue",
      toolCallId: "tool-issue",
      userName: "users/1",
      call: makeCall(),
      issueDecision: {
        issueName: "projects/demo/issues/12",
        issueTitle: "Example change",
        impact: "final-approval-with-rollout",
        action: "approve",
        snapshot: "before",
      },
    });
    const controller = new AbortController();
    const resolution = resolvePendingApiCall({
      actionId,
      chatId: "chat-issue",
      toolCallId: "tool-issue",
      userName: "users/1",
      approved: true,
      signal: controller.signal,
    });
    await vi.waitFor(() =>
      expect(mocks.validateIssueDecisionPermission).toHaveBeenCalledOnce()
    );
    controller.abort();
    finishValidation(true);

    expect(JSON.parse(await resolution).error).toMatch(/stopped/i);
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("a replacement request invalidates an older action for the same tool", async () => {
    const first = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call: makeCall(),
    });
    const second = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call: makeCall(),
    });

    const stale = await resolvePendingApiCall({
      actionId: first.actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    expect(JSON.parse(stale).error).toMatch(/not found|used/i);
    await resolvePendingApiCall({
      actionId: second.actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    expect(mocks.dispatchPreparedApiCall).toHaveBeenCalledTimes(1);
  });

  test("expires after five minutes without dispatching", async () => {
    vi.useFakeTimers();
    const { actionId, expiresAt } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call: makeCall(),
    });
    expect(expiresAt).toBe(Date.now() + 5 * 60_000);
    vi.setSystemTime(expiresAt);

    const result = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });

    expect(JSON.parse(result).error).toMatch(/expired/i);
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
    vi.useRealTimers();
  });

  test("consumes approval before a pending dispatch settles", async () => {
    let release!: (value: string) => void;
    mocks.dispatchPreparedApiCall.mockImplementationOnce(
      () => new Promise<string>((resolve) => (release = resolve))
    );
    const { actionId } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call: makeCall(),
    });

    const first = resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    const replay = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    expect(JSON.parse(replay).error).toMatch(/not found|used/i);
    expect(mocks.dispatchPreparedApiCall).toHaveBeenCalledTimes(1);
    release(JSON.stringify({ status: 200, response: {} }));
    await first;
  });
});
