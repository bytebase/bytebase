import { render } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import type { Conversation, Message } from "../../types";
import { ChatView } from "./ChatView";

vi.mock("./Markdown/CodeBlock", () => ({
  CodeBlock: () => <div data-testid="code-block" />,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("../context", () => ({
  useAIContext: () => ({ events: { emit: vi.fn() } }),
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

describe("ChatView", () => {
  test("keeps the first message separated from the panel header", () => {
    const message = createMessage("AI", "LOADING");
    const scrollTo = vi.fn();
    HTMLElement.prototype.scrollTo = scrollTo;

    const { container } = render(
      <ChatView conversation={message.conversation} />
    );

    expect(container.querySelector(".message")?.parentElement).toHaveClass(
      "pt-2"
    );
  });
});
