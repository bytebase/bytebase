import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, expect, test } from "vitest";
import { registerAgentSqlEditorHost } from "@/modules/agent/logic/sqlEditorHost";
import { useAgentStore } from "@/modules/agent/store/agent";
import { useSqlEditorAgentOverlay } from "./useSqlEditorAgentOverlay";

let slot: HTMLElement;
let unregister: () => void;
beforeEach(() => {
  useAgentStore.setState(useAgentStore.getInitialState(), true);
  useAgentStore.getState().setWorkspaceName("workspaces/demo", "member@example.com");
  slot = document.createElement("div");
  document.body.append(slot);
  unregister = registerAgentSqlEditorHost({id:"editor", slot, mainColumn:slot, resizer:slot, paneRatio:0.3, captureContext:()=>({kind:"ready", snapshot:{version:1}})});
});
afterEach(() => { unregister(); slot.remove(); localStorage.clear(); });

test("nested native surfaces yield one visible Agent and restore its held run and draft on last close", () => {
  const store = useAgentStore.getState();
  const chat = store.createChat();
  const controller = new AbortController();
  store.setAbortController(chat.id, controller);
  const ask = { kind: "confirm" as const, toolCallId: "native-query", prompt: "Allow the query?" };
  store.setPendingAsk(chat.id, ask);
  store.setDraft(chat.id,{text:"Unsent follow-up"});
  store.open();
  const outer = renderHook(({open})=>useSqlEditorAgentOverlay(open), {initialProps:{open:true}});
  const inner = renderHook(()=>useSqlEditorAgentOverlay(true));
  expect(useAgentStore.getState().minimized).toBe(true);
  outer.rerender({open:false});
  expect(useAgentStore.getState().minimized).toBe(true);
  inner.unmount();
  expect(useAgentStore.getState().minimized).toBe(false);
  expect(store.getDraft(chat.id).text).toBe("Unsent follow-up");
  expect(store.getAbortController(chat.id)).toBe(controller);
  expect(store.getPendingAsk(chat.id)).toEqual(ask);
  expect(controller.signal.aborted).toBe(false);
  outer.unmount();
});

test.each(["closed","minimized","missing-host"] as const)("does not claim already %s Agent", (initial) => {
  const store = useAgentStore.getState();
  if (initial !== "closed") store.open();
  if (initial === "minimized") store.minimize();
  if (initial === "missing-host") unregister();
  const before = useAgentStore.getState();
  const view = renderHook(()=>useSqlEditorAgentOverlay(true));
  view.unmount();
  expect(useAgentStore.getState().visible).toBe(before.visible);
  expect(useAgentStore.getState().minimized).toBe(before.minimized);
});

test.each(["close", "identity", "manual-restore"] as const)("does not undo a later %s even if the minimized state recurs", (change) => {
  const store=useAgentStore.getState();
  store.open();
  const view=renderHook(()=>useSqlEditorAgentOverlay(true));
  act(()=>{
    if(change==="close") useAgentStore.setState({visible:false});
    else if(change==="identity") useAgentStore.setState({userEmail:"other@example.com"});
    else {store.restore(); store.minimize();}
  });
  view.unmount();
  expect(useAgentStore.getState().minimized).toBe(true);
});
