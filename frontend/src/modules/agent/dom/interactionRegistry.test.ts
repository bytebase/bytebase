import { afterEach, expect, test, vi } from "vitest";
import { getElementByRef } from "./domTree";
import { lazyExtractDomRefSuggestions, lazyPrepareDomAction } from "./index";
import {
  type AgentDomBinding,
  type AgentDomContext,
  type AgentDomOwner,
  type CommittedDomMutation,
  isCapturedDomMutationCurrent,
  matchesCapturedDomMutationCall,
  prepareCapturedInteraction,
  refreshCommittedDomMutation,
  registerAgentDomOwner,
  validateCapturedDomMutation,
} from "./interactionRegistry";

const context: AgentDomContext = {
  actor: "users/1",
  workspace: "workspaces/demo",
  chatId: "chat",
  runId: "run",
  runMode: "default",
  pageKey: "/db",
  pageGeneration: 1,
  isCurrent: () => true,
};
const disposers: (() => void)[] = [];
afterEach(() => {
  for (const dispose of disposers.splice(0)) dispose();
  document.body.replaceChildren();
  vi.resetAllMocks();
});
function setup() {
  document.body.innerHTML =
    '<div id="wrapper" role="button"><div id="target">Delete</div></div><button id="portal">Delete</button>';
  const target = document.getElementById("target")!;
  const run = vi.fn(() => ({ success: true, message: "Updated draft" }));
  const snapshot = {
    pageKey: context.pageKey,
    resourceName: "instances/i/databases/db",
    projectName: "projects/p",
    revision: "1",
  };
  const state: { binding: AgentDomBinding } = {
    binding: {
      id: "row/1",
      verb: "click",
      eligible: () => true,
      effect: "local-draft",
      run,
    },
  };
  const refresh = vi.fn(
    async (_context: AgentDomContext, _committed: CommittedDomMutation) => {}
  );
  const owner: AgentDomOwner = {
    ownerId: "labels",
    actor: context.actor,
    workspace: context.workspace,
    getSnapshot: () => ({ ...snapshot }),
    getBinding: (el, verb) =>
      el === target && verb === state.binding.verb
        ? { ...state.binding }
        : undefined,
    refresh,
  };
  const dispose = registerAgentDomOwner(owner);
  disposers.push(dispose);
  async function observe() {
    const refs = await lazyExtractDomRefSuggestions(context);
    const ref = refs.find(
      (r) => r.tag === "div" && r.actions !== "manual-required"
    )?.ref;
    expect(ref).toBeDefined();
    return ref!;
  }
  return { owner, target, snapshot, state, run, refresh, dispose, observe };
}
const click = (ref: string, ctx = context) =>
  lazyPrepareDomAction({ ref, type: "click" }, ctx);
