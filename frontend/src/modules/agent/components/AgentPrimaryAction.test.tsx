import { act, type ReactElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { AgentPrimaryAction, AgentPrimaryActionLink } from "./AgentPrimaryAction";

const mocks = vi.hoisted(() => ({ push: vi.fn() }));
vi.mock("@/app/router", () => ({
  router: { resolve: () => ({ href: "/settings/global-masking" }), push: mocks.push },
}));

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
const roots: Root[] = [];
function render(element: ReactElement) {
  const container = document.createElement("div");
  const root = createRoot(container);
  roots.push(root);
  act(() => root.render(element));
  return container;
}
afterEach(() => {
  act(() => { for (const root of roots.splice(0)) root.unmount(); });
  vi.clearAllMocks();
});

describe("AgentPrimaryAction", () => {
  test("matches the compact ghost design with a trailing navigation icon", () => {
    const click = vi.fn();
    const container = render(<AgentPrimaryAction onClick={click}>Open rule draft</AgentPrimaryAction>);
    const button = container.querySelector("button")!;
    expect(button).toHaveClass("h-7", "text-xs", "text-accent", "bg-transparent", "hover:bg-accent/10", "gap-1.5");
    expect(button).not.toHaveClass("border", "h-auto", "hover:underline");
    expect(button.title).toBe("Open rule draft");
    expect(button.querySelector("span")).toHaveClass("truncate");
    expect(button.lastElementChild).toHaveClass("lucide-arrow-up-right", "size-4", "shrink-0");
    act(() => button.click());
    expect(click).toHaveBeenCalledOnce();
  });

  test.each([
    { actionKind: "apply" as const, loading: false, completed: false, icon: "lucide-file-pen-line", disabled: false },
    { actionKind: "apply" as const, loading: true, completed: false, icon: "lucide-loader-circle", disabled: true },
    { actionKind: "apply" as const, loading: false, completed: true, icon: "lucide-check", disabled: true },
  ])("keeps the same dimensions for $icon", ({ icon, disabled, ...presentation }) => {
    const click = vi.fn();
    const container = render(<AgentPrimaryAction {...presentation} onClick={click}>Apply to draft</AgentPrimaryAction>);
    const button = container.querySelector("button")!;
    expect(button).toHaveClass("h-7", "text-xs");
    expect(button.querySelector("svg")).toHaveClass(icon, "size-4");
    expect(button.disabled).toBe(disabled);
    act(() => button.click());
    expect(click).toHaveBeenCalledTimes(disabled ? 0 : 1);
  });

  test("retains an explicit disabled state", () => {
    const container = render(<AgentPrimaryAction disabled>Open rule draft</AgentPrimaryAction>);
    expect(container.querySelector("button")?.disabled).toBe(true);
  });

  test("preserves native navigation and modified-click behavior for links", () => {
    const container = render(<AgentPrimaryActionLink to="/settings/global-masking">Open masking settings</AgentPrimaryActionLink>);
    const link = container.querySelector("a")!;
    expect(link.getAttribute("href")).toBe("/settings/global-masking");
    expect(link).toHaveClass("h-7", "text-accent", "hover:bg-accent/10");
    expect(link.title).toBe("Open masking settings");
    expect(link.lastElementChild).toHaveClass("lucide-arrow-up-right", "size-4");
    act(() => link.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, metaKey: true })));
    expect(mocks.push).not.toHaveBeenCalled();
    act(() => link.click());
    expect(mocks.push).toHaveBeenCalledWith("/settings/global-masking");
  });

  test("does not navigate a loading link", () => {
    const container = render(<AgentPrimaryActionLink to="/settings/global-masking" loading>Open masking settings</AgentPrimaryActionLink>);
    const link = container.querySelector("a")!;
    expect(link.getAttribute("aria-disabled")).toBe("true");
    act(() => link.click());
    expect(mocks.push).not.toHaveBeenCalled();
  });
});
