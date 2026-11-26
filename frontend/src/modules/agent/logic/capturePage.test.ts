import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { captureAgentPageSnapshot } from "./capturePage";
import type { AgentSqlContextCapture } from "./sqlEditorContext";
import { registerAgentSqlEditorHost } from "./sqlEditorHost";

vi.mock("@/app/router", () => ({
  router: { currentRoute: { value: { fullPath: "/known?tab=example" } } },
}));

beforeEach(() => {
  document.title = "Example";
});

let unregister: (() => void) | undefined;
afterEach(() => {
  unregister?.();
  document.body.replaceChildren();
});

const host = (captureContext: () => unknown) => {
  const slot = document.createElement("div");
  document.body.appendChild(slot);
  unregister = registerAgentSqlEditorHost({
    id: "mount-1",
    slot,
    mainColumn: document.createElement("div"),
    resizer: document.createElement("div"),
    paneRatio: 0.3,
    captureContext: () => captureContext() as AgentSqlContextCapture,
  });
};

test("captures ordinary page title and full route without a SQL Editor host", () => {
  expect(captureAgentPageSnapshot()).toEqual({
    path: "/known?tab=example",
    title: "Example",
  });
});

test("does not fall back to Workspace during SQL Editor Project initialization", () => {
  host(vi.fn(() => ({ kind: "pending" })));
  expect(captureAgentPageSnapshot()).toBeUndefined();
});

test("captures a ready SQL snapshot and not the host adapter", () => {
  host(
    vi.fn(() => ({
      kind: "ready",
      snapshot: { version: 1, project: "projects/example", callback: vi.fn() },
    }))
  );
  expect(captureAgentPageSnapshot()).toEqual({
    path: "/known?tab=example",
    title: "Example",
    sqlEditor: { version: 1, project: "projects/example" },
  });
});

test("does not fall back when the host reports malformed ready context", () => {
  host(vi.fn(() => ({ kind: "ready", snapshot: { version: 2 } })));
  expect(captureAgentPageSnapshot()).toBeUndefined();
});
