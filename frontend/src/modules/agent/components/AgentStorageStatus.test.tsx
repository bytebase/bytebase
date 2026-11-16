import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { act } from "react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { AgentStorageStatus } from "./AgentStorageStatus";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

const pressure = { reason: "budget" as const, phase: "saving" as const, bytes: 3145728, limitBytes: 3145728 };

describe("AgentStorageStatus", () => {
  test("stays absent without pressure", () => {
    const { container } = render(<AgentStorageStatus onRetrySaving={vi.fn()} />);
    expect(container).toBeEmptyDOMElement();
  });

  test.each(["budget", "quota", "unavailable"] as const)("shows only a save failure and Retry for %s without moving focus", (reason) => {
    const draft = document.createElement("textarea");
    document.body.appendChild(draft);
    draft.focus();
    render(<AgentStorageStatus pressure={{ ...pressure, reason, phase: "admission", otherScopeBytes: 1024 }} onRetrySaving={vi.fn()} />);
    expect(screen.getByRole("region", { name: "agent.storage.title" })).toHaveTextContent(/^agent.storage.save-failedagent.storage.retry$/);
    expect(screen.getAllByRole("button")).toHaveLength(1);
    expect(screen.getByRole("status")).toHaveTextContent("agent.storage.save-failed");
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
    rerender(<AgentStorageStatus pressure={{ ...pressure, bytes: 42, reason: "quota" }} onRetrySaving={onRetrySaving} />);
    fireEvent.click(screen.getByRole("button", { name: "agent.storage.retry" }));
    expect(onRetrySaving).toHaveBeenCalledOnce();
    expect(screen.getByRole("button")).toBeDisabled();
    expect(screen.getByRole("button")).toHaveAttribute("aria-busy", "true");
    await act(async () => finish(true));
    expect(screen.getByRole("region")).toHaveTextContent(/^agent.storage.save-failedagent.storage.retry$/);
    expect(screen.getByRole("button")).toBeEnabled();
    expect(fetch).not.toHaveBeenCalled();
    rerender(<AgentStorageStatus onRetrySaving={onRetrySaving} />);
    expect(screen.queryByRole("region")).toBeNull();
  });

  test.each([false, "throw"])("keeps unsuccessful retries recoverable: %s", async (result) => {
    const onRetrySaving = vi.fn(async () => true);
    if (result === false) onRetrySaving.mockResolvedValueOnce(false);
    else onRetrySaving.mockRejectedValueOnce(new Error("Unavailable"));
    render(<AgentStorageStatus pressure={pressure} onRetrySaving={onRetrySaving} />);
    await act(async () => fireEvent.click(screen.getByRole("button")));
    expect(screen.getByRole("region")).toHaveTextContent(/^agent.storage.save-failedagent.storage.retry$/);
    expect(screen.queryByRole("alert")).toBeNull();
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
});