test("registered local drafts need no subscription metadata or network read", async () => {
  const { observe, state, run } = setup();
  state.binding = {
    id: "local",
    verb: "click",
    eligible: () => true,
    effect: "local-draft",
    run,
  };
  const fetch = vi.spyOn(globalThis, "fetch");
  try {
    expect(await click(await observe())).toMatchObject({
      kind: "result",
      result: { success: true },
    });
    expect(run).toHaveBeenCalledTimes(1);
    expect(fetch).not.toHaveBeenCalled();
  } finally {
    fetch.mockRestore();
  }
});
test("keeps_binding_on_unrelated_rerender and invalidates before React commits", async () => {
  const { observe, run } = setup();
  const ref = await observe();
  expect(await click(ref)).toMatchObject({
    kind: "result",
    result: { success: true },
  });
  expect(await click(ref)).toMatchObject({
    kind: "result",
    result: { success: false },
  });
  expect(run).toHaveBeenCalledTimes(1);
});
test("rejects_recycled_label_row_after_owner_revision_changes", async () => {
  const { observe, run, snapshot, state } = setup();
  state.binding = { ...state.binding, verb: "input" };
  const ref = await observe();
  snapshot.revision = "2";
  state.binding = { ...state.binding, id: "row/2" };
  expect(
    await lazyPrepareDomAction({ ref, type: "input", value: "new" }, context)
  ).toMatchObject({ result: { success: false } });
  expect(run).not.toHaveBeenCalled();
});
test.each([
  "actor",
  "workspace",
  "runId",
  "chatId",
  "pageKey",
  "pageGeneration",
  "projectName",
  "runMode",
] as const)("rejects different %s", async (key) => {
  const { observe, run } = setup();
  expect(
    await click(await observe(), { ...context, [key]: "other" })
  ).toMatchObject({ result: { success: false } });
  expect(run).not.toHaveBeenCalled();
});
test.each(["local-draft", "persist"] as const)(
  "direct preparation cannot use a retired %s entry",
  async (effect) => {
    const { observe, state, run } = setup();
    const prepare = vi.fn(async () => ({
      call: {
        operationId: "example.Update",
        path: "/example/Update",
        body: "{}",
      },
      preview: { kind: "database-labels" as const, changes: [] },
    }));
    if (effect === "persist")
      state.binding = {
        id: "save",
        verb: "click",
        effect,
        eligible: () => true,
        prepare,
        validateFresh: async () => true,
      };
    const ref = await observe(),
      entry = getElementByRef(ref)!;
    const nextRef = await observe();
    for (const candidate of [ref, nextRef])
      expect(
        await prepareCapturedInteraction(
          { ref: candidate, type: "click" },
          entry,
          context
        )
      ).toMatchObject({ kind: "result", result: { success: false } });
    expect(run).not.toHaveBeenCalled();
    expect(prepare).not.toHaveBeenCalled();
  }
);
test("a control inside a disclosure loses its owner action when closed", async () => {
  const { target, observe, run } = setup();
  const details = document.createElement("details");
  details.open = true;
  target.replaceWith(details);
  details.append(target);
  const ref = await observe();
  details.open = false;
  expect(await click(ref)).toMatchObject({ result: { success: false } });
  expect(run).not.toHaveBeenCalled();
});
test.each(["current", "retired", "background"])(
  "a proven no-change preparation rechecks its %s context and ref",
  async (condition) => {
    const { observe, state } = setup();
    const result = { success: true, message: "Already selected" };
    let foreground = true;
    const scope = { ...context, isCurrent: () => foreground };
    const prepare = vi.fn(
      async (_value: string | undefined, _ctx: AgentDomContext) => {
        if (condition === "retired") await observe();
        if (condition === "background") foreground = false;
        return { result };
      }
    );
    state.binding = {
      id: "environment",
      verb: "select",
      effect: "persist",
      eligible: () => true,
      prepare,
      validateFresh: async () => true,
    };
    const prepared = await lazyPrepareDomAction(
      { ref: await observe(), type: "select", value: "environments/p" },
      scope
    );
    expect(prepare).toHaveBeenCalledWith("environments/p", scope);
    if (condition === "current")
      expect(prepared).toEqual({ kind: "result", result });
    else
      expect(prepared).toMatchObject({
        kind: "result",
        result: { success: false },
      });
    expect(prepared).not.toHaveProperty("origin");
  }
);
test.each(["cancel", "unmount", "foreground", "revision", "binding"])(
  "rechecks %s during mutation preparation",
  async (change) => {
    let current = true;
    const { dispose, state, run, observe, snapshot } = setup();
    const controller = new AbortController();
    state.binding = {
      id: "save",
      verb: "click",
      effect: "persist",
      eligible: () => true,
      prepare: async () => {
        if (change === "cancel") controller.abort();
        if (change === "unmount") dispose();
        if (change === "foreground") current = false;
        if (change === "revision") snapshot.revision = "2";
        if (change === "binding")
          state.binding = { ...state.binding, id: "other" };
        return {
          call: {
            operationId: "example.Update",
            path: "/example/Update",
            body: "{}",
          },
          preview: { kind: "database-labels", changes: [] },
        };
      },
      validateFresh: async () => true,
    };
    expect(
      await click(await observe(), {
        ...context,
        signal: controller.signal,
        isCurrent: () => current,
      })
    ).toMatchObject({ result: { success: false } });
    expect(run).not.toHaveBeenCalled();
  }
);
test.each([undefined, "unknown"])(
  "an unclassified effect %s stays manual",
  async (effect) => {
    const { state, run } = setup();
    Object.assign(state.binding, { effect });
    const refs = await lazyExtractDomRefSuggestions(context);
    for (const { ref } of refs)
      expect(await click(ref)).toMatchObject({ result: { success: false } });
    expect(run).not.toHaveBeenCalled();
  }
);
test("owner permission doesn't cover parents or portaled controls", async () => {
  const { run } = setup();
  const refs = await lazyExtractDomRefSuggestions(context);
  const manual = refs.filter((r) => r.actions === "manual-required");
  expect(manual.some((r) => r.tag === "button")).toBe(true);
  for (const { ref } of manual)
    expect(await click(ref)).toMatchObject({
      result: { code: "manual-required" },
    });
  expect(run).not.toHaveBeenCalled();
});
test.each([
  "password",
  "agent",
  "critical",
  "opacity",
  "display",
  "inert",
  "disabled",
  "disconnected",
])("rejects a %s target after extraction", async (change) => {
  const { observe, target, run } = setup();
  const ref = await observe();
  const parent = target.parentElement!;
  if (change === "password") target.innerHTML = '<input type="password">';
  if (change === "agent" || change === "critical") {
    parent.setAttribute("data-bb-layer-family", change);
    target.setAttribute("data-bb-layer-family", "overlay");
  }
  if (change === "opacity") parent.style.opacity = "0";
  if (change === "display") parent.style.display = "none";
  if (change === "inert") parent.setAttribute("data-base-ui-inert", "");
  if (change === "disabled") parent.setAttribute("aria-disabled", "true");
  if (change === "disconnected") target.remove();
  expect(await click(ref)).toMatchObject({ result: { success: false } });
  expect(run).not.toHaveBeenCalled();
});
test("refuses ambiguous registrations and old mount identities", async () => {
  const { observe, owner, dispose, run } = setup();
  const old = await observe();
  disposers.push(registerAgentDomOwner({ ...owner }));
  expect(await click(old)).toMatchObject({ result: { success: false } });
  dispose();
  dispose();
  expect(await click(old)).toMatchObject({ result: { success: false } });
  expect(await click(await observe())).toMatchObject({
    result: { success: true },
  });
  expect(run).toHaveBeenCalledTimes(1);
});

