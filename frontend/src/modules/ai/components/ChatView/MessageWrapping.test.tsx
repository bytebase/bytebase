import { render } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import type { Conversation, Message } from "../../types";
import { AIMessageView } from "./AIMessageView";
import { UserMessageView } from "./UserMessageView";

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
    prompt: longContent,
    status,
    error: "",
    conversation,
    ...overrides,
  };
  conversation.messageList.push(message);
  return message;
}

describe("AI chat message wrapping", () => {
  test.each([
    ["AI", AIMessageView],
    ["USER", UserMessageView],
  ] as const)("keeps long %s content inside the message bubble", (_, View) => {
    const message = createMessage(
      View === AIMessageView ? "AI" : "USER",
      "DONE"
    );
    const { container } = render(<View message={message} />);

    const bubble = container.firstElementChild;
    const markdown = container.querySelector(".markdown");
    expect(bubble?.className).toContain("min-w-0");
    expect(bubble?.className).toContain(
      View === AIMessageView ? "max-w-full" : "max-w-[60%]"
    );
    expect(markdown?.className).toContain("wrap-anywhere");
    expect(markdown?.className).toContain("min-w-0");
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

  test("constrains inline code, images, and tables to the message width", () => {
    const content = [
      `\`${longContent}\``,
      `![diagram](${longContent})`,
      [`| ${longContent} |`, "| --- |", `| ${longContent} |`].join("\n"),
    ].join("\n\n");
    const message = createMessage("USER", "DONE", { content });
    const { container } = render(<UserMessageView message={message} />);

    const inlineCode = container.querySelector("code");
    expect(inlineCode?.className).toContain("max-w-full");
    expect(inlineCode?.className).toContain("wrap-anywhere");
    expect(inlineCode?.className).toContain("whitespace-normal");
    expect(container.querySelector("img")?.className).toContain("max-w-full");
    expect(container.querySelector("table")?.className).toContain("w-full");
    expect(container.querySelector("table")?.className).toContain(
      "table-fixed"
    );
  });
});
