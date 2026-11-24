import { writeFileSync } from "node:fs";
import { expect, test, type Locator, type Page } from "@playwright/test";
import { loadTestEnv } from "../framework/env";

test.use({ hasTouch: true, isMobile: true });

const activeId = "mobile-safety-active";
const archivedId = "mobile-safety-archived";
const activeTitle = "Active mobile discussion";
const archivedTitle = "Archived mobile discussion";

async function openChatList(page: Page, archived: boolean, recoveryEvidence: boolean) {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const { workspace } = await env.api.getActuatorInfo();
  const workspaceId = workspace.split("/").at(-1)!;
  await page.addInitScript(
    ({ key, project, workspace, recoveryEvidence, activeId, archivedId, activeTitle, archivedTitle }) => {
      const now = Date.now();
      localStorage.setItem(key, JSON.stringify({
        currentChatId: activeId,
        chats: [
          { id: activeId, title: activeTitle, context: project, workspaceName: workspace, createdTs: now, updatedTs: now, status: "idle", totalTokensUsed: 0, archived: false },
          { id: archivedId, title: archivedTitle, context: project, workspaceName: workspace, createdTs: now, updatedTs: now, status: "idle", totalTokensUsed: 0, archived: true, apiAttempts: recoveryEvidence ? [null] : undefined },
        ],
        messagesByChatId: {
          [activeId]: [{ id: `${activeId}-answer`, chatId: activeId, role: "assistant", content: "Mobile chat action review.", createdTs: now }],
          [archivedId]: [{ id: `${archivedId}-answer`, chatId: archivedId, role: "assistant", content: "Archived chat action review.", createdTs: now }],
        },
        pendingAskByChatId: {},
        draftsByChatId: {},
        queuedUserMessagesByChatId: {},
      }));
    },
    { key: `bb-agent-state-v3.${workspaceId.length}:${workspaceId}.${env.adminEmail}`, project: env.project, workspace, recoveryEvidence, activeId, archivedId, activeTitle, archivedTitle }
  );
  await page.goto(`${env.baseURL}/landing`);
  await page.getByRole("button", { name: "Agent", exact: true }).click();
  const agent = page.locator("[data-agent-window]");
  await expect(agent).toBeVisible();
  if (!(await agent.getByRole("button", { name: activeTitle, exact: true }).isVisible())) {
    await agent.getByRole("button", { name: "Chats", exact: true }).click();
  }
  if (archived) await agent.getByRole("button", { name: "View archived chats", exact: true }).click();
  const title = archived ? archivedTitle : activeTitle;
  const row = agent.locator(`[data-agent-chat-row="${archived ? archivedId : activeId}"]`);
  await expect(row.getByRole("button", { name: title, exact: true })).toBeVisible();
  await row.getByRole("button", { name: `More: ${title}`, exact: true }).click();
  return agent;
}

async function expectBottomActions(surface: Locator, width: number) {
  const actions = surface.locator("[data-agent-action-group]");
  await expect(actions).toBeVisible();
  const bounds = (await surface.boundingBox())!;
  expect(bounds.x).toBeGreaterThanOrEqual(0);
  expect(bounds.x + bounds.width).toBeLessThanOrEqual(width);
  const row = (await actions.boundingBox())!;
  const buttons = actions.getByRole("button");
  for (const button of await buttons.all()) {
    const box = (await button.boundingBox())!;
    expect(box.x).toBeGreaterThanOrEqual(row.x - 1);
    expect(box.x + box.width).toBeLessThanOrEqual(row.x + row.width + 1);
    expect(await button.evaluate((node) => node.scrollWidth - node.clientWidth)).toBeLessThanOrEqual(1);
  }
  const last = (await buttons.last().boundingBox())!;
  expect(Math.abs(last.x + last.width - row.x - row.width)).toBeLessThanOrEqual(1);
  await expect(actions).toHaveCSS("flex-wrap", "wrap");
}

for (const width of [320, 390]) {
  for (const scenario of ["active-menu", "archive-confirmation", "archived-menu", "delete-confirmation", "delete-recovery-warning"] as const) {
    test(`mobile chat ${scenario} at ${width}px`, async ({ page }, testInfo) => {
      await page.setViewportSize({ width, height: 844 });
      const archived = scenario.startsWith("delete") || scenario === "archived-menu";
      const agent = await openChatList(page, archived, scenario === "delete-recovery-warning");
      let surface = page.locator("[data-agent-dropdown-menu-content]");
      await expect(surface).toBeVisible();
      if (scenario === "archive-confirmation") {
        await surface.locator("[data-agent-archive-chat]").click();
        surface = page.locator("[data-agent-archive-chat-confirmation]");
      } else if (scenario.startsWith("delete")) {
        await surface.locator("[data-agent-delete-chat]").click();
        surface = page.getByRole("dialog", { name: "Confirm", exact: true });
      }
      await expect(surface).toBeVisible();
      if (scenario.includes("confirmation") || scenario === "delete-recovery-warning") {
        await expectBottomActions(surface, width);
      }
      if (scenario === "delete-recovery-warning") {
        await expect(surface).toContainText("Deleting this history removes local recovery evidence.");
      }
      const panelFile = `mobile-chat-${scenario}-surface-${width}.png`;
      const windowFile = `mobile-chat-${scenario}-window-${width}.png`;
      const pageFile = `mobile-chat-${scenario}-page-${width}.png`;
      await surface.screenshot({ path: testInfo.outputPath(panelFile) });
      await agent.screenshot({ path: testInfo.outputPath(windowFile) });
      await page.screenshot({ path: testInfo.outputPath(pageFile) });
      writeFileSync(testInfo.outputPath("screenshots.json"), JSON.stringify({
        title: `Chat ${scenario.replaceAll("-", " ")}`,
        width,
        screenshots: [
          { file: panelFile, label: "Action surface" },
          { file: windowFile, label: "Agent window" },
          { file: pageFile, label: "Mobile viewport" },
        ],
      }));
      if (scenario.includes("confirmation") || scenario === "delete-recovery-warning") {
        await surface.getByRole("button", { name: "Cancel", exact: true }).click();
        await expect(surface).toBeHidden();
      }
    });
  }
}
