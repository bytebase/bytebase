import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import enUS from "@/locales/en-US.json";
import type { AgentChat } from "../logic/types";
import { AgentTokenUsage } from "./AgentTokenUsage";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, values?: Record<string, unknown>) => {
      const translationKey = key === "agent.token-usage.model-calls" ? `${key}_${values?.count === 1 ? "one" : "other"}` : key;
      const value = translationKey.split(".").reduce<unknown>((result, part) =>
        typeof result === "object" && result !== null ? (result as Record<string, unknown>)[part] : undefined, enUS);
      return typeof value === "string" ? value.replace(/{{(\w+)}}/g, (_, name) => String(values?.[name] ?? "")) : key;
    },
  }),
}));

const chat: AgentChat = {
  id: "chat-1", title: "Ping", createdTs: 0, updatedTs: 0, status: "idle", archived: false,
  totalTokensUsed: 150,
  lastRunUsage: {
    runId: "run-1",
    calls: [
      { purpose: "agent", toolNames: ["done"], totalTokens: 120, inputTokens: 100, outputTokens: 20, cachedInputTokens: 0 },
      { purpose: "follow-up", toolNames: [], totalTokens: 30, inputTokens: 20, outputTokens: 10, cachedInputTokens: 0 },
    ],
  },
};

let root: Root | undefined;
afterEach(() => {
  act(() => root?.unmount());
  root = undefined;
  document.body.innerHTML = "";
});

async function renderUsage(value: AgentChat) {
  const container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  await act(async () => root!.render(<AgentTokenUsage chat={value} />));
  const trigger = container.querySelector("button")!;
  await act(async () => trigger.click());
  return { trigger, panel: document.querySelector("[data-agent-token-usage]")! };
}

describe("AgentTokenUsage", () => {
  test("opens the latest answer breakdown in the agent layer", async () => {
    const { trigger, panel } = await renderUsage(chat);
    expect(trigger.textContent).toContain("Chat tokens: 150");
    expect(trigger.getAttribute("aria-expanded")).toBe("true");
    expect(panel.closest("#bb-react-layer-agent")).not.toBeNull();
    expect(panel.textContent).toContain("Latest answer");
    expect(panel.textContent).toContain("2 model calls");
    expect(panel.textContent).toContain("Follow-up questions");
    const totals = panel.querySelector("tfoot")!.textContent;
    expect(totals).toContain("120");
    expect(totals).toContain("30");
    expect(totals).toContain("150");
    expect(panel.querySelector("[data-agent-cached-input]")?.textContent).toContain("0");
    await act(async () => panel.querySelector<HTMLButtonElement>('[aria-label="Close"]')!.click());
    expect(trigger.getAttribute("aria-expanded")).toBe("false");
  });

  test("does not fabricate a breakdown for a legacy chat", async () => {
    const { panel } = await renderUsage({ ...chat, lastRunUsage: undefined });
    expect(panel.textContent).toContain("150");
    expect(panel.textContent).toContain("Not recorded");
    expect(panel.querySelector("table")).toBeNull();
  });

  test("shows partial reported totals and distinguishes unknown metrics from zero", async () => {
    const { trigger, panel } = await renderUsage({
      ...chat, hasUnreportedUsage: true,
      lastRunUsage: { runId: "run-1", calls: [
        { purpose: "agent", toolNames: [], totalTokens: 0, inputTokens: 0, outputTokens: 0 },
        { purpose: "follow-up", toolNames: [] },
      ] },
    });
    expect(trigger.textContent).toContain("At least 150");
    expect(panel.textContent).toContain("Not reported");
    expect(panel.querySelector("tbody tr")!.textContent).toContain("0");
    expect(panel.querySelector("tfoot")!.textContent).toContain("At least 0");
    expect(panel.querySelector("[data-agent-cached-input]")?.textContent).toContain("Not reported");
  });
});
