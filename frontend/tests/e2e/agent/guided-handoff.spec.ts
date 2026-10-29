import { expect, test } from "@playwright/test";
import { loadTestEnv } from "../framework/env";

test("a saved Agent handoff opens the matching filtered Issue queue only after click", async ({
  page,
}) => {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const { workspace } = await env.api.getActuatorInfo();
  const projectId = env.project.split("/").at(-1)!;
  const chatId = "agent-guided-handoff-e2e";
  const now = Date.now();

  await page.addInitScript(
    ({ key, chatId, now, workspace, project }) => {
      if (sessionStorage.getItem("agent-guided-handoff-seeded")) return;
      localStorage.setItem(
        key,
        JSON.stringify({
          currentChatId: chatId,
          chats: [
            {
              id: chatId,
              title: "Review changes",
              context: project,
              workspaceName: workspace,
              createdTs: now - 1,
              updatedTs: now,
              status: "idle",
              totalTokensUsed: 0,
              archived: false,
            },
          ],
          messagesByChatId: {
            [chatId]: [
              {
                id: "handoff-user",
                chatId,
                role: "user",
                content: "What needs review?",
                createdTs: now - 1,
              },
              {
                id: "handoff-answer",
                chatId,
                role: "assistant",
                content: "Review the current change queue.",
                createdTs: now,
                metadata: {
                  handoff: {
                    kind: "review-database-changes",
                    project,
                  },
                },
              },
            ],
          },
          pendingAskByChatId: {},
          draftsByChatId: {},
          queuedUserMessage: null,
        })
      );
      sessionStorage.setItem("agent-guided-handoff-seeded", "1");
    },
    {
      key: `bb-agent-state-v2.${workspace.split("/").at(-1)}`,
      chatId,
      now,
      workspace,
      project: env.project,
    }
  );
  await page.goto(`${env.baseURL}/landing`);
  await page.getByRole("button", { name: "Agent", exact: true }).click();

  const handoff = page.locator("[data-agent-handoff]");
  await expect(handoff).toHaveText(/Review database changes/);
  await expect(page).toHaveURL(/\/landing$/);
  await handoff.click();

  await expect(page).toHaveURL(new RegExp(`/projects/${projectId}/issues\\?`));
  await expect(page.locator('[data-search-scope-id="status"]')).toContainText("Open");
  await expect(page.locator('[data-search-scope-id="issue-type"]')).toContainText("Database Change");
});
