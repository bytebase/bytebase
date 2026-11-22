import { afterEach, expect, test, vi } from "vitest";
import {
  lazyExtractDomRefSuggestions,
  lazyExtractDomTree,
  lazyPrepareDomAction,
} from "./index";
import type { AgentDomContext } from "./interactionRegistry";

const context: AgentDomContext = {
  actor: "users/1",
  workspace: "workspaces/demo",
  chatId: "chat",
  runId: "run",
  runMode: "default",
  pageKey: "/current",
  pageGeneration: 1,
  isCurrent: () => true,
};
afterEach(() => {
  document.body.replaceChildren();
  vi.restoreAllMocks();
});

test("returns structured suggestions and reads a scoped ref", async () => {
  document.body.innerHTML = '<button aria-label="Run query">Run</button>';
  const [entry] = await lazyExtractDomRefSuggestions(context);
  expect(entry).toMatchObject({
    tag: "button",
    label: "Run query",
    actions: "manual-required",
  });
  expect(
    await lazyPrepareDomAction({ type: "read", ref: entry.ref }, context)
  ).toEqual({ kind: "result", result: { success: true, message: "Run" } });
});
test("unscoped suggestions grant no page authority", async () => {
  document.body.innerHTML = "<button>Run</button>";
  const [entry] = await lazyExtractDomRefSuggestions();
  expect(
    await lazyPrepareDomAction({ type: "read", ref: entry.ref }, context)
  ).toMatchObject({ result: { success: false } });
});
test.each([
  "data-agent-sensitive",
  "data-agent-window",
  'data-bb-layer-family="critical"',
  'data-bb-layer-family="agent"',
])(
  "omits %s from indexed wrappers, sibling labels and referenced labels",
  async (attr) => {
    document.body.innerHTML = `<div role="button" aria-label="Group"><span ${attr}><span data-bb-layer-family="overlay" id="secret">private-value</span></span></div>
   <div><span ${attr}>private-value</span><input></div>
   <button aria-labelledby="secret">Normal</button>
   <table><thead><tr><th><span ${attr}>private-value</span></th></tr></thead><tbody><tr><td><button aria-label="Do">Do</button></td></tr></tbody></table>
   <div class="monaco-editor"><div class="view-lines"><span class="view-line" ${attr}>private-value</span></div></div>`;
    expect((await lazyExtractDomTree(context)).tree).not.toContain(
      "private-value"
    );
    const entries = await lazyExtractDomRefSuggestions(context);
    expect(JSON.stringify(entries)).not.toContain("private-value");
    const wrapper = entries.find((e) => e.label === "Group")!;
    for (const type of ["read", "input"] as const) {
      const result = await lazyPrepareDomAction(
        { ref: wrapper.ref, type, value: "changed" },
        context
      );
      expect(result).toMatchObject({ result: { success: false } });
      expect(JSON.stringify(result)).not.toContain("private-value");
    }
  }
);
test.each(["password", "sensitive"])(
  "protects %s acquired after extraction",
  async (change) => {
    document.body.innerHTML = '<input value="private-value">';
    const [entry] = await lazyExtractDomRefSuggestions(context);
    const input = document.querySelector("input")!;
    if (change === "password") input.type = "password";
    else input.setAttribute("data-agent-sensitive", "");
    for (const type of ["read", "input"] as const) {
      const result = await lazyPrepareDomAction(
        { ref: entry.ref, type, value: "new" },
        context
      );
      expect(result).toMatchObject({ result: { success: false } });
      expect(JSON.stringify(result)).not.toContain("private-value");
    }
    expect(input.value).toBe("private-value");
    expect(await lazyExtractDomRefSuggestions(context)).toEqual([]);
  }
);
test("snapshot and action recheck foreground after the lazy import", async () => {
  document.body.innerHTML = '<input value="hello">';
  let current = true;
  const scope = { ...context, isCurrent: () => current };
  const [entry] = await lazyExtractDomRefSuggestions(scope);
  const read = lazyPrepareDomAction({ type: "read", ref: entry.ref }, scope);
  const tree = lazyExtractDomTree(scope);
  current = false;
  expect(await read).toMatchObject({ result: { success: false } });
  expect(await tree).toEqual({ tree: "", count: 0 });
});
test("rejects all unregistered link activations without navigation", async () => {
  document.body.innerHTML = '<a href="/projects/demo">Project</a>';
  const [entry] = await lazyExtractDomRefSuggestions(context);
  const click = vi.fn();
  document.querySelector("a")!.addEventListener("click", click);
  const previous = location.href;
  expect(
    await lazyPrepareDomAction({ type: "click", ref: entry.ref }, context)
  ).toMatchObject({ result: { code: "manual-required" } });
  expect(location.href).toBe(previous);
  expect(click).not.toHaveBeenCalled();
});
test("gives refresh guidance for malformed, missing and disconnected refs", async () => {
  document.body.innerHTML = "<button>Save</button>";
  const [entry] = await lazyExtractDomRefSuggestions(context);
  document.querySelector("button")!.remove();
  for (const ref of [1, "1", "e0", entry.ref]) {
    expect(
      await lazyPrepareDomAction({ type: "read", ref: ref as string }, context)
    ).toMatchObject({
      result: {
        success: false,
        message: expect.stringContaining("get_page_state"),
      },
    });
  }
});
