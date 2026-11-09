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
  test("uses the compact primary action for configuration without changing navigation", () => {
    const container = render(<AgentAIConfigurationRecovery />);
    const link = container.querySelector("a")!;

    expect(link).toHaveAttribute("data-agent-primary-action");
    expect(link).toHaveClass(
      "h-7",
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

  test("keeps dismiss as a secondary action", () => {
    const onDismiss = vi.fn();
    const container = render(
      <AgentAIConfigurationRecovery onDismiss={onDismiss} />
    );
    const dismiss = container.querySelector("button")!;

    expect(dismiss.textContent).toBe("common.dismiss");
    expect(dismiss).toHaveClass("border", "h-7");
    expect(dismiss).not.toHaveAttribute("data-agent-primary-action");
    act(() => dismiss.click());
    expect(onDismiss).toHaveBeenCalledOnce();
    expect(mocks.clearError).toHaveBeenCalledWith("chat-1");
    expect(mocks.toggle).not.toHaveBeenCalled();
    expect(mocks.push).not.toHaveBeenCalled();
  });

  test("does not offer configuration without permission", () => {
    mocks.allowConfigure = false;
    const container = render(<AgentAIConfigurationRecovery />);

    expect(container.querySelector("a")).toBeNull();
    expect(container.textContent).toContain(
      "agent.ai-not-configured.contact-admin"
    );
  });
});
