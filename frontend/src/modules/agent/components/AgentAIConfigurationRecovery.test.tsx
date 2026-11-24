import { act, type ReactElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/app/router/handles";
import {
  AI_ASSISTANT_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
} from "@/lib/productIntro";
import { AgentAIConfigurationRecovery } from "./AgentAIConfigurationRecovery";

const mocks = vi.hoisted(() => ({
  allowConfigure: true,
  clearError: vi.fn(),
  toggle: vi.fn(),
  push: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/app/router", () => ({
  router: {
    resolve: () => ({ href: "/settings/general#ai-assistant" }),
    push: mocks.push,
  },
}));
vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: () => mocks.allowConfigure,
}));
vi.mock("../store/agent", () => {
  const state = {
    currentChatId: "chat-1",
    clearError: mocks.clearError,
    toggle: mocks.toggle,
  };
  const useAgentStore = <T,>(selector: (value: typeof state) => T) =>
    selector(state);
  useAgentStore.getState = () => state;
  return { useAgentStore };
});

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;
const roots: Root[] = [];
function render(element: ReactElement) {
  const container = document.createElement("div");
  const root = createRoot(container);
  roots.push(root);
  act(() => root.render(element));
  return container;
}

beforeEach(() => {
  mocks.allowConfigure = true;
});
afterEach(() => {
  act(() => {
    for (const root of roots.splice(0)) root.unmount();
  });
  vi.clearAllMocks();
});

describe("AgentAIConfigurationRecovery", () => {
  test("uses compact mobile spacing without changing desktop alert spacing", () => {
    const container = render(<AgentAIConfigurationRecovery />);
    const alert = container.querySelector('[role="alert"]')!;
    const actions = container.querySelector("[data-agent-ai-configuration-actions]")!;

    expect(alert).toHaveClass("p-2", "gap-x-2", "sm:px-4", "sm:py-3", "sm:gap-x-3");
    expect(actions).toHaveClass("mt-2", "sm:mt-3");
  });

  test("uses the compact primary action for configuration without changing navigation", () => {
    const container = render(<AgentAIConfigurationRecovery />);
    const link = container.querySelector("a")!;

    expect(link).toHaveAttribute("data-agent-primary-action");
    expect(link).toHaveClass(
      "h-auto",
      "min-h-7",
      "max-w-full",
      "whitespace-normal",
      "wrap-anywhere",
      "text-xs",
      "bg-transparent",
      "text-accent",
      "hover:bg-accent/10"
    );
    expect(link.textContent).toBe("agent.ai-not-configured.configure");
    expect(link.lastElementChild).toHaveClass(
      "lucide-arrow-up-right",
      "size-4"
    );
    act(() => link.click());
    expect(mocks.clearError).toHaveBeenCalledWith("chat-1");
    expect(mocks.toggle).toHaveBeenCalledOnce();
    expect(mocks.push).toHaveBeenCalledWith({
      name: SETTING_ROUTE_WORKSPACE_GENERAL,
      hash: "#ai-assistant",
      query: { [PRODUCT_INTRO_QUERY_KEY]: AI_ASSISTANT_PRODUCT_INTRO },
    });
  });

  test("does not offer configuration without permission", () => {
    mocks.allowConfigure = false;
    const container = render(<AgentAIConfigurationRecovery />);

    expect(container.querySelector("a")).toBeNull();
    expect(container.textContent).toContain(
      "agent.ai-not-configured.contact-admin"
    );
  });

  test("places only Configure in the final full-width, right-aligned wrapping row", () => {
    const container = render(<AgentAIConfigurationRecovery />);
    const alert = container.querySelector('[role="alert"]')!;
    const link = container.querySelector("a")!;
    const actions = link.parentElement!;

    expect(alert).not.toHaveClass("max-w-[80%]");
    expect(actions).toHaveClass("flex", "w-full", "flex-wrap", "justify-end", "gap-2");
    expect(actions.children).toHaveLength(1);
    expect(actions.lastElementChild).toBe(link);
    expect(container.querySelector("button")).toBeNull();
    expect(actions.parentElement?.lastElementChild).toBe(actions);
    expect(actions.textContent).not.toContain("agent.ai-not-configured.description");
  });

  test("shows contact-admin guidance without any actions for a restricted user", () => {
    mocks.allowConfigure = false;
    const container = render(<AgentAIConfigurationRecovery />);
    expect(container.textContent).toContain("agent.ai-not-configured.contact-admin");
    expect(container.querySelector("[data-agent-ai-configuration-actions]")).toBeNull();
    expect(container.querySelector("button")).toBeNull();
    expect(container.querySelector("a")).toBeNull();
  });
});
