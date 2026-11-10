import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { AgentPageHelpButton } from "./AgentPageHelpButton";

const mocks = vi.hoisted(() => ({ open: vi.fn() }));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: { defaultValue?: string }) =>
      options?.defaultValue ?? key,
  }),
}));

vi.mock("../store/agent", () => ({
  useAgentStore: { getState: () => ({ open: mocks.open }) },
}));

describe("AgentPageHelpButton", () => {
  beforeEach(() => mocks.open.mockReset());

  test("opens the existing Agent window", () => {
    render(<AgentPageHelpButton />);
    const helpButton = screen.getByRole("button", {
      name: "Ask Agent",
    });
    expect(helpButton.className.split(" ")).toContain("h-9");
    expect(helpButton).toHaveClass("text-accent", "bg-transparent");
    expect(helpButton).not.toHaveClass("border");
    expect(helpButton.querySelector(".lucide-bot")).toBeTruthy();
    fireEvent.click(helpButton);
    expect(mocks.open).toHaveBeenCalledOnce();
  });
});
