import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { act } from "react";
import { afterEach, describe, expect, test, vi } from "vitest";
import enUS from "@/locales/en-US.json";
import type { AgentChat, AgentRunCheckpoint } from "../logic/types";
import { AgentTokenBudgetStatus } from "./AgentTokenBudgetStatus";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, values?: Record<string, unknown>) => {
      const value = key.split(".").reduce<unknown>((result, part) =>
        typeof result === "object" && result !== null ? (result as Record<string, unknown>)[part] : undefined, enUS);
      return typeof value === "string" ? value.replace(/{{(\w+)}}/g, (_, name) => String(values?.[name] ?? "")) : key;
    },
  }),
}));

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const chat: AgentChat = {
  id: "chat-1", title: "Budget", createdTs: 0, updatedTs: 0, status: "running", archived: false,
  totalTokensUsed: 800,
  runBudget: { version: 1, runId: "run-1", limit: 1000, totalTokensUsed: 800, hasUnreportedUsage: false, providerAttemptCount: 0, revision: 0 },
};
const checkpoint: AgentRunCheckpoint = {
  version: 1, runId: "run-1", suspension: "token-limit", rootUserMessageId: "user-1", throughMessageId: "answer-1",
  bindings: { page: { path: "/", title: "Home" }, references: [], runMode: "default" }, maskingSections: [], phase: "working",
};

function setup(value: AgentChat | null = chat, disabled = false, recovery: { remoteBusy?: boolean; canResume?: boolean; pendingAttemptId?: string } = {}) {
  if (value?.runBudget && value.modelAttempts === undefined) {
    value = { ...value, modelAttempts: [
      { attemptId: "run-1:1", runId: "run-1", ordinal: 1, outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: value.runBudget.totalTokensUsed } },
      ...(value.runBudget.hasUnreportedUsage ? [{ attemptId: "run-1:2", runId: "run-1", ordinal: 2, outcome: "error" as const }] : []),
    ] };
  }
  const callbacks = { onIncreaseLimit: vi.fn(async (_value: number) => true), onResume: vi.fn<() => Promise<boolean | void> | boolean | void>(), onEndRun: vi.fn() };
  const view = render(<AgentTokenBudgetStatus chat={value} {...callbacks} disabled={disabled} {...recovery} />);
  return { ...view, ...callbacks };
}

async function openIncrease() {
  await act(async () => fireEvent.click(screen.getByRole("button", { name: "Increase limit" })));
  return screen.getByRole("spinbutton", { name: "This run's token limit" });
}

