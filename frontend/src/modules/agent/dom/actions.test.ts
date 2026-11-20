import { afterEach, expect, test, vi } from "vitest";
import {
  extractDomRefSuggestions,
  extractDomTree,
  prepareAgentDomInteraction,
} from "./actions";
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
const editor = vi.hoisted(() => ({
  getMonacoEditor: vi.fn(),
  getValue: vi.fn(),
}));
vi.mock("@/components/monaco/core", () => ({
  isMonacoLoaded: () => true,
  getMonacoEditor: editor.getMonacoEditor,
}));

afterEach(() => {
  document.body.replaceChildren();
  vi.useRealTimers();
});

test.each([
  ["click", "<button>Save</button>"],
  ["click", '<a href="#changed">Navigate</a>'],
  ["input", '<input value="original">'],
  [
    "select",
    "<select><option>original</option><option>changed</option></select>",
  ],
] as const)(
  "unknown %s (%s) never activates the control",
  async (type, html) => {
    vi.useFakeTimers();
    document.body.innerHTML = '<input id="focused">' + html;
    const focused = document.querySelector<HTMLInputElement>("#focused")!;
    focused.focus();
    const target = document.body.lastElementChild!;
    const later = vi.fn();
    const received = vi.fn(() => setTimeout(later, 1));
    for (const name of [
      "focus",
      "mousedown",
      "mouseup",
      "click",
      "input",
      "change",
    ]) {
      target.addEventListener(name, received);
    }
    const ref = extractDomRefSuggestions(undefined, context).at(-1)!.ref;
    const prepared = await prepareAgentDomInteraction(
      { type, ref, value: "changed" },
      target,
      context
    );
    expect(prepared).toMatchObject({
      kind: "result",
      result: { success: false, code: "manual-required" },
    });
    await vi.runAllTimersAsync();
    expect(received).not.toHaveBeenCalled();
    expect(later).not.toHaveBeenCalled();
    expect(document.activeElement).toBe(focused);
    if (
      target instanceof HTMLInputElement ||
      target instanceof HTMLSelectElement
    )
      expect(target.value).toBe("original");
  }
);

test.each(["available", "nested-critical", "different-editor", "cancel"])(
  "reads only the exact permissible Monaco model: %s",
  async (change) => {
    document.body.innerHTML =
      '<div class="monaco-editor"><div class="view-lines">Visible SQL</div></div>';
    const target = document.body.firstElementChild!;
    const controller = new AbortController();
    const ctx = { ...context, signal: controller.signal };
    const ref = extractDomRefSuggestions(undefined, ctx)[0].ref;
    editor.getValue.mockClear().mockReturnValue("Full SQL");
    editor.getMonacoEditor.mockImplementation(async () => {
      if (change === "nested-critical")
        target.innerHTML =
          '<span data-bb-layer-family="critical">Private</span>';
      if (change === "cancel") controller.abort();
      return {
        editor: {
          getEditors: () => [
            {
              getDomNode: () =>
                change === "different-editor" ? document.body : target,
              getValue: editor.getValue,
            },
          ],
        },
      };
    });
    const result = await prepareAgentDomInteraction(
      { type: "read", ref },
      target,
      ctx
    );
    if (change === "available")
      expect(result).toMatchObject({
        result: { success: true, message: "Full SQL" },
      });
    else {
      expect(editor.getValue).not.toHaveBeenCalled();
      if (change === "different-editor")
        expect(result).toMatchObject({
          result: { success: true, message: "Visible SQL" },
        });
      else expect(result).toMatchObject({ result: { success: false } });
    }
  }
);

test("scrolls a visible nonsensitive target but refuses a hidden ancestor", async () => {
  document.body.innerHTML = "<div><button>Scroll here</button></div>";
  const target = document.querySelector("button")!;
  target.scrollIntoView = vi.fn();
  const { ref } = extractDomRefSuggestions(undefined, context)[0];
  expect(
    await prepareAgentDomInteraction({ ref, type: "scroll" }, target, context)
  ).toMatchObject({ result: { success: true } });
  target.parentElement!.hidden = true;
  expect(
    await prepareAgentDomInteraction({ ref, type: "scroll" }, target, context)
  ).toMatchObject({ result: { success: false } });
  expect(target.scrollIntoView).toHaveBeenCalledTimes(1);
});

test("read-only page state cannot authorize either read or scroll actions", async () => {
  document.body.innerHTML = '<input value="page data">';
  const ctx = { ...context, runMode: "read-only" as const };
  const { ref } = extractDomRefSuggestions(undefined, ctx)[0];
  const target = document.querySelector("input")!;
  target.scrollIntoView = vi.fn();
  for (const type of ["read", "scroll"] as const) {
    const prepared = await prepareAgentDomInteraction(
      { ref, type },
      target,
      ctx
    );
    expect(prepared).toMatchObject({
      kind: "result",
      result: { success: false },
    });
    expect(JSON.stringify(prepared)).not.toContain("page data");
  }
  expect(target.scrollIntoView).not.toHaveBeenCalled();
});
test("closing a disclosure retires its content for inspection but preserves the first summary", async () => {
  document.body.innerHTML =
    "<details open>private direct text<summary>Public <span>title</span></summary><div><button>private action</button></div><summary>private second summary</summary></details>";
  const details = document.querySelector("details")!;
  const target = document.querySelector("button")!;
  const ref = extractDomRefSuggestions(undefined, context).find(
    (s) => s.tag === "button"
  )!.ref;
  details.open = false;
  expect(
    await prepareAgentDomInteraction({ ref, type: "read" }, target, context)
  ).toMatchObject({ result: { success: false } });
  const { tree } = extractDomTree(undefined, context);
  expect(tree).toContain("Public title");
  expect(tree).not.toContain("private");
});
