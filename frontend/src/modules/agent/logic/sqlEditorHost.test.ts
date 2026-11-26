import { afterEach, expect, test, vi } from "vitest";
import { useAgentStore } from "../store/agent";
import {
  getRegisteredAgentSqlEditorHost,
  registerAgentSqlEditorHost,
} from "./sqlEditorHost";

afterEach(() => {
  useAgentStore.setState({ sqlEditorHost: undefined });
  document.body.replaceChildren();
});

test("only the current registration can clear a mounted SQL host", () => {
  const slot = document.createElement("div");
  document.body.appendChild(slot);
  const host = {
    id: "editor",
    slot,
    mainColumn: slot,
    resizer: slot,
    paneRatio: 0.3,
    captureContext: vi.fn(() => ({ kind: "pending" as const })),
  };
  const first = registerAgentSqlEditorHost(host);
  const second = registerAgentSqlEditorHost(host);
  first();
  expect(getRegisteredAgentSqlEditorHost()?.slot).toBe(slot);
  expect(getRegisteredAgentSqlEditorHost()?.captureContext()).toEqual({
    kind: "pending",
  });
  slot.remove();
  expect(getRegisteredAgentSqlEditorHost()).toBeUndefined();
  second();
  expect(useAgentStore.getState().sqlEditorHost).toBeUndefined();
});
