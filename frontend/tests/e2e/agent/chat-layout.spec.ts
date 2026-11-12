import { expect, test, type Page } from "@playwright/test";
import { loadTestEnv } from "../framework/env";

const chatId = "agent-chat-layout-e2e";
const userContent = JSON.stringify({
  database: `instances/test-instance/databases/${"orders_".repeat(100)}`,
  fingerprint: "x".repeat(1200),
}, null, 2);
const assistantContent = `Review result: ${"longtoken".repeat(200)}.`;
const sql = `SELECT '${"value_".repeat(400)}' AS value;`;

async function openLayoutFixture(page: Page) {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const { workspace } = await env.api.getActuatorInfo();
  const workspaceId = workspace.split("/").at(-1)!;
  await page.addInitScript(
    ({ key, workspace, project, chatId, messages }) => {
      const now = Date.now();
      localStorage.setItem(key, JSON.stringify({
        currentChatId: chatId,
        chats: [{
          id: chatId,
          title: "Chat layout regression",
          context: project,
          workspaceName: workspace,
          createdTs: now,
          updatedTs: now,
          status: "idle",
          totalTokensUsed: 0,
          archived: false,
        }],
        messagesByChatId: {
          [chatId]: messages.map((message, index) => ({
            ...message,
            id: `${chatId}-${index}`,
            chatId,
            createdTs: now + index,
          })),
        },
        pendingAskByChatId: {},
        draftsByChatId: {},
        queuedUserMessagesByChatId: {},
      }));
    },
    {
      key: `bb-agent-state-v2.${workspaceId.length}:${workspaceId}.${env.adminEmail}`,
      workspace,
      project: env.project,
      chatId,
      messages: [
        { role: "user", content: userContent },
        { role: "assistant", content: assistantContent },
        { role: "assistant", content: `SQL statement:\n\n\`\`\`sql\n${sql}\n\`\`\`` },
      ],
    }
  );
  await page.goto(`${env.baseURL}/landing`);
  await page.getByRole("button", { name: "Agent", exact: true }).click();
  await expect(page.locator("[data-agent-window]")).toBeVisible();
}

for (const width of [320, 390, 1440]) {
  test(`long Agent messages stay inside the conversation at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await openLayoutFixture(page);
    const pane = page.locator("[data-agent-chat-scroll-area]");
    const userBubble = page.locator("[data-agent-user-message]");
    const answers = page.locator("[data-agent-answer]");
    const pre = answers.nth(1).locator("pre");
    await expect(userBubble).toBeVisible();
    await expect(answers).toHaveCount(2);
    await expect(pre).toBeVisible();

    expect(await userBubble.textContent()).toBe(userContent);
    expect(await answers.first().textContent()).toBe(assistantContent);
    expect(await pre.textContent()).toBe(`${sql}\n`);
    await expect(userBubble).toHaveCSS("white-space", "pre-wrap");
    await expect(pre).toHaveCSS("white-space", "pre");
    await expect(pre).toHaveCSS("overflow-x", "auto");
    await expect.poll(() => pre.evaluate((element) =>
      element.scrollWidth - element.clientWidth
    )).toBeGreaterThan(0);
    await expect.poll(() => pane.evaluate((element) =>
      element.scrollWidth - element.clientWidth
    )).toBeLessThanOrEqual(1);

    const paneBounds = (await pane.boundingBox())!;
    expect(paneBounds).not.toBeNull();
    for (const bubble of [userBubble, answers.first(), answers.nth(1)]) {
      const bounds = (await bubble.boundingBox())!;
      expect(bounds).not.toBeNull();
      expect(bounds.x).toBeGreaterThanOrEqual(paneBounds.x - 1);
      expect(bounds.x + bounds.width).toBeLessThanOrEqual(
        paneBounds.x + paneBounds.width + 1
      );
    }
  });
}
