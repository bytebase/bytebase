import { create } from "@bufbuild/protobuf";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import type { AppRouterInstance } from "@/app/router";
import { useAppStore } from "@/stores/app";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { lazyExtractDomRefSuggestions } from "../../dom";
import {
  type AgentDomContext,
  registerAgentDomOwner,
} from "../../dom/interactionRegistry";
import * as apiAttempt from "../apiAttempt";
import { checkPreparedApiPermission } from "./apiPermissionPreflight";
import { dispatchApprovedApiCall, dispatchPreparedApiCall } from "./callApi";
import { createDomActionTool } from "./domAction";
import { createToolExecutor } from "./index";
import {
  getPendingApiCallPreview,
  resolvePendingApiCall,
} from "./pendingApiCall";

vi.mock("./callApi", async (original) => ({
  ...(await original<typeof import("./callApi")>()),
  dispatchApprovedApiCall: vi.fn(),
  dispatchPreparedApiCall: vi.fn(),
}));
vi.mock("./apiMethodClass", async (original) => ({
  ...(await original<typeof import("./apiMethodClass")>()),
  apiCallNeedsConfirmation: () => false,
}));
vi.mock("./apiPermissionPreflight", () => ({
  checkPreparedApiPermission: vi.fn(async () => ({ kind: "allowed" })),
}));
const ctx: AgentDomContext = {
  actor: "users/1",
  workspace: "workspaces/demo",
  chatId: "chat",
  runId: "run",
  runMode: "default",
  pageKey: "/db",
  pageGeneration: 1,
  isCurrent: () => true,
};
beforeEach(() => {
  vi.spyOn(useAppStore, "getState").mockReturnValue({
    ...useAppStore.getState(),
    currentUser: create(UserSchema, {
      name: ctx.actor,
      workspace: ctx.workspace,
    }),
    currentUserName: ctx.actor,
    workspaceResourceName: () => ctx.workspace,
  });
});
afterEach(() => {
  document.body.replaceChildren();
  vi.restoreAllMocks();
  vi.clearAllMocks();
});

test("a generic confirmation cannot activate a manual control", async () => {
  const button = document.createElement("button");
  button.textContent = "Save";
  button.onclick = vi.fn();
  document.body.append(button);
  const [ref] = await lazyExtractDomRefSuggestions(ctx);
  const execute = createToolExecutor({} as AppRouterInstance);
  const question = await execute(
    "ask_user",
    { kind: "confirm", prompt: "Save?" },
    "plain"
  );
  expect(question.kind).toBe("ask_user");
  if (question.kind === "ask_user")
    expect(question.ask.apiPermission).toBeUndefined();
  const result = await createDomActionTool()(
    { type: "click", ref: ref.ref },
    ctx
  );
  expect(result).toMatchObject({
    kind: "result",
    result: { code: "manual-required" },
  });
  expect(button.onclick).not.toHaveBeenCalled();
});

