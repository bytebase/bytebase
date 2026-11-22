import { expect, test, type Page } from "@playwright/test";
import { loadTestEnv } from "../framework/env";

const chatId = "agent-chat-menu-e2e";
const chatTitle = "Chat menu visibility";

async function openChatMenuFixture(page: Page) {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const { workspace } = await env.api.getActuatorInfo();
  const workspaceId = workspace.split("/").at(-1)!;
  await page.addInitScript(
    ({ key, workspace, project, chatId, chatTitle }) => {
      const now = Date.now();
      localStorage.setItem(
        key,
        JSON.stringify({
          currentChatId: chatId,
          chats: [{
            id: chatId,
            title: chatTitle,
            context: project,
            workspaceName: workspace,
            createdTs: now,
            updatedTs: now,
            status: "idle",
            totalTokensUsed: 0,
            archived: false,
          }],
          messagesByChatId: {
            [chatId]: [{
              id: `${chatId}-answer`,
              chatId,
              role: "assistant",
              content: "Saved discussion for menu visibility checks.",
              createdTs: now,
            }],
          },
          pendingAskByChatId: {},
          draftsByChatId: {},
          queuedUserMessagesByChatId: {},
        })
      );
    },
    {
      key: `bb-agent-state-v2.${workspaceId.length}:${workspaceId}.${env.adminEmail}`,
      workspace,
      project: env.project,
      chatId,
      chatTitle,
    }
  );
  await page.goto(`${env.baseURL}/landing`);
  await page.getByRole("button", { name: "Agent", exact: true }).click();
  await expect(page.locator("[data-agent-window]")).toBeVisible();
  const title = page.getByRole("button", { name: chatTitle, exact: true });
  if ((await title.count()) === 0) {
    await page.getByRole("button", { name: "Chats", exact: true }).click();
  }
  await expect(title).toBeVisible();
  return {
    title,
    more: page.getByRole("button", { name: `More: ${chatTitle}`, exact: true }),
    menu: page.locator("[data-agent-dropdown-menu-content]"),
  };
}

test("More hides after pointer dismissal outside its row without losing restored focus", async ({
  page,
}) => {
  const { title, more, menu } = await openChatMenuFixture(page);
  expect(await page.evaluate(() => matchMedia("(pointer: fine)").matches)).toBe(true);
  await title.hover();
  await expect(more).toHaveCSS("opacity", "1");
  await expect(more).toHaveCSS("pointer-events", "auto");
  await more.click();
  await expect(menu).toBeVisible();
  await page.mouse.move(0, 0);
  await expect(more).toHaveCSS("opacity", "1");
  await expect(more).toHaveCSS("pointer-events", "auto");

  await page.locator("[data-agent-window-header]").click({
    position: { x: 10, y: 10 },
  });
  await expect(menu).toBeHidden();
  await expect(more).toBeFocused();
  await expect(more).toHaveCSS("opacity", "0");
  await expect(more).toHaveCSS("pointer-events", "none");
});

test("keyboard focus reveals More and Escape preserves its visible focus target", async ({
  page,
}) => {
  const { title, more, menu } = await openChatMenuFixture(page);
  await page.mouse.move(0, 0);
  await title.focus();
  await page.keyboard.press("Shift+Tab");
  await page.keyboard.press("Tab");
  await expect(title).toBeFocused();
  await expect(more).toHaveCSS("opacity", "1");
  await expect(more).toHaveCSS("pointer-events", "auto");

  await page.keyboard.press("Tab");
  await expect(more).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(menu).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(menu).toBeHidden();
  await expect(more).toBeFocused();
  await expect(more).toHaveCSS("opacity", "1");
  await expect(more).toHaveCSS("pointer-events", "auto");
});

test.describe("coarse pointer", () => {
  test.use({
    hasTouch: true,
    isMobile: true,
    viewport: { width: 390, height: 844 },
  });

  test("More is visible without hover on touch screens", async ({ page }) => {
    const { more } = await openChatMenuFixture(page);
    expect(
      await page.evaluate(() => matchMedia("(pointer: coarse)").matches)
    ).toBe(true);
    await expect(more).toHaveCSS("opacity", "1");
    await expect(more).toHaveCSS("pointer-events", "auto");
  });
});
