import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { act } from "react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { AgentStorageStatus } from "./AgentStorageStatus";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

const pressure = { reason: "unavailable" as const, phase: "saving" as const, bytes: 3145728, limitBytes: 3145728 };
const fullPressure = { ...pressure, reason: "budget" as const };

describe("AgentStorageStatus", () => {
  test("places the outlined recovery action in its own bottom-right row", () => {
    render(<AgentStorageStatus pressure={pressure} onRetrySaving={vi.fn()} />);
    const region = screen.getByRole("alert");
    const retry = screen.getByRole("button", { name: "agent.storage.retry" });
    expect(region).toHaveClass("rounded-sm", "border-0", "bg-error/10", "px-3", "py-2", "text-sm", "text-error", "shadow-none");
    expect(region).not.toHaveClass("text-xs", "text-warning", "bg-warning/10");
    const header = region.querySelector("[data-agent-storage-header]");
    expect(header).toContainElement(screen.getByText("agent.storage.save-failed"));
    expect(header).not.toContainElement(retry);
    expect(region.querySelector("[data-agent-error-footer]")).toContainElement(retry);
    expect(region.querySelector(":scope > svg")).toBeNull();
    expect(retry).toHaveClass("h-auto", "min-h-7", "max-w-full", "whitespace-normal", "wrap-anywhere", "gap-1.5", "border");
    expect(retry.querySelector("svg")).toHaveAttribute("width", "16");
    expect(retry.querySelector("svg")).toHaveAttribute("height", "16");
    expect(retry.querySelector("svg")).toHaveClass("shrink-0");
  });

  test("keeps Error details and expanded diagnostics above recovery actions", async () => {
    const rawError = "QuotaExceededError: Browser storage is full";
    const fetch = vi.fn();
    vi.stubGlobal("fetch", fetch);
    const onRetrySaving = vi.fn(async () => true);
    render(<AgentStorageStatus pressure={pressure} error={rawError} onRetrySaving={onRetrySaving} />);
    const region = screen.getByRole("alert", { name: "agent.storage.title" });
    const header = region.querySelector<HTMLElement>("[data-agent-storage-header]");
    expect(header).not.toBeNull();
    const status = screen.getByText("agent.storage.save-failed");
    const retry = screen.getByRole("button", { name: "agent.storage.retry" });
    const details = screen.getByRole("button", { name: "error-page.error-details" });
    expect(header).toContainElement(status);
    expect(header).not.toContainElement(retry);
    expect(header).not.toContainElement(details);
    expect(header && (header.compareDocumentPosition(details) & Node.DOCUMENT_POSITION_FOLLOWING)).toBeTruthy();
    expect(region.querySelectorAll("svg.lucide-triangle-alert")).toHaveLength(0);
    expect(region.querySelector(":scope > svg")).toBeNull();
    const actions = region.querySelector("[data-agent-error-actions]");
    expect(actions).toHaveClass("flex", "min-w-0", "flex-wrap", "justify-end");
    expect(actions).toContainElement(retry);
    expect(actions).not.toContainElement(details);
    const footer = region.querySelector("[data-agent-error-footer]");
    expect(footer).toContainElement(details);
    expect(footer).toContainElement(retry);
    expect(details).toHaveAttribute("data-agent-error-details");
    expect(details).toHaveClass("text-xs", "underline", "decoration-dotted");
    expect(details).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText(rawError)).toBeNull();
    await act(async () => fireEvent.click(details));
    expect(details).toHaveAttribute("aria-expanded", "true");
    const diagnostic = screen.getByText(rawError);
    expect(region).toContainElement(diagnostic);
    expect(header).not.toContainElement(diagnostic);
    expect(footer).toContainElement(diagnostic);
    expect(actions).not.toContainElement(diagnostic);
    expect(diagnostic.compareDocumentPosition(actions!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(header && (header.compareDocumentPosition(diagnostic) & Node.DOCUMENT_POSITION_FOLLOWING)).toBeTruthy();
    expect(within(region).getAllByRole("button")).toHaveLength(2);
    expect(onRetrySaving).not.toHaveBeenCalled();
    await act(async () => fireEvent.click(retry));
    expect(onRetrySaving).toHaveBeenCalledOnce();
    expect(fetch).not.toHaveBeenCalled();
    await act(async () => fireEvent.click(details));
    expect(details).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText(rawError)).toBeNull();
    expect(onRetrySaving).toHaveBeenCalledOnce();
  });

  test("stays absent without pressure", () => {
    const { container } = render(<AgentStorageStatus onRetrySaving={vi.fn()} />);
    expect(container).toBeEmptyDOMElement();
  });

  test("shows only a save failure and Retry for unavailable storage without moving focus", () => {
    const draft = document.createElement("textarea");
    document.body.appendChild(draft);
    draft.focus();
    render(<AgentStorageStatus pressure={{ ...pressure, phase: "admission", otherScopeBytes: 1024 }} onRetrySaving={vi.fn()} />);
    expect(screen.getByRole("alert", { name: "agent.storage.title" })).toHaveTextContent(/^agent.storage.save-failedagent.storage.retry$/);
    expect(screen.getAllByRole("button")).toHaveLength(1);
    expect(screen.getByText("agent.storage.save-failed")).toBeVisible();
    expect(draft).toHaveFocus();
    draft.remove();
  });

  test("keeps retry pending across pressure changes and delegates success clearing to the parent", async () => {
    const fetch = vi.fn();
    vi.stubGlobal("fetch", fetch);
    let finish!: (saved: boolean) => void;
    const onRetrySaving = vi.fn(() => new Promise<boolean>((resolve) => { finish = resolve; }));
    const { rerender } = render(<AgentStorageStatus pressure={pressure} onRetrySaving={onRetrySaving} />);
    fireEvent.click(screen.getByRole("button", { name: "agent.storage.retry" }));
    rerender(<AgentStorageStatus pressure={{ ...pressure, bytes: 42 }} onRetrySaving={onRetrySaving} />);
    fireEvent.click(screen.getByRole("button", { name: "agent.storage.retry" }));
    expect(onRetrySaving).toHaveBeenCalledOnce();
    expect(screen.getByRole("button")).toBeDisabled();
    expect(screen.getByRole("button")).toHaveAttribute("aria-busy", "true");
    await act(async () => finish(true));
    expect(screen.getByRole("alert")).toHaveTextContent(/^agent.storage.save-failedagent.storage.retry$/);
    expect(screen.getByRole("button")).toBeEnabled();
    expect(fetch).not.toHaveBeenCalled();
    rerender(<AgentStorageStatus onRetrySaving={onRetrySaving} />);
    expect(screen.queryByRole("alert")).toBeNull();
  });

  test.each([false, "throw"])("keeps unsuccessful retries recoverable: %s", async (result) => {
    const onRetrySaving = vi.fn(async () => true);
    if (result === false) onRetrySaving.mockResolvedValueOnce(false);
    else onRetrySaving.mockRejectedValueOnce(new Error("Unavailable"));
    render(<AgentStorageStatus pressure={pressure} onRetrySaving={onRetrySaving} />);
    await act(async () => fireEvent.click(screen.getByRole("button")));
    expect(screen.getByRole("alert")).toHaveTextContent(/^agent.storage.save-failedagent.storage.retry$/);
    expect(screen.getAllByRole("alert")).toHaveLength(1);
    expect(screen.getByRole("button")).toBeEnabled();
    await act(async () => fireEvent.click(screen.getByRole("button")));
    expect(onRetrySaving).toHaveBeenCalledTimes(2);
  });

  test("respects disabled", () => {
    const onRetrySaving = vi.fn();
    render(<AgentStorageStatus pressure={pressure} onRetrySaving={onRetrySaving} disabled />);
    expect(screen.getByRole("button")).toBeDisabled();
    fireEvent.click(screen.getByRole("button"));
    expect(onRetrySaving).not.toHaveBeenCalled();
  });

  test.each(["budget", "quota"] as const)("offers confirmed recovery instead of a futile Retry for %s", async (reason) => {
    const onFreeSpaceAndRetry = vi.fn(async () => true);
    const onRetrySaving = vi.fn();
    render(<AgentStorageStatus pressure={{ ...fullPressure, reason }} onRetrySaving={onRetrySaving} onFreeSpaceAndRetry={onFreeSpaceAndRetry} />);
    expect(screen.getByText("agent.storage.full")).toBeVisible();
    expect(screen.queryByRole("button", { name: "agent.storage.retry" })).toBeNull();
    const recovery = screen.getByRole("button", { name: "agent.storage.free-and-retry" });
    expect(recovery.parentElement).toHaveClass("justify-end", "flex-wrap");
    expect(recovery.querySelector("svg")).toHaveAttribute("width", "16");
    expect(recovery.querySelector("svg")).toHaveAttribute("height", "16");
    await act(async () => fireEvent.click(recovery));
    const dialog = await screen.findByRole("dialog", { name: "agent.storage.free-title" });
    expect(dialog.closest("[data-bb-layer-family]")).toHaveAttribute("data-bb-layer-family", "agent");
    expect(dialog).toHaveClass("w-64", "p-3");
    expect(screen.queryByRole("alertdialog")).toBeNull();
    expect(within(dialog).getByRole("button", { name: "agent.storage.free-and-retry" })).toHaveClass("h-auto", "min-h-7", "max-w-full", "whitespace-normal", "wrap-anywhere");
    expect(within(dialog).getByRole("button", { name: "agent.storage.free-and-retry" }).querySelector("svg")).toBeNull();
    expect(onFreeSpaceAndRetry).not.toHaveBeenCalled();
    expect(screen.queryByRole("checkbox")).toBeNull();
    await act(async () => fireEvent.click(within(dialog).getByRole("button", { name: "common.cancel" })));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    expect(onFreeSpaceAndRetry).not.toHaveBeenCalled();
    expect(onRetrySaving).not.toHaveBeenCalled();
  });

  test("offers only confirmation actions and closes after successful recovery", async () => {
    const onFreeSpaceAndRetry = vi.fn(async () => true);
    render(<AgentStorageStatus pressure={fullPressure} onRetrySaving={vi.fn()} onFreeSpaceAndRetry={onFreeSpaceAndRetry} />);
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "agent.storage.free-and-retry" })));
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).queryByRole("checkbox")).toBeNull();
    expect(within(dialog).getAllByRole("button")).toHaveLength(2);
    await act(async () => fireEvent.click(within(dialog).getByRole("button", { name: "agent.storage.free-and-retry" })));
    expect(onFreeSpaceAndRetry).toHaveBeenCalledExactlyOnceWith();
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  test("prevents duplicate recovery and dismissal while freeing space", async () => {
    let finish!: (saved: boolean) => void;
    const onFreeSpaceAndRetry = vi.fn(() => new Promise<boolean>((resolve) => { finish = resolve; }));
    render(<AgentStorageStatus pressure={fullPressure} onRetrySaving={vi.fn()} onFreeSpaceAndRetry={onFreeSpaceAndRetry} />);
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "agent.storage.free-and-retry" })));
    const dialog = await screen.findByRole("dialog");
    const confirm = within(dialog).getByRole("button", { name: "agent.storage.free-and-retry" });
    const cancel = within(dialog).getByRole("button", { name: "common.cancel" });
    fireEvent.click(confirm);
    fireEvent.click(confirm);
    fireEvent.click(cancel);
    fireEvent.keyDown(dialog, { key: "Escape" });
    expect(onFreeSpaceAndRetry).toHaveBeenCalledExactlyOnceWith();
    expect(confirm).toBeDisabled();
    expect(confirm).toHaveAttribute("aria-busy", "true");
    expect(cancel).toBeDisabled();
    expect(screen.getByRole("dialog")).toBe(dialog);
    await act(async () => finish(true));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  test.each([false, "throw"])("keeps failed recovery visible and actionable without another alert: %s", async (result) => {
    const onFreeSpaceAndRetry = vi.fn(async () => true);
    if (result === false) onFreeSpaceAndRetry.mockResolvedValueOnce(false);
    else onFreeSpaceAndRetry.mockRejectedValueOnce(new Error("Storage is still full"));
    render(<AgentStorageStatus pressure={fullPressure} onRetrySaving={vi.fn()} onFreeSpaceAndRetry={onFreeSpaceAndRetry} />);
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "agent.storage.free-and-retry" })));
    const dialog = await screen.findByRole("dialog");
    await act(async () => fireEvent.click(within(dialog).getByRole("button", { name: "agent.storage.free-and-retry" })));
    expect(screen.getByRole("dialog")).toBe(dialog);
    expect(within(dialog).getByRole("status")).toHaveTextContent("agent.storage.free-failed");
    expect(within(dialog).getByRole("button", { name: "agent.storage.free-and-retry" })).toBeEnabled();
    expect(screen.getAllByRole("alert")).toHaveLength(1);
    await act(async () => fireEvent.click(within(dialog).getByRole("button", { name: "agent.storage.free-and-retry" })));
    expect(onFreeSpaceAndRetry).toHaveBeenCalledTimes(2);
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });
});
