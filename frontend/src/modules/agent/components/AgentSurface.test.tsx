import { render, screen, waitFor } from "@testing-library/react";
import { useEffect } from "react";
import { expect, test, vi } from "vitest";
import { getLayerRoot } from "@/components/ui/layer";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { SessionExpiredSurface } from "@/app/components/SessionExpiredSurface";
import { lazyExtractDomTree } from "../dom";
import { AgentSurface } from "./AgentSurface";

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
