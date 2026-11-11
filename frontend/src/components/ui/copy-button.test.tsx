import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { LayerFamily } from "./layer";
import { CopyButton } from "./copy-button";

const mocks = vi.hoisted(() => ({
  writeTextToClipboard: vi.fn(),
  notify: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/lib/clipboard", () => ({
  writeTextToClipboard: mocks.writeTextToClipboard,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => ({ notify: mocks.notify }) },
}));

beforeEach(() => {
  vi.useFakeTimers();
  mocks.writeTextToClipboard.mockReset();
  mocks.writeTextToClipboard.mockResolvedValue(true);
  mocks.notify.mockReset();
});

afterEach(() => {
  cleanup();
  document.body.innerHTML = "";
  vi.useRealTimers();
});

describe("CopyButton", () => {
  test.each<{ tooltipLayer?: LayerFamily; expectedLayer: LayerFamily }>([
    { expectedLayer: "overlay" },
    { tooltipLayer: "agent", expectedLayer: "agent" },
  ])("mounts the tooltip into $expectedLayer", async ({ tooltipLayer, expectedLayer }) => {
    render(<CopyButton content="Original text" tooltipLayer={tooltipLayer} />);
    const button = screen.getByRole("button", { name: "common.copy" });

    await act(async () => {
      fireEvent.focus(button);
      vi.advanceTimersByTime(100);
    });

    expect(document.getElementById(`bb-react-layer-${expectedLayer}`)?.textContent ?? "").toContain("common.copy");
    const otherLayer = expectedLayer === "agent" ? "overlay" : "agent";
    expect(document.getElementById(`bb-react-layer-${otherLayer}`)?.textContent ?? "").not.toContain("common.copy");
    expect(button).toHaveClass("h-6", "px-1");
    expect(button.querySelector(".lucide-copy")).toHaveClass("size-3.5");
  });

  test("copies the current content and shows successful feedback for two seconds", async () => {
    const content = vi.fn(() => "Question with @Database and\na second line");
    render(<CopyButton content={content} tooltipLayer="agent" />);
    const button = screen.getByRole("button", { name: "common.copy" });
    expect(content).not.toHaveBeenCalled();

    await act(async () => fireEvent.click(button));

    expect(content).toHaveBeenCalledOnce();
    expect(mocks.writeTextToClipboard).toHaveBeenCalledExactlyOnceWith("Question with @Database and\na second line");
    expect(mocks.notify).toHaveBeenCalledExactlyOnceWith({
      module: "bytebase",
      style: "SUCCESS",
      title: "common.copied",
    });
    expect(button.querySelector(".lucide-check")).toHaveClass("size-3.5", "text-success");
    act(() => vi.advanceTimersByTime(2000));
    expect(button.querySelector(".lucide-copy")).not.toBeNull();
  });

  test("reports failed copying without successful feedback", async () => {
    mocks.writeTextToClipboard.mockResolvedValue(false);
    render(<CopyButton content="Original text" />);
    const button = screen.getByRole("button", { name: "common.copy" });

    await act(async () => fireEvent.click(button));

    expect(mocks.writeTextToClipboard).toHaveBeenCalledExactlyOnceWith("Original text");
    expect(mocks.notify).toHaveBeenCalledExactlyOnceWith({
      module: "bytebase",
      style: "CRITICAL",
      title: "common.copy-failed",
    });
    expect(button.querySelector(".lucide-check")).toBeNull();
    expect(button.querySelector(".lucide-copy")).not.toBeNull();
  });

  test("takes its accessible name from the label, keeping the generic tooltip", async () => {
    render(<CopyButton content="ana@example.com" label="Copy email for Ana" />);
    const button = screen.getByRole("button", { name: "Copy email for Ana" });

    await act(async () => {
      fireEvent.focus(button);
      vi.advanceTimersByTime(100);
    });

    expect(screen.queryByRole("button", { name: "common.copy" })).toBeNull();
    expect(document.getElementById("bb-react-layer-overlay")?.textContent ?? "").toContain("common.copy");
  });
});