test("a duplicate mount invalidates earlier evidence even after it is removed", async () => {
  const { owner, observe, run } = setup();
  const old = await observe();
  const dispose = registerAgentDomOwner({ ...owner });
  dispose();
  expect(await click(old)).toMatchObject({ result: { success: false } });
  expect(run).not.toHaveBeenCalled();
});

test("a mount cannot be rebound to another principal without reregistering", async () => {
  const { owner, run } = setup();
  owner.actor = "users/2";
  const scope = { ...context, actor: owner.actor };
  const refs = await lazyExtractDomRefSuggestions(scope);
  expect(refs.every((r) => r.actions === "manual-required")).toBe(true);
  for (const { ref } of refs)
    expect(await click(ref, scope)).toMatchObject({
      result: { success: false },
    });
  expect(run).not.toHaveBeenCalled();
});

test("keeps the exact registered target inside a tree terminal", async () => {
  const { target, observe, run } = setup();
  const terminal = document.createElement("button");
  target.replaceWith(terminal);
  terminal.append(target);
  expect(await click(await observe())).toMatchObject({
    result: { success: true },
  });
  expect(run).toHaveBeenCalledOnce();
});

test("unmounting while preparation awaits cannot create a permit", async () => {
  const { state, observe, dispose } = setup();
  state.binding = {
    id: "save",
    verb: "click",
    eligible: () => true,
    effect: "persist",
    validateFresh: async () => true,
    prepare: async () => {
      await Promise.resolve();
      dispose();
      return {
        call: {
          operationId: "example.Update",
          path: "/example/Update",
          body: "{}",
        },
        preview: { kind: "database-labels", changes: [] },
      };
    },
  };
  expect(await click(await observe())).toMatchObject({
    kind: "result",
    result: { success: false },
  });
});

test("unavailable owner preparation produces a tool result without leaking exception details", async () => {
  const { state, observe } = setup();
  state.binding = {
    id: "save",
    verb: "click",
    eligible: () => true,
    effect: "persist",
    validateFresh: async () => true,
    prepare: async () => {
      throw new Error("sensitive owner exception");
    },
  };
  const prepared = await click(await observe());
  expect(prepared).toMatchObject({
    kind: "result",
    result: { success: false },
  });
  expect(JSON.stringify(prepared)).not.toContain("sensitive owner exception");
});
test("permission captures immutable values, survives ref replacement but not freshness races", async () => {
  const { observe, state, refresh, snapshot, target } = setup();
  const call = {
    operationId: "databaseService.UpdateDatabase",
    path: "/databaseService/UpdateDatabase",
    body: '{"labels":{"x":"y"}}',
  };
  const preview = {
    kind: "database-labels" as const,
    changes: [{ field: "x", before: null, after: "y" }],
  };
  const validateFresh = vi.fn(async () => true);
  state.binding = {
    id: "save",
    verb: "click",
    eligible: () => true,
    effect: "persist",
    prepare: async () => ({ call, preview }),
    validateFresh,
  };
  const prepared = await click(await observe());
  expect(prepared.kind).toBe("confirm-api");
  if (prepared.kind !== "confirm-api") throw Error();
  call.body = "changed";
  preview.changes[0].after = "changed";
  expect(prepared.call.body).not.toBe(call.body);
  expect(prepared.origin.preview.changes[0].after).toBe("y");
  await observe();
  expect(isCapturedDomMutationCurrent(prepared.origin, context)).toBe(true);
  expect(validateFresh).not.toHaveBeenCalled();
  expect(await validateCapturedDomMutation(prepared.origin, context)).toBe(
    true
  );
  await refreshCommittedDomMutation(prepared.origin, context);
  expect(refresh).toHaveBeenCalledTimes(1);
  validateFresh.mockImplementation(async () => {
    snapshot.revision = "changed";
    return true;
  });
  expect(await validateCapturedDomMutation(prepared.origin, context)).toBe(
    false
  );
  target.setAttribute("aria-disabled", "true");
  state.binding.eligible = () => false;
  await refreshCommittedDomMutation(prepared.origin, context);
  expect(refresh).toHaveBeenCalledTimes(2);
  const [refreshContext, committed] = refresh.mock.lastCall!;
  expect(refreshContext.isCurrent()).toBe(true);
  expect(committed.call).toBe(prepared.call);
  expect(committed.revision).toBe("1");
  expect(Object.isFrozen(committed)).toBe(true);
  snapshot.projectName = "projects/other";
  expect(refreshContext.isCurrent()).toBe(false);
});

