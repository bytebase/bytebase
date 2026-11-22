import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { Button } from "@/components/ui/button";
import { AgentHelpButton } from "./AgentHelpButton";

afterEach(cleanup);

describe("AgentHelpButton", () => {
  test.each([false, true])(
    "supports neutral outline styling with iconOnly=%s",
    (iconOnly) => {
      render(
        <AgentHelpButton appearance="outline" size="sm" iconOnly={iconOnly}>
          Agent
        </AgentHelpButton>
      );
      const help = screen.getByRole("button", { name: "Agent" });
      expect(help).toHaveClass(
        "h-7",
        "border",
        "border-control-border",
        "text-control",
        "bg-transparent"
      );
      expect(help).not.toHaveClass("text-accent", "hover:bg-accent/10");
      expect(help.querySelector("svg")).toHaveClass("shrink-0");
      if (iconOnly) {
        expect(help).toHaveClass("w-7", "px-0");
        expect(help).toHaveTextContent("");
      } else {
        expect(help).toHaveTextContent("Agent");
      }
    }
  );

  test("matches ordinary page commands with primary ghost styling", () => {
    const click = vi.fn();
    render(<><AgentHelpButton onClick={click}>Ask Agent</AgentHelpButton><Button>Create rule</Button></>);
    const help = screen.getByRole("button", { name: "Ask Agent" });
    expect(help).toHaveClass("h-9", "px-3", "text-sm", "text-accent", "bg-transparent");
    expect(screen.getByRole("button", { name: "Create rule" })).toHaveClass("h-9", "px-3", "text-sm");
    expect(help).not.toHaveClass("border", "h-auto");
    expect(help.querySelector("svg")).toHaveAttribute("width", "16");
    expect(help.querySelector("svg")).toHaveAttribute("aria-hidden", "true");
    expect(help).toHaveAttribute("type", "button");
    fireEvent.click(help);
    expect(click).toHaveBeenCalledOnce();
  });

  test.each([
    ["xs", "h-6", "w-6", "14"],
    ["sm", "h-7", "w-7", "16"],
    ["md", "h-9", "w-9", "16"],
    ["lg", "h-10", "w-10", "20"],
  ] as const)("keeps %s icon-only help square and labeled", (size, height, width, icon) => {
    render(<AgentHelpButton size={size} iconOnly>Help with expression</AgentHelpButton>);
    const help = screen.getByRole("button", { name: "Help with expression" });
    expect(help).toHaveClass(height, width, "px-0", "text-accent");
    expect(help).toHaveTextContent("");
    expect(help).toHaveAttribute("title", "Help with expression");
    expect(help.querySelector("svg")).toHaveAttribute("width", icon);
  });

  test("bounds long localized labels and preserves disabled behavior", () => {
    const click = vi.fn();
    const label = "Ask Agent to review this masking rule and its database scope";
    render(<AgentHelpButton disabled onClick={click}>{label}</AgentHelpButton>);
    const help = screen.getByRole("button", { name: label });
    expect(help).toHaveClass("max-w-full", "min-w-0");
    expect(help.querySelector("span")).toHaveClass("min-w-0", "truncate");
    expect(help).toBeDisabled();
    fireEvent.click(help);
    expect(click).not.toHaveBeenCalled();
  });
});
