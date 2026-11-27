import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { getLayerRoot } from "@/components/ui/layer";
import { AgentContextPicker } from "./AgentContextPicker";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useProjectList: () => ({
    projects: [],
    isLoading: false,
    isFetchingMore: false,
    hasMore: false,
    loadMore: vi.fn(),
  }),
}));

afterEach(() => {
  cleanup();
  document.body.innerHTML = "";
});

describe("AgentContextPicker", () => {
  test("keeps the focused popup fixed to the viewport when opened and reopened", async () => {
    const agentLayer = getLayerRoot("agent");
    render(
      <div style={{ position: "fixed" }}>
        <AgentContextPicker
          context="workspaces/test"
          contextTitle="Test Workspace"
          workspace={{ name: "workspaces/test", title: "Test Workspace" }}
          disabled={false}
          onSelect={vi.fn()}
        />
      </div>,
      { container: agentLayer }
    );
    const trigger = screen.getByRole("button", {
      name: "agent.context: Test Workspace",
    });

    for (let opening = 0; opening < 2; opening++) {
      fireEvent.click(trigger);
      const input = await screen.findByRole("textbox", {
        name: "common.filter-by-name",
      });
      const positioner = screen.getByRole("presentation");

      // Base UI starts every popup fixed and transparent before positioning it.
      await waitFor(() => expect(positioner.style.opacity).not.toBe("0"));
      expect(positioner.style.position).toBe("fixed");
      expect(agentLayer.contains(positioner)).toBe(true);
      await waitFor(() => expect(input).toHaveFocus());

      fireEvent.click(trigger);
      await waitFor(() => expect(input).not.toBeInTheDocument());
    }
  });
});