test("freshness reads use each origin's exact immutable call", async () => {
  const { observe, state } = setup();
  const source = {
    operationId: "example.Update",
    path: "/example/Update",
    body: "{}",
  };
  const validateFresh = vi.fn(
    async (_ctx: AgentDomContext, _call: CommittedDomMutation["call"]) => true
  );
  state.binding = {
    id: "environment",
    verb: "select",
    effect: "persist",
    eligible: () => true,
    validateFresh,
    prepare: async (value) => {
      source.body = JSON.stringify({ environment: value });
      return {
        call: source,
        preview: { kind: "database-environment", changes: [] },
      };
    },
  };
  const ref = await observe();
  const prepare = (value: string) =>
    lazyPrepareDomAction({ ref, type: "select", value }, context);
  const a = await prepare("environments/a"),
    b = await prepare("environments/b");
  source.body = "changed";
  if (a.kind !== "confirm-api" || b.kind !== "confirm-api")
    throw Error("expected preparations");
  expect(await validateCapturedDomMutation(b.origin, context)).toBe(true);
  expect(await validateCapturedDomMutation(a.origin, context)).toBe(true);
  expect(validateFresh.mock.calls).toEqual([
    [context, b.call],
    [context, a.call],
  ]);
  expect(validateFresh.mock.calls[0][1]).toBe(b.call);
  expect(validateFresh.mock.calls[1][1]).toBe(a.call);
  expect(a.call.body).toBe('{"environment":"environments/a"}');
  expect(b.call.body).toBe('{"environment":"environments/b"}');
  expect(Object.isFrozen(a.call)).toBe(true);
  expect(matchesCapturedDomMutationCall(a.origin, { ...a.call })).toBe(true);
  expect(matchesCapturedDomMutationCall(a.origin, b.call)).toBe(false);
  expect(
    matchesCapturedDomMutationCall(a.origin, {
      ...a.call,
      path: "/other/Update",
    })
  ).toBe(false);
  expect(
    matchesCapturedDomMutationCall(a.origin, {
      ...a.call,
      operationId: "other.Update",
    })
  ).toBe(false);
  expect(matchesCapturedDomMutationCall({ ...a.origin }, a.call)).toBe(false);
});

test("refresh skips different mounts or scope and keeps refresh errors separate", async () => {
  const { observe, state, owner, dispose, snapshot, refresh } = setup();
  const prepare = vi.fn(async () => ({
    call: {
      operationId: "example.Update",
      path: "/example/Update",
      body: "{}",
    },
    preview: { kind: "database-labels" as const, changes: [] },
  }));
  state.binding = {
    id: "save",
    verb: "click",
    effect: "persist",
    eligible: () => true,
    validateFresh: async () => true,
    prepare,
  };
  const p = await click(await observe());
  if (p.kind !== "confirm-api") throw Error("expected preparation");
  await refreshCommittedDomMutation({ ...p.origin }, context);
  await refreshCommittedDomMutation(p.origin, { ...context, actor: "users/2" });
  snapshot.resourceName = "instances/other/databases/db";
  await refreshCommittedDomMutation(p.origin, context);
  expect(refresh).not.toHaveBeenCalled();
  snapshot.resourceName = p.origin.resourceName;
  refresh.mockRejectedValueOnce(new Error("read failed"));
  await expect(refreshCommittedDomMutation(p.origin, context)).rejects.toThrow(
    "read failed"
  );
  dispose();
  disposers.push(registerAgentDomOwner({ ...owner }));
  await refreshCommittedDomMutation(p.origin, context);
  expect(refresh).toHaveBeenCalledTimes(1);
  expect(prepare).toHaveBeenCalledTimes(1);
});
