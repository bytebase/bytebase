import { act, fireEvent, render, renderHook, screen, waitFor } from "@testing-library/react";
import { useEffect } from "react";
import { expect, test, vi } from "vitest";
import { getLayerRoot } from "@/components/ui/layer";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { SessionExpiredSurface } from "@/app/components/SessionExpiredSurface";
import { lazyExtractDomTree } from "../dom";
import { AgentSurface, useAgentDockBounds } from "./AgentSurface";

vi.mock("@/routes/auth/SigninPage", () => ({
  SigninPage: () => <button type="button">Sign in again</button>,
}));

test("retains the same Agent-layer subtree in every presentation", async () => {
  const unmounted = vi.fn();
  const Child = () => {
    useEffect(() => () => { unmounted(); }, []);
    return <button type="button">Answer private Agent question</button>;
  };
  const child = <Child />;
  const outside = document.createElement("button");
  outside.textContent = "Visible editor action";
  document.body.appendChild(outside);
  const view = render(<AgentSurface presentation="floating">{child}</AgentSurface>);
  try {
    const button = screen.getByRole("button", {name: "Answer private Agent question"});
    const frame = button.closest("[data-agent-window]");
    expect(frame?.parentElement).toBe(getLayerRoot("agent"));
    view.rerender(<AgentSurface presentation="sql-editor-docked">{child}</AgentSurface>);
    expect(button.closest("[data-agent-window]")).toBe(frame);
    const {tree} = await lazyExtractDomTree();
    expect(tree).toContain("Visible editor action");
    expect(tree).not.toContain("Answer private Agent question");
    view.rerender(<AgentSurface presentation="sql-editor-overlay">{child}</AgentSurface>);
    expect(frame?.parentElement).toBe(getLayerRoot("agent"));
    expect(getLayerRoot("agent").querySelector("button")).toBe(button);
    expect(unmounted).not.toHaveBeenCalled();
  } finally {
    view.unmount();
    outside.remove();
  }
});

test("docked controls stay accessible above ordinary dialogs but not session recovery", async () => {
  const content = (critical: boolean) => <>
    <AgentSurface presentation="sql-editor-docked"><button type="button">Agent control</button></AgentSurface>
    <Dialog open><DialogContent><DialogTitle>Ordinary dialog</DialogTitle></DialogContent></Dialog>
    {critical && <SessionExpiredSurface currentPath="/sql-editor" />}
  </>;
  const view = render(content(false));
  try {
    await waitFor(() => expect(screen.getByRole("dialog", {name: "Ordinary dialog"})).toBeVisible());
    expect(screen.getByRole("button", {name: "Agent control"})).toBeVisible();
    view.rerender(content(true));
    await waitFor(() => expect(getLayerRoot("agent")).toHaveAttribute("aria-hidden", "true"));
    expect(screen.queryByRole("button", {name: "Agent control"})).toBeNull();
    expect(screen.getByRole("button", {name: "Sign in again"})).toBeVisible();
  } finally {
    view.unmount();
  }
});

test("commits a dock ratio only for keyboard or completed pointer resizing", () => {
  const mainColumn = document.createElement("div");
  const slot = document.createElement("div");
  const resizer = document.createElement("div");
  mainColumn.append(resizer, slot);
  document.body.append(mainColumn);
  let available = 1200;
  vi.spyOn(mainColumn, "getBoundingClientRect").mockImplementation(() => new DOMRect(0, 0, available, 720));
  vi.spyOn(slot, "getBoundingClientRect").mockImplementation(() => new DOMRect(0, 0, parseFloat(slot.style.width) || 0, 720));
  vi.spyOn(resizer, "getBoundingClientRect").mockImplementation(() => new DOMRect(0, 0, resizer.hidden ? 0 : 4, 720));
  resizer.setPointerCapture = vi.fn();
  const settlePaneRatio = vi.fn();
  const host = {id:"editor", mainColumn, slot, resizer, paneRatio:0.4, settlePaneRatio, captureContext: () => ({kind:"pending"} as const)};
  const view = renderHook(() => useAgentDockBounds(host));
  try {
    expect(slot.style.width).toBe("480px");
    expect(settlePaneRatio).not.toHaveBeenCalled();
    fireEvent.keyDown(resizer, {key:"ArrowLeft"});
    expect(settlePaneRatio).toHaveBeenLastCalledWith(496 / 1200);
    fireEvent.pointerDown(resizer, {pointerId:1, button:0, clientX:720});
    fireEvent.pointerMove(resizer, {pointerId:1, clientX:600});
    expect(settlePaneRatio).toHaveBeenCalledTimes(1);
    fireEvent.pointerUp(resizer, {pointerId:1, clientX:600});
    expect(settlePaneRatio).toHaveBeenLastCalledWith(616 / 1200);
    available = 1078.5;
    act(() => window.dispatchEvent(new Event("resize")));
    fireEvent.keyDown(resizer, {key:"End"});
    expect(resizer.getAttribute("aria-valuemax")).toBe("594");
    expect(resizer.getAttribute("aria-valuenow")).toBe("594");
    expect(slot.style.width).toBe("594px");
    expect(settlePaneRatio).toHaveBeenLastCalledWith(594 / 1078.5);
    available = 820;
    act(() => window.dispatchEvent(new Event("resize")));
    expect(resizer.hidden).toBe(true);
    expect(slot.style.width).toBe("");
    expect(settlePaneRatio).toHaveBeenCalledTimes(3);
  } finally {
    view.unmount();
    mainColumn.remove();
  }
});
