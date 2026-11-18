import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import type { ComponentProps } from "react";
import { describe, expect, test, vi } from "vitest";
import enUS from "@/locales/en-US.json";
import type { QueuedUserMessage } from "../store/agent";
import { AgentFollowUpQueueList } from "./AgentFollowUpQueueList";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, values?: Record<string, unknown>) => {
      const translation = key.split(".").reduce<unknown>((value, part) => {
        return typeof value === "object" && value !== null
          ? (value as Record<string, unknown>)[part]
          : undefined;
      }, enUS);
      return typeof translation === "string"
        ? translation.replace(/{{(\w+)}}/g, (_, part) => String(values?.[part] ?? ""))
        : key;
    },
  }),
}));

const messages: QueuedUserMessage[] = [
  { id: "second", chatId: "chat", text: "Compare the plans", page: { path: "/", title: "Home" } },
  { id: "third", chatId: "chat", text: "Summarize the rollout", page: { path: "/", title: "Home" } },
];

function renderQueue(overrides: Partial<ComponentProps<typeof AgentFollowUpQueueList>> = {}) {
  const onRemove = vi.fn();
  const onResume = vi.fn();
  const result = render(
    <AgentFollowUpQueueList
      queue={{ messages, paused: false }}
      onRemove={onRemove}
      onResume={onResume}
      {...overrides}
    />
  );
  return { ...result, onRemove, onResume };
}

describe("AgentFollowUpQueueList", () => {
  test.each([undefined, { messages: [], paused: true }])("hides the empty queue", (queue) => {
    expect(renderQueue({ queue }).container).toBeEmptyDOMElement();
  });

  test("shows pending prompts in order and removes the selected item by ID", () => {
    const { onRemove } = renderQueue();
    expect(screen.getByText("2 queued")).toBeVisible();
    const rows = screen.getAllByRole("listitem");
    expect(rows[0]).toHaveTextContent(messages[0].text);
    expect(rows[1]).toHaveTextContent(messages[1].text);

    fireEvent.click(within(rows[1]).getByRole("button", { name: "Remove queued message 2" }));
    expect(onRemove).toHaveBeenCalledExactlyOnceWith("third");
    expect(screen.queryByRole("button", { name: "Resume queue" })).not.toBeInTheDocument();
  });

  test("keeps the queue borderless with a secondary count above the prompts", () => {
    const { container } = renderQueue();
    expect(container.firstElementChild).not.toHaveClass("border-t");
    expect(screen.getByText("2 queued").closest(".text-control-light")).not.toBeNull();
    expect(screen.getByRole("status")).toHaveTextContent("2 queued");
  });

  test("keeps prompt text icon-free and the remove action last in each row", () => {
    renderQueue();
    for (const [index, row] of screen.getAllByRole("listitem").entries()) {
      const remove = within(row).getByRole("button", { name: `Remove queued message ${index + 1}` });
      expect(row.lastElementChild).toContainElement(remove);
      expect([...row.querySelectorAll("svg")].every(icon => remove.contains(icon))).toBe(true);
      expect(within(row).getByText(messages[index].text)).toHaveAttribute("tabindex", "0");
    }
  });

  test("bounds a long queue without hiding its later removal controls", () => {
    const longQueue = Array.from({ length: 30 }, (_, index) => ({
      ...messages[0],
      id: `queued-${index}`,
      text: `Pending question ${index + 1}`,
    }));
    const { onRemove } = renderQueue({ queue: { messages: longQueue, paused: false } });
    const list = screen.getByRole("list", { name: "30 queued" });
    expect(list).toHaveClass("overflow-y-auto");
    expect([...list.classList].some(name => name.startsWith("max-h-"))).toBe(true);
    expect(within(list).getAllByRole("listitem")).toHaveLength(30);
    fireEvent.click(within(list).getByRole("button", { name: "Remove queued message 30" }));
    expect(onRemove).toHaveBeenCalledExactlyOnceWith("queued-29");
  });

  test("shows and resumes a paused queue", () => {
    const { onResume } = renderQueue({ queue: { messages, paused: true } });
    expect(screen.getByText("Paused")).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Resume queue" }));
    expect(onResume).toHaveBeenCalledTimes(1);
  });

  test.each([true, false])("disables queue mutations when not allowed (read only: %s)", (disabled) => {
    const { onRemove, onResume } = renderQueue({
      disabled,
      resumeDisabled: !disabled,
      queue: { messages, paused: true },
    });
    const resume = screen.getByRole("button", { name: "Resume queue" });
    const remove = screen.getByRole("button", { name: "Remove queued message 1" });
    expect(resume).toBeDisabled();
    expect((remove as HTMLButtonElement).disabled).toBe(disabled);
    fireEvent.click(resume);
    fireEvent.click(remove);
    expect(onResume).not.toHaveBeenCalled();
    expect(onRemove).toHaveBeenCalledTimes(disabled ? 0 : 1);
  });

  test("makes the full text of a long prompt available in the agent tooltip", async () => {
    const prompt = `${"very-long-prompt ".repeat(16)}\nSecond line`;
    renderQueue({ queue: { messages: [{ ...messages[0], text: prompt }], paused: false } });
    const normalized = prompt.replace(/\s+/g, " ");
    const text = within(screen.getByRole("listitem")).getByText(normalized);
    fireEvent.focusIn(text);

    await waitFor(() => {
      expect(document.querySelector("[data-agent-tooltip-content]")).toHaveTextContent(normalized);
    });
  });
});