test.each([
  "fresh",
  "preparation denied",
  "IAM changed",
  "revision changed during IAM",
  "revision changed during fingerprint",
  "other project",
])(
  "a registered persistence action classified READ requires a fresh exact API permission: %s",
  async (state) => {
    const button = document.createElement("button");
    button.textContent = "Save";
    document.body.append(button);
    const resourceName = "instances/i/databases/db";
    const call = {
      operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
      path: "/bytebase.v1.DatabaseService/UpdateDatabase",
      body: JSON.stringify({
        database: { name: resourceName, labels: { tier: "dev" } },
        updateMask: "labels",
      }),
    };
    let revision = "1";
    const validateFresh = vi.fn(async () => true);
    if (state === "IAM changed")
      vi.mocked(checkPreparedApiPermission).mockResolvedValueOnce({
        kind: "denied",
        result: '{"error":"Not called"}',
      });
    if (state === "revision changed during IAM")
      vi.mocked(checkPreparedApiPermission).mockImplementationOnce(async () => {
        revision = "2";
        return { kind: "allowed" };
      });
    if (state === "revision changed during fingerprint")
      vi.spyOn(apiAttempt, "fingerprintPreparedApiCall").mockImplementationOnce(
        async () => {
          revision = "2";
          return "fingerprint";
        }
      );
    const dispose = registerAgentDomOwner({
      ownerId: "labels",
      actor: ctx.actor,
      workspace: ctx.workspace,
      getSnapshot: () => ({
        pageKey: ctx.pageKey,
        resourceName,
        projectName: "projects/p",
        revision,
      }),
      getBinding: (el, verb) =>
        el === button && verb === "click"
          ? {
              id: "save",
              verb,
              effect: "persist",
              eligible: () => true,
              validateFresh,
              prepare: async () =>
                state === "preparation denied"
                  ? {
                      result: {
                        success: false,
                        code: "manual-required",
                        message: "The database is unavailable.",
                      },
                    }
                  : {
                      call,
                      preview: {
                        kind: "database-labels",
                        changes: [
                          { field: "tier", before: null, after: "dev" },
                        ],
                      },
                    },
            }
          : undefined,
    });
    try {
      const context = {
        ...ctx,
        projectName: state === "other project" ? "projects/q" : "projects/p",
      };
      const [ref] = await lazyExtractDomRefSuggestions(context);
      const execute = createToolExecutor({} as AppRouterInstance, {
        chatId: ctx.chatId,
        runId: ctx.runId,
        userName: ctx.actor,
        workspaceName: ctx.workspace,
        context: context.projectName,
        getDomContext: () => context,
        priorAttemptIds: () => ["earlier-unknown-attempt"],
        isApiScopeCurrent: () => true,
      });
      const result = await execute(
        "dom_action",
        { type: "click", ref: ref.ref },
        "original-dom"
      );
      expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
      expect(dispatchApprovedApiCall).not.toHaveBeenCalled();
      if (state !== "fresh") {
        expect(result.kind).toBe("tool_result");
        if (state === "IAM changed" && result.kind === "tool_result")
          expect(JSON.parse(result.result)).toMatchObject({
            agentExecution: {
              result: "not-sent",
              reason: "preflight-rejected",
            },
          });
        if (state === "other project")
          expect(validateFresh).not.toHaveBeenCalled();
        return;
      }
      expect(result).toMatchObject({
        kind: "ask_user",
        ask: {
          toolCallId: "original-dom",
          kind: "confirm",
          apiPermission: { operationId: call.operationId },
        },
      });
      if (result.kind !== "ask_user" || !result.ask.apiPermission)
        throw new Error("missing permission");
      const identity = {
        actionId: result.ask.apiPermission.actionId,
        chatId: ctx.chatId,
        toolCallId: "original-dom",
        userName: ctx.actor,
      };
      expect(getPendingApiCallPreview(identity)).toBeUndefined();
      expect(
        getPendingApiCallPreview({ ...identity, domContext: context })
      ).toMatchObject({
        body: call.body,
        dom: { resourceName, kind: "database-labels" },
        priorAttemptIds: ["earlier-unknown-attempt"],
      });
      expect(validateFresh).not.toHaveBeenCalled();
      expect(checkPreparedApiPermission).toHaveBeenCalledOnce();
      expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
      vi.mocked(dispatchApprovedApiCall).mockResolvedValueOnce(
        JSON.stringify({ status: 200, response: {} })
      );
      const approved = await resolvePendingApiCall({
        ...identity,
        domContext: context,
        approved: true,
      });
      expect(JSON.parse(approved)).toMatchObject({ status: 200 });
      expect(validateFresh).toHaveBeenCalledOnce();
      expect(checkPreparedApiPermission).toHaveBeenCalledTimes(2);
      expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
      expect(dispatchApprovedApiCall).toHaveBeenCalledExactlyOnceWith(
        call,
        expect.objectContaining({
          dom: expect.objectContaining({ beforeSend: expect.any(Function) }),
        })
      );
    } finally {
      dispose();
    }
  }
);
