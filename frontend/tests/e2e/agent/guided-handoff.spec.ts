import { expect, test, type Page } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";

type AgentMessageMetadata = Record<string, unknown>;

async function seedChat(
  page: Page,
  env: TestEnv & { api: ReturnType<typeof loadTestEnv>["api"] },
  options: {
    chatId: string;
    title: string;
    context: string;
    prompt: string;
    answer: string;
    metadata: AgentMessageMetadata;
  }
) {
  await env.api.login(env.adminEmail, env.adminPassword);
  const { workspace } = await env.api.getActuatorInfo();
  const workspaceId = workspace.split("/").at(-1)!;
  const now = Date.now();
  await page.addInitScript(
    ({ key, options, now, workspace }) => {
      const marker = `agent-guided-handoff-seeded.${options.chatId}`;
      if (sessionStorage.getItem(marker)) return;
      localStorage.setItem(
        key,
        JSON.stringify({
          currentChatId: options.chatId,
          chats: [
            {
              id: options.chatId,
              title: options.title,
              context: options.context,
              workspaceName: workspace,
              createdTs: now - 1,
              updatedTs: now,
              status: "idle",
              totalTokensUsed: 0,
              archived: false,
            },
          ],
          messagesByChatId: {
            [options.chatId]: [
              {
                id: `${options.chatId}-user`,
                chatId: options.chatId,
                role: "user",
                content: options.prompt,
                createdTs: now - 1,
              },
              {
                id: `${options.chatId}-answer`,
                chatId: options.chatId,
                role: "assistant",
                content: options.answer,
                createdTs: now,
                metadata: options.metadata,
              },
            ],
          },
          pendingAskByChatId: {},
          draftsByChatId: {},
          queuedUserMessage: null,
        })
      );
      sessionStorage.setItem(marker, "1");
    },
    {
      key: `bb-agent-state-v2.${workspaceId.length}:${workspaceId}.${env.adminEmail}`,
      options,
      now,
      workspace,
    }
  );
}

async function openAgent(page: Page, env: TestEnv) {
  await page.goto(`${env.baseURL}/landing`);
  await page.getByRole("button", { name: "Agent", exact: true }).click();
}

function trackEffectfulAgentCalls(page: Page) {
  const calls: string[] = [];
  page.on("request", (request) => {
    if (
      request.method() === "POST" &&
      /\/(CreatePlan|CreateIssue|CreateDatabase|CreateInstance|TransferDatabase|CreateIssueComment|Query)$/.test(
        new URL(request.url()).pathname
      )
    ) {
      calls.push(request.url());
    }
  });
  return calls;
}

test("a saved Agent handoff opens the matching filtered Issue queue only after click", async ({
  page,
}) => {
  const env = loadTestEnv();
  const projectId = env.project.split("/").at(-1)!;
  const chatId = "agent-guided-handoff-e2e";
  await seedChat(page, env, {
    chatId,
    title: "Review changes",
    context: env.project,
    prompt: "What needs review?",
    answer: "Review the current change queue.",
    metadata: {
      handoff: { kind: "review-database-changes", project: env.project },
    },
  });
  await openAgent(page, env);

  const handoff = page.locator("[data-agent-handoff]");
  await expect(handoff).toHaveText(/Review database changes/);
  await expect(page).toHaveURL(/\/landing$/);
  await handoff.click();

  await expect(page).toHaveURL(new RegExp(`/projects/${projectId}/issues\\?`));
  await expect(page.locator('[data-search-scope-id="status"]')).toContainText("Open");
  await expect(page.locator('[data-search-scope-id="issue-type"]')).toContainText("Database Change");
});