describe("AgentTokenBudgetStatus", () => {
  test("identifies the current run and uses the preference's K-token input units", async () => {
    setup({ ...chat, status: "paused", runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, limit: 6000, totalTokensUsed: 7202 } });
    const input = await openIncrease();
    expect(screen.getByRole("heading", { name: "Increase limit for this run" })).toBeVisible();
    expect(input).toHaveAttribute("inputmode", "decimal");
    expect(input).toHaveAttribute("min", "7.203");
    expect(input).toHaveAttribute("max", "9007199254740.991");
    expect(input).toHaveAttribute("step", "0.001");
    expect(input).toHaveDisplayValue("8");
    expect(input).toBeValid();
    expect(input).toHaveAccessibleDescription("K tokens");
    fireEvent.change(input, { target: { value: "7.202" } });
    expect(input).toHaveAccessibleDescription("K tokens Enter a limit greater than 7.202 K tokens, with up to 3 decimal places.");
    fireEvent.change(input, { target: { value: "7.203" } });
    expect(input).toBeValid();
    expect(input).toHaveAccessibleDescription("K tokens");
  });

  test.each<[number, number, string, number]>([
    [1000, 1100, "2", 2000],
    [6000, 8022, "9", 9000],
    [8000, 8000, "9", 9000],
    [10000, 8000, "11", 11000],
    [9007199253739999, 9007199253739999, "9007199253740", 9007199253740000],
    [9007199254740990, 9007199254740990, "9007199254740.991", 9007199254740991],
  ])("prefills a rounded cap above limit %i and usage %i within safe bounds", async (limit, totalTokensUsed, displayed, tokens) => {
    const paused = totalTokensUsed >= limit;
    const { onIncreaseLimit, onResume } = setup({ ...chat, status: paused ? "paused" : "running", runCheckpoint: paused ? checkpoint : undefined, runBudget: { ...chat.runBudget!, limit, totalTokensUsed } });
    const input = await openIncrease();
    expect(input).toHaveDisplayValue(displayed);
    expect(input).toBeValid();
    expect(input).not.toHaveAttribute("aria-invalid", "true");
    expect(screen.queryByRole("alert")).toBeNull();
    expect(onIncreaseLimit).not.toHaveBeenCalled();
    expect(onResume).not.toHaveBeenCalled();
    const confirm = screen.getByRole("button", { name: paused ? "Increase and resume" : "Apply" });
    expect(confirm).toBeEnabled();
    await act(async () => fireEvent.click(confirm));
    expect(onIncreaseLimit).toHaveBeenCalledExactlyOnceWith(tokens);
    expect(onResume).toHaveBeenCalledTimes(paused ? 1 : 0);
  });

  test("reopens with the valid default instead of a discarded draft", async () => {
    const { onIncreaseLimit, onResume } = setup({ ...chat, status: "paused", runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } });
    const input = await openIncrease();
    fireEvent.change(input, { target: { value: "5" } });
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Cancel" })));
    const reopened = await openIncrease();
    expect(reopened).toHaveDisplayValue("2");
    expect(reopened).toBeValid();
    expect(screen.queryByRole("alert")).toBeNull();
    expect(onIncreaseLimit).not.toHaveBeenCalled();
    expect(onResume).not.toHaveBeenCalled();
  });

  test.each<[string, number]>([
    ["1.101", 1101],
    ["12.345", 12345],
    ["2.0000", 2000],
    ["9007199254740.991", Number.MAX_SAFE_INTEGER],
  ])("stores %s K as exactly %i tokens before resuming", async (entered, tokens) => {
    const { onIncreaseLimit, onResume } = setup({ ...chat, status: "paused", runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } });
    const input = await openIncrease();
    fireEvent.change(input, { target: { value: entered } });
    expect(input).not.toHaveAttribute("aria-invalid", "true");
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Increase and resume" })));
    expect(onIncreaseLimit).toHaveBeenCalledExactlyOnceWith(tokens);
    expect(onResume).toHaveBeenCalledOnce();
  });

  test("leaves normal and legacy chats to the compact usage trigger", () => {
    const { container, rerender } = setup({ ...chat, runBudget: { ...chat.runBudget!, totalTokensUsed: 790 } });
    expect(container.textContent).toBe("");
    rerender(<AgentTokenBudgetStatus chat={null} onIncreaseLimit={vi.fn()} onResume={vi.fn()} onEndRun={vi.fn()} />);
    expect(container.textContent).toBe("");
    rerender(<AgentTokenBudgetStatus chat={{ ...chat, runBudget: undefined }} onIncreaseLimit={vi.fn()} onResume={vi.fn()} onEndRun={vi.fn()} />);
    expect(container.textContent).toBe("");
  });

  test("warns at 80% without stealing focus or showing recovery commands", () => {
    const focus = document.createElement("textarea");
    document.body.appendChild(focus);
    focus.focus();
    setup();
    expect(screen.getByRole("status")).toHaveTextContent("Approaching this run's token limit.");
    expect(screen.getByText("800 / 1,000 tokens")).toBeVisible();
    expect(screen.queryByRole("button", { name: "Resume" })).toBeNull();
    expect(screen.getAllByRole("button")).toHaveLength(1);
    expect(focus).toHaveFocus();
    focus.remove();
  });

  test("does not show a missing-usage warning while the first response is pending", () => {
    const { container } = setup({ ...chat, runId: "run-1", modelAttempts: [], runBudget: { ...chat.runBudget!, totalTokensUsed: 0, providerAttemptCount: 1 } }, false, { pendingAttemptId: "run-1:1" });
    expect(container.textContent).toBe("");
  });

  test("keeps a genuine earlier missing-usage warning while the next response is pending", () => {
    setup({ ...chat, runId: "run-1", modelAttempts: [{ attemptId: "run-1:1", runId: "run-1", ordinal: 1, outcome: "error" }], runBudget: { ...chat.runBudget!, totalTokensUsed: 0, providerAttemptCount: 2 } }, false, { pendingAttemptId: "run-1:2" });
    expect(screen.getByRole("status")).toHaveTextContent("Some token usage was not reported.");
    expect(screen.getByRole("region")).toHaveTextContent("At least 0 / 1,000 tokens");
  });

  test("offers only Increase limit and End run at the limit", async () => {
    const { onEndRun, onResume } = setup({ ...chat, status: "paused", runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } });
    expect(screen.getByRole("status")).toHaveTextContent("Run paused at its token limit.");
    expect(screen.queryByRole("button", { name: "Resume" })).toBeNull();
    expect(screen.getAllByRole("button")).toHaveLength(2);
    expect(screen.getByRole("button", { name: "Increase limit" })).toBeEnabled();
    expect(onEndRun).not.toHaveBeenCalled();
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "End run" })));
    expect(onEndRun).toHaveBeenCalledOnce();
    expect(onResume).not.toHaveBeenCalled();
  });

  test("requests an increase and resume for this run without writing defaults", async () => {
    const { onIncreaseLimit, onResume, onEndRun } = setup({ ...chat, status: "paused", runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } });
    const storageWrite = vi.spyOn(Storage.prototype, "setItem");
    const input = await openIncrease();
    expect(input.closest("#bb-react-layer-agent")).not.toBeNull();
    fireEvent.change(input, { target: { value: "2" } });
    expect(onIncreaseLimit).not.toHaveBeenCalled();
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Increase and resume" })));
    expect(onIncreaseLimit).toHaveBeenCalledExactlyOnceWith(2000);
    expect(onResume).toHaveBeenCalledOnce();
    expect(onEndRun).not.toHaveBeenCalled();
    expect(storageWrite).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: "Resume" })).toBeNull();
    storageWrite.mockRestore();
  });

  test.each(["", "0", "-1", "1", "1.1", "1.1001", "9007199254740.992"])("rejects invalid or insufficient K-token run increase %j", async (value) => {
    const { onIncreaseLimit } = setup({ ...chat, status: "paused", runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } });
    const input = await openIncrease();
    fireEvent.change(input, { target: { value } });
    expect(input).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByRole("alert")).toHaveTextContent("Enter a limit greater than 1.1 K tokens, with up to 3 decimal places.");
    expect(screen.getByRole("button", { name: "Increase and resume" })).toBeDisabled();
    expect(onIncreaseLimit).not.toHaveBeenCalled();
  });

  test("cancels an increase without applying or resuming", async () => {
    const { onIncreaseLimit, onResume } = setup({ ...chat, status: "paused", runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } });
    await openIncrease();
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Cancel" })));
    expect(screen.queryByRole("spinbutton")).toBeNull();
    expect(onIncreaseLimit).not.toHaveBeenCalled();
    expect(onResume).not.toHaveBeenCalled();
  });

  test("keeps a rejected increase and resume editable without showing Resume", async () => {
    const { onIncreaseLimit, onResume } = setup({ ...chat, status: "paused", runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } });
    onIncreaseLimit.mockResolvedValueOnce(false);
    const input = await openIncrease();
    fireEvent.change(input, { target: { value: "2" } });
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Increase and resume" })));
    expect(screen.getByRole("spinbutton")).toHaveValue(2);
    expect(screen.getByRole("alert")).toHaveTextContent("Couldn't update this run. Try again.");
    expect(screen.queryByRole("button", { name: "Resume" })).toBeNull();
    expect(onResume).not.toHaveBeenCalled();
  });

  test("prevents duplicate commands while an increase is pending", async () => {
    const { onIncreaseLimit, onResume, onEndRun } = setup({ ...chat, status: "paused", runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } });
    let finish!: (value: boolean) => void;
    onIncreaseLimit.mockImplementationOnce(() => new Promise<boolean>((resolve) => { finish = resolve; }));
    const input = await openIncrease();
    fireEvent.change(input, { target: { value: "2" } });
    fireEvent.click(screen.getByRole("button", { name: "Increase and resume" }));
    expect(screen.getByRole("spinbutton")).toBeDisabled();
    expect(screen.getByRole("button", { name: "Increase and resume" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Increase and resume" }));
    expect(screen.queryByRole("button", { name: "Resume" })).toBeNull();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "End run" })).toBeDisabled();
    expect(onIncreaseLimit).toHaveBeenCalledOnce();
    expect(onResume).not.toHaveBeenCalled();
    expect(onEndRun).not.toHaveBeenCalled();
    await act(async () => finish(true));
    expect(onResume).toHaveBeenCalledOnce();
    expect(screen.queryByRole("spinbutton")).toBeNull();
  });

  test("closes the editor after saving without waiting for the resumed run to finish", async () => {
    const exhaustedChat = { ...chat, status: "paused" as const, runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } };
    const { rerender, onIncreaseLimit, onResume, onEndRun } = setup(exhaustedChat);
    let finish!: () => void;
    onResume.mockImplementationOnce(() => new Promise<void>((resolve) => { finish = resolve; }));
    const input = await openIncrease();
    fireEvent.change(input, { target: { value: "2" } });
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Increase and resume" })));
    rerender(<AgentTokenBudgetStatus chat={{ ...exhaustedChat, runBudget: { ...exhaustedChat.runBudget, limit: 2000 } }} onIncreaseLimit={onIncreaseLimit} onResume={onResume} onEndRun={onEndRun} />);
    expect(screen.queryByRole("spinbutton")).toBeNull();
    expect(onIncreaseLimit).toHaveBeenCalledExactlyOnceWith(2000);
    expect(onResume).toHaveBeenCalledOnce();
    expect(screen.getByRole("button", { name: "Resume" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "End run" })).toBeDisabled();
    await act(async () => finish());
  });

  test("shows a prevented resume after closing the successfully saved editor", async () => {
    const { onIncreaseLimit, onResume } = setup({ ...chat, status: "paused", runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } });
    onResume.mockReturnValueOnce(false);
    const input = await openIncrease();
    fireEvent.change(input, { target: { value: "2" } });
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Increase and resume" })));
    expect(screen.queryByRole("spinbutton")).toBeNull();
    expect(screen.getByRole("alert")).toHaveTextContent("Couldn't update this run. Try again.");
    expect(onIncreaseLimit).toHaveBeenCalledExactlyOnceWith(2000);
    expect(onResume).toHaveBeenCalledOnce();
  });

  test("requires explicit Resume after the persisted budget becomes eligible", async () => {
    const { onResume } = setup({ ...chat, status: "paused", runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, limit: 2000, totalTokensUsed: 1100 } });
    expect(onResume).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Resume" })).toBeEnabled();
    expect(screen.queryByRole("button", { name: "Increase limit" })).toBeNull();
    expect(screen.getAllByRole("button")).toHaveLength(2);
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Resume" })));
    expect(onResume).toHaveBeenCalledOnce();
  });

  test("keeps a completed over-budget answer completed without recovery commands", () => {
    setup({ ...chat, status: "idle", runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } });
    expect(screen.getByRole("status")).toHaveTextContent("Run completed above its token limit.");
    expect(screen.queryByRole("button")).toBeNull();
  });

  test("does not call a completed answer above-limit when its usage equals the limit", () => {
    const { container } = setup({ ...chat, status: "idle", runBudget: { ...chat.runBudget!, totalTokensUsed: 1000 } });
    expect(container.textContent).toBe("");
  });

  test("does not offer a run increase for a completed answer below its limit", () => {
    const { container } = setup({ ...chat, status: "idle" });
    expect(container.textContent).toBe("");
  });

  test("does not present a token increase as recovery from a provider error", () => {
    setup({ ...chat, status: "error", runBudget: { ...chat.runBudget!, totalTokensUsed: 1100, hasUnreportedUsage: true } });
    expect(screen.getByRole("status")).toHaveTextContent("Some token usage was not reported.");
    expect(screen.queryByRole("button")).toBeNull();
  });

  test.each(["error", "idle"] as const)("offers explicit End run for managed interrupted work in %s state", async (status) => {
    const { onEndRun, onResume } = setup({ ...chat, runId: "run-1", status, interrupted: true, runBudget: { ...chat.runBudget!, totalTokensUsed: 100 } });
    expect(screen.getByRole("status")).toHaveTextContent("Interrupted");
    expect(screen.queryByRole("button", { name: "Resume" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Increase limit" })).toBeNull();
    expect(onEndRun).not.toHaveBeenCalled();
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "End run" })));
    expect(onEndRun).toHaveBeenCalledOnce();
    expect(onResume).not.toHaveBeenCalled();
  });

  test("leaves End run available even when managed recovery metadata is missing", () => {
    setup({ ...chat, runId: "run-1", status: "error", runBudget: undefined }, false, { canResume: false });
    expect(screen.getByRole("button", { name: "End run" })).toBeEnabled();
    expect(screen.queryByRole("button", { name: "Resume" })).toBeNull();
    expect(screen.queryByText(/tokens$/)).toBeNull();
  });

  test("preserves token-pause controls when configuration recovery changes status", () => {
    setup({ ...chat, status: "error", requiresAIConfiguration: true, runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } }, false, { canResume: false });
    expect(screen.getByRole("status")).toHaveTextContent("Run paused at its token limit.");
    expect(screen.getByRole("button", { name: "Increase limit" })).toBeEnabled();
    expect(screen.queryByRole("button", { name: "Resume" })).toBeNull();
    expect(screen.getByRole("button", { name: "End run" })).toBeEnabled();
  });

  test("allows an explicit remote recovery attempt without claiming the owner is gone", async () => {
    const { onResume, onEndRun } = setup({ ...chat, runId: "run-1", runCheckpoint: { ...checkpoint, suspension: "interrupted" }, runBudget: { ...chat.runBudget!, totalTokensUsed: 100 } }, false, { remoteBusy: true, canResume: true });
    expect(screen.getByRole("status")).toHaveTextContent("This run may be active in another tab.");
    expect(onResume).not.toHaveBeenCalled();
    expect(onEndRun).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Resume" })).toBeEnabled();
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Resume" })));
    expect(onResume).toHaveBeenCalledOnce();
    expect(onEndRun).not.toHaveBeenCalled();
  });

  test("disables invalid remote continuation but keeps explicit retirement available", () => {
    setup({ ...chat, runId: "run-1", runCheckpoint: { ...checkpoint, suspension: "interrupted", startupComplete: false }, runBudget: { ...chat.runBudget!, totalTokensUsed: 100 } }, false, { remoteBusy: true, canResume: false });
    expect(screen.getByRole("button", { name: "Resume" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "End run" })).toBeEnabled();
  });

  test("does not offer remote Resume without a checkpoint", () => {
    const { container } = setup({ ...chat, runId: "run-1", runBudget: { ...chat.runBudget!, totalTokensUsed: 100 } }, false, { remoteBusy: true, canResume: true });
    expect(container.textContent).toBe("");
  });

  test("offers a budget increase before recovering an exhausted remote run", () => {
    setup({ ...chat, status: "idle", interrupted: true, runId: "run-1", runCheckpoint: { ...checkpoint, suspension: "interrupted" }, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } }, false, { remoteBusy: true, canResume: true });
    expect(screen.getByRole("button", { name: "Increase limit" })).toBeEnabled();
    expect(screen.queryByRole("button", { name: "Resume" })).toBeNull();
    expect(screen.getByRole("button", { name: "End run" })).toBeEnabled();
  });

  test.each([800, 1100])("keeps an active increase apply-only at %i tokens", async (totalTokensUsed) => {
    const { onIncreaseLimit, onResume } = setup({ ...chat, runBudget: { ...chat.runBudget!, totalTokensUsed } });
    const input = await openIncrease();
    expect(screen.queryByRole("button", { name: "Increase and resume" })).toBeNull();
    fireEvent.change(input, { target: { value: "2" } });
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Apply" })));
    expect(onIncreaseLimit).toHaveBeenCalledExactlyOnceWith(2000);
    expect(onResume).not.toHaveBeenCalled();
  });

  test("does not promise automatic continuation for an invalid token checkpoint", async () => {
    const { onIncreaseLimit, onResume } = setup({ ...chat, status: "paused", runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } }, false, { canResume: false });
    const input = await openIncrease();
    expect(screen.queryByRole("button", { name: "Increase and resume" })).toBeNull();
    fireEvent.change(input, { target: { value: "2" } });
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Apply" })));
    expect(onIncreaseLimit).toHaveBeenCalledExactlyOnceWith(2000);
    expect(onResume).not.toHaveBeenCalled();
  });

  test("keeps exhausted remote recovery increases separate from resuming", async () => {
    const { onIncreaseLimit, onResume } = setup({ ...chat, status: "idle", interrupted: true, runId: "run-1", runCheckpoint: { ...checkpoint, suspension: "interrupted" }, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } }, false, { remoteBusy: true, canResume: true });
    const input = await openIncrease();
    expect(screen.queryByRole("button", { name: "Increase and resume" })).toBeNull();
    fireEvent.change(input, { target: { value: "2" } });
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Apply" })));
    expect(onIncreaseLimit).toHaveBeenCalledExactlyOnceWith(2000);
    expect(onResume).not.toHaveBeenCalled();
  });

  test("keeps a thrown increase failure editable", async () => {
    const { onIncreaseLimit } = setup({ ...chat, status: "paused", runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } });
    onIncreaseLimit.mockRejectedValueOnce(new Error("save failed"));
    const input = await openIncrease();
    fireEvent.change(input, { target: { value: "2" } });
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Increase and resume" })));
    expect(screen.getByRole("alert")).toHaveTextContent("Couldn't update this run. Try again.");
    expect(screen.getByRole("spinbutton")).toHaveValue(2);
    expect(screen.getByRole("button", { name: "Increase and resume" })).toBeEnabled();
    expect(onIncreaseLimit).toHaveBeenCalledExactlyOnceWith(2000);
  });

  test("retains the editor through an optimistic increase and failed save", async () => {
    const exhaustedChat = { ...chat, status: "paused" as const, runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, totalTokensUsed: 1100 } };
    const { rerender, onIncreaseLimit, onResume, onEndRun } = setup(exhaustedChat);
    let finish!: (value: boolean) => void;
    onIncreaseLimit.mockImplementationOnce(() => new Promise<boolean>((resolve) => { finish = resolve; }));
    const input = await openIncrease();
    fireEvent.change(input, { target: { value: "3" } });
    fireEvent.click(screen.getByRole("button", { name: "Increase and resume" }));
    rerender(<AgentTokenBudgetStatus chat={{ ...exhaustedChat, runBudget: { ...exhaustedChat.runBudget, limit: 3000 } }} onIncreaseLimit={onIncreaseLimit} onResume={onResume} onEndRun={onEndRun} />);
    expect(screen.getByRole("spinbutton")).toHaveValue(3);
    await act(async () => finish(false));
    expect(screen.getByText("Couldn't update this run. Try again.")).toHaveAttribute("role", "alert");
    expect(screen.getByRole("spinbutton")).toBeEnabled();
    fireEvent.change(screen.getByRole("spinbutton"), { target: { value: "4" } });
    expect(screen.getByRole("button", { name: "Increase and resume" })).toBeEnabled();
    expect(onResume).not.toHaveBeenCalled();
  });

  test("keeps remote recovery disabled during a local operation", () => {
    setup({ ...chat, runId: "run-1", runCheckpoint: { ...checkpoint, suspension: "interrupted" }, runBudget: { ...chat.runBudget!, totalTokensUsed: 100 } }, true, { remoteBusy: true, canResume: true });
    expect(screen.getByRole("button", { name: "Resume" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "End run" })).toBeDisabled();
  });

  test("uses a lower bound for partial accounting without a precise percentage", () => {
    setup({ ...chat, runBudget: { ...chat.runBudget!, hasUnreportedUsage: true } });
    expect(screen.getByRole("region")).toHaveTextContent("At least 800 / 1,000 tokens");
    expect(screen.getByRole("region")).toHaveTextContent("Some token usage was not reported.");
    expect(screen.getByRole("region").textContent).not.toContain("%");
  });

  test("disables all commands for a non-owner", () => {
    setup({ ...chat, status: "paused", runCheckpoint: checkpoint, runBudget: { ...chat.runBudget!, limit: 2000 } }, true);
    for (const button of screen.getAllByRole("button")) expect(button).toBeDisabled();
  });
});
