import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { PromptInput } from "./PromptInput";

const mocks = vi.hoisted(() => ({
  eventsOn: vi.fn(() => vi.fn()),
  setPendingPreInput: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/utils", () => ({
  keyboardShortcutStr: (value: string) => value,
}));

vi.mock("./context", () => ({
  useAIContext: () => ({
    pendingPreInput: undefined,
    setPendingPreInput: mocks.setPendingPreInput,
    events: { on: mocks.eventsOn },
  }),
}));

afterEach(() => {
  vi.clearAllMocks();
});

describe("PromptInput", () => {
  test("allows only vertical resizing within half the viewport", () => {
    render(<PromptInput onEnter={vi.fn()} />);

    const textarea = screen.getByRole("textbox");
    expect(textarea).toHaveClass("resize-y", "max-h-[50vh]");
    expect(textarea).not.toHaveClass("resize-x", "resize");
  });

  test("preserves a manually selected height while editing", () => {
    render(<PromptInput onEnter={vi.fn()} />);

    const textarea = screen.getByRole("textbox") as HTMLTextAreaElement;
    vi.spyOn(textarea, "getBoundingClientRect").mockReturnValue({
      bottom: 100,
      height: 80,
      left: 0,
      right: 300,
      top: 20,
      width: 300,
      x: 0,
      y: 20,
      toJSON: () => ({}),
    });
    Object.defineProperty(textarea, "scrollHeight", {
      configurable: true,
      value: 120,
    });

    fireEvent.pointerDown(textarea, { clientX: 295, clientY: 95 });
    textarea.style.height = "280px";
    fireEvent.change(textarea, { target: { value: "Explain this query" } });

    expect(textarea.style.height).toBe("280px");
  });
});