test.describe("project setup handoffs", () => {
  for (const scenario of [
    {
      kind: "connect-project-instance",
      pathname: (projectId: string) => `/projects/${projectId}/instances/new`,
    },
    {
      kind: "create-project-database",
      pathname: (projectId: string) => `/projects/${projectId}/databases`,
    },
    {
      kind: "transfer-project-databases",
      pathname: () => "/databases",
    },
  ] as const) {
    test(`opens ${scenario.kind} only after the member clicks`, async ({ page }) => {
      const env = loadTestEnv();
      const projectId = env.project.split("/").at(-1)!;
      const chatId = `agent-${scenario.kind}-e2e`;
      await seedChat(page, env, {
        chatId,
        title: "Set up this project",
        context: env.project,
        prompt: "What should I set up next?",
        answer: "Continue with the recommended project setup step.",
        metadata: {
          handoff: { kind: scenario.kind, project: env.project },
        },
      });
      const effectfulCalls = trackEffectfulAgentCalls(page);
      await openAgent(page, env);

      const handoff = page.locator("[data-agent-handoff]");
      await expect(handoff).toBeVisible();
      await expect(page).toHaveURL(/\/landing$/);
      await handoff.click();

      await expect(page).toHaveURL((url) =>
        url.pathname === scenario.pathname(projectId)
      );
      if (scenario.kind === "create-project-database") {
        await expect(page).toHaveURL(/[?&]createDatabase=1(?:&|$)/);
      }
      if (scenario.kind === "transfer-project-databases") {
        expect(new URL(page.url()).searchParams.get("targetProject")).toBe(
          env.project
        );
      }
      expect(effectfulCalls).toEqual([]);
    });
  }
});

test("a Plan draft opens populated and unsaved after a member click", async ({
  page,
}) => {
  const env = loadTestEnv();
  const chatId = "agent-plan-draft-e2e";
  const sql = "CREATE TABLE agent_handoff_test (id INT PRIMARY KEY);";
  await seedChat(page, env, {
    chatId,
    title: "Draft a database change",
    context: env.project,
    prompt: "Create a table for this database.",
    answer: `Review this proposed change:\n\n\`\`\`sql\n${sql}\n\`\`\``,
    metadata: {
      actions: [
        {
          kind: "draft-plan",
          project: env.project,
          database: env.database,
        },
      ],
    },
  });
  const effectfulCalls = trackEffectfulAgentCalls(page);
  await openAgent(page, env);

  const draftPlan = page.locator("[data-agent-action-draft-plan]");
  await expect(draftPlan).toBeVisible();
  await expect(page).toHaveURL(/\/landing$/);
  await draftPlan.click();

  await expect(page).toHaveURL(/\/projects\/[^/]+\/plans\/create\?/);
  const destination = new URL(page.url());
  expect(destination.searchParams.get("databaseList")).toBe(env.database);
  expect(destination.searchParams.get("sqlStorageKey")).toBeTruthy();
  await expect(page.getByRole("code").first().locator(".view-lines")).toContainText(
    sql
  );
  expect(effectfulCalls).toEqual([]);
});

test("an Issue comment handoff opens an unsent draft on the exact Issue", async ({
  page,
}) => {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const projectId = `agent-handoff-${Date.now()}`;
  const project = await env.api.createProject(
    projectId,
    "Agent Handoff Test Project"
  );
  try {
    const issue = await env.api.createRoleGrantIssue(
      project.name,
      {
        role: "roles/projectDeveloper",
        user: `users/${env.adminEmail}`,
      },
      "Review project access"
    );
    const issueId = issue.name.split("/").at(-1)!;
    const chatId = "agent-issue-comment-handoff-e2e";
    const comment = "Please verify that this access is limited to the required scope.";
    await seedChat(page, env, {
      chatId,
      title: "Review issue security",
      context: project.name,
      prompt: "Any security concern for this Issue?",
      answer: "Review the proposed access before approval.",
      metadata: {
        actions: [
          {
            kind: "draft-issue-comment",
            issue: issue.name,
            comment,
          },
        ],
      },
    });
    let postedComment = false;
    page.on("request", (request) => {
      if (
        request.method() === "POST" &&
        /\/(?:CreateIssueComment|issues\/[^/]+:comment)$/.test(
          new URL(request.url()).pathname
        )
      ) {
        postedComment = true;
      }
    });
    await openAgent(page, env);

    const draftAction = page.locator("[data-agent-action-draft-issue-comment]");
    await expect(draftAction).toBeVisible();
    await expect(page).toHaveURL(/\/landing$/);
    await draftAction.click();

    await expect(page).toHaveURL(
      new RegExp(`/projects/${projectId}/issues/${issueId}\\?`)
    );
    await expect(page.locator("#issue-comment-editor textarea")).toHaveValue(
      comment
    );
    expect(postedComment).toBe(false);
  } finally {
    await env.api.deleteProject(project.name).catch(() => undefined);
  }
});
