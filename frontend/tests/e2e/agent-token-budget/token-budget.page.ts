import { expect, type Page, type TestInfo } from "@playwright/test";
import * as path from "node:path";

export class TokenBudgetPage {
  initialDefault?: string;
  constructor(
    readonly page: Page,
    readonly baseURL: string,
    readonly testInfo: TestInfo
  ) {}

  get window() {
    return this.page.locator("[data-agent-window]");
  }
  get region() {
    return this.page.getByRole("region", {
      name: "Run token budget",
      exact: true,
    });
  }
  get input() {
    return this.window.locator("textarea").last();
  }
  get usage() {
    return this.page.locator("[data-agent-token-usage]");
  }

  async open() {
    await this.page.goto(`${this.baseURL}/landing`);
    await this.reopen();
  }

  async reopen() {
    await this.page.getByRole("button", { name: "Agent", exact: true }).click();
    await expect(this.window).toBeVisible();
    const chats = this.window.getByRole("button", {
      name: "Chats",
      exact: true,
    });
    if (
      (await this.window.locator("textarea").count()) === 0 &&
      (await chats.count()) > 0
    ) {
      await chats.first().click();
      await this.window
        .getByRole("button", { name: /Budget QA TB-/ })
        .first()
        .click();
    }
  }

  async setDefaultLimit(value: string) {
    await this.window
      .getByRole("button", { name: "Agent preferences", exact: true })
      .click();
    const limit = this.window.getByRole("spinbutton", {
      name: "Token limit per run",
      exact: true,
    });
    this.initialDefault ??= await limit.inputValue();
    await limit.fill(value);
    await this.window
      .getByRole("button", { name: "Back to chat", exact: true })
      .click();
  }

  async send(text: string) {
    await this.input.fill(text);
    await this.window
      .getByRole("button", { name: "Send", exact: true })
      .click();
  }

  async paused() {
    await expect(this.region).toContainText("Run paused at its token limit.");
    await expect(
      this.window.getByRole("button", { name: "Stop", exact: true })
    ).toBeHidden();
  }

  async increase(value: string, { resume = false } = {}) {
    await this.region
      .getByRole("button", { name: "Increase limit", exact: true })
      .click();
    const dialog = this.page.getByRole("dialog", {
      name: "Increase limit for this run",
      exact: true,
    });
    await expect(dialog.getByText("K tokens", { exact: true })).toBeVisible();
    await dialog
      .getByRole("spinbutton", { name: "This run's token limit", exact: true })
      .fill(value);
    await dialog
      .getByRole("button", {
        name: resume ? "Increase and resume" : "Apply",
        exact: true,
      })
      .click();
  }

  async openUsage() {
    await this.window.getByRole("button", { name: /^Chat tokens:/ }).click();
    await expect(this.usage).toBeVisible();
  }

  async state() {
    return this.page.evaluate(() => {
      const key = Object.keys(localStorage).find((key) =>
        key.startsWith("bb-agent-state-v3.")
      );
      if (!key) return null;
      const state = JSON.parse(localStorage.getItem(key)!);
      const chat = state.chats.find(
        (chat: { id: string }) => chat.id === state.currentChatId
      );
      const messages = state.messagesByChatId[state.currentChatId] ?? [];
      const readToolCallIds: string[] = messages
        .flatMap(
          (message: { toolCalls?: { id: string; name: string }[] }) =>
            message.toolCalls ?? []
        )
        .filter((call: { name: string }) => call.name === "get_page_state")
        .map((call: { id: string }) => call.id);
      return {
        runId: chat?.runId,
        budget: chat?.runBudget,
        status: chat?.status,
        attempts: chat?.modelAttempts?.map(
          (attempt: { attemptId: string; outcome: string }) => ({
            attemptId: attempt.attemptId,
            outcome: attempt.outcome,
          })
        ),
        readToolCallIds,
        readToolResultIds: messages
          .filter(
            (message: { role: string; toolCallId?: string }) =>
              message.role === "tool" &&
              readToolCallIds.includes(message.toolCallId!)
          )
          .map((message: { toolCallId: string }) => message.toolCallId),
        userRows: messages.filter(
          (message: { role: string }) => message.role === "user"
        ).length,
        toolRows: messages.filter(
          (message: { role: string }) => message.role === "tool"
        ).length,
        preference: JSON.parse(
          localStorage.getItem(
            Object.keys(localStorage).find((key) =>
              key.startsWith("bb-agent-preferences-v1.")
            )!
          ) ?? "null"
        ),
      };
    });
  }

  async screenshot(name: string) {
    const directory = process.env.BYTEBASE_TOKEN_BUDGET_SCREENSHOTS;
    const file = directory
      ? path.join(directory, `${name}-deterministic-provider-mock.png`)
      : this.testInfo.outputPath(`${name}-deterministic-provider-mock.png`);
    await this.page.screenshot({
      path: file,
      mask: [
        this.page.getByRole("banner"),
        this.page.getByRole("navigation").filter({ hasNot: this.window }),
        this.window.locator(
          "[data-agent-window-header], [data-agent-chat-group-header], [data-agent-context], [data-agent-message-timestamp]"
        ),
      ],
      maskColor: "#e5e7eb",
    });
    await this.testInfo.attach(name, { path: file, contentType: "image/png" });
  }
}
