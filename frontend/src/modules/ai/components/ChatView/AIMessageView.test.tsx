import { render } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import type { Conversation, Message } from "../../types";
import { AIMessageView } from "./AIMessageView";

vi.mock("./Markdown/CodeBlock", () => ({
  CodeBlock: () => <div data-testid="code-block" />,
}));

const longContent =
  "https://example.com/" + "unbroken-path-segment".repeat(20);

function createMessage(
  author: Message["author"],
  status: Message["status"],
  overrides: Partial<Message> = {}
): Message {
  const conversation: Conversation = {
    id: "conversation-1",
    created_ts: 1,
    name: "Test",
    instance: "instances/test",
    database: "instances/test/databases/test",
    messageList: [],
  };
  const message: Message = {
    id: "message-1",
    created_ts: 1,
    author,
    content: longContent,
    status,
    error: "",
    conversation,
    ...overrides,
  };
  conversation.messageList.push(message);
  return message;
}

describe("AIMessageView", () => {
  test("keeps long AI content inside the message bubble", () => {
    const message = createMessage("AI", "DONE");
    const { container } = render(<AIMessageView message={message} />);

    const bubble = container.firstElementChild;
    const markdown = container.querySelector(".markdown");
    expect(bubble?.className).toContain("min-w-0");
    expect(bubble?.className).toContain("max-w-full");
    expect(markdown?.className).toContain("wrap-anywhere");
    expect(markdown?.className).toContain("min-w-0");
    expect(markdown?.className).toContain("text-sm");
    expect(markdown?.className).toContain("leading-5");
    expect(markdown?.textContent).toContain(longContent);
  });

  test("wraps long provider errors inside the failed message bubble", () => {
    const message = createMessage("AI", "FAILED", { error: longContent });
    const { container } = render(<AIMessageView message={message} />);

    const bubble = container.firstElementChild;
    const error = container.querySelector("span");
    expect(bubble?.className).toContain("min-w-0");
    expect(bubble?.className).toContain("max-w-[80%]");
    expect(error?.className).toContain("min-w-0");
    expect(error?.className).toContain("wrap-anywhere");
    expect(error?.textContent).toBe(longContent);
  });
});
