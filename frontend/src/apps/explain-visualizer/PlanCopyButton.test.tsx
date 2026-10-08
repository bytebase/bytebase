import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { PlanCopyButton } from "./PlanCopyButton";

const writeText = vi.fn<(text: string) => Promise<void>>();

const button = () => screen.getByTestId("plan-copy-button");

beforeEach(() => {
  vi.useFakeTimers({ shouldAdvanceTime: true });
  writeText.mockReset().mockResolvedValue(undefined);
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText },
  });
});

afterEach(() => {
  vi.useRealTimers();
});

describe("PlanCopyButton", () => {
  test("names what it copies until it is used", () => {
    render(<PlanCopyButton content="SELECT 1" label="Copy query" />);
    expect(button()).toHaveTextContent("Copy query");
  });

  test("puts the content on the clipboard and says so", async () => {
    render(<PlanCopyButton content="SELECT 1" label="Copy query" />);

    fireEvent.click(button());

    expect(writeText).toHaveBeenCalledWith("SELECT 1");
    await waitFor(() => expect(button()).toHaveTextContent("Copied"));
  });

  test("goes back to offering the copy after the confirmation", async () => {
    render(<PlanCopyButton content="SELECT 1" label="Copy query" />);

    fireEvent.click(button());
    await waitFor(() => expect(button()).toHaveTextContent("Copied"));

    vi.advanceTimersByTime(2000);
    await waitFor(() => expect(button()).toHaveTextContent("Copy query"));
  });

  test("reports a clipboard the browser refused rather than lying", async () => {
    writeText.mockRejectedValue(new Error("denied"));
    // The clipboard library then falls back to a hidden textarea; jsdom
    // implements no `execCommand` at all, so that fails too.
    Object.defineProperty(document, "execCommand", {
      configurable: true,
      value: () => false,
    });
    render(<PlanCopyButton content="SELECT 1" label="Copy query" />);

    fireEvent.click(button());

    await waitFor(() => expect(button()).toHaveTextContent("Copy failed"));
  });

  test("announces the outcome to a reader looking elsewhere", async () => {
    const { container } = render(
      <PlanCopyButton content="SELECT 1" label="Copy query" />
    );
    const live = container.querySelector("[aria-live='polite']");

    expect(live).toHaveTextContent("");
    fireEvent.click(button());
    await waitFor(() => expect(live).toHaveTextContent("Copied"));
  });

  test("can be disabled when there is nothing to copy", () => {
    render(<PlanCopyButton content="" label="Copy query" disabled />);

    fireEvent.click(button());
    expect(writeText).not.toHaveBeenCalled();
  });
});
