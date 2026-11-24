import { fromBinary, fromJsonString } from "@bufbuild/protobuf";
import type { Page } from "@playwright/test";
import {
  type AIChatRequest,
  AIChatRequestSchema,
} from "../../../src/types/proto-es/v1/ai_service_pb";
import {
  answer,
  expect,
  test,
} from "../agent-token-budget/token-budget.fixture";
import { loadTestEnv } from "../framework/env";

test.setTimeout(90_000);

const projectRead = "bytebase.v1.ProjectService.GetProject";
const projectUpdate = "bytebase.v1.ProjectService.UpdateProject";
const windowOf = (page: Page) =>
  page.locator("[data-agent-window]:not(button)");
const button = (page: Page, name: string) =>
  windowOf(page).getByRole("button", { name, exact: true });
const tool = (id: string, name: string, args: object) => ({
  stopReason: "TURN_COMPLETED",
  toolCalls: [{ id, name, arguments: JSON.stringify(args) }],
});

async function send(page: Page, text: string) {
  await windowOf(page).locator("textarea").last().fill(text);
  await button(page, "Send").click();
}

function resultOf(requests: AIChatRequest[], id: string) {
  const result = requests
    .flatMap((request) => request.messages)
    .find((message) => message.toolCallId === id)?.content;
  if (!result) throw new Error(`Missing provider-bound tool result ${id}`);
  return JSON.parse(result);
}

test("a diagnostic chat keeps exact one-time API permission and its project boundary across follow-ups", async ({
  page,
  context,
  provider,
}, testInfo) => {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const { workspace } = await env.api.getActuatorInfo();
  const suffix = Date.now();
  const title = "Diagnostic API permission fixture";
  const project = await env.api.createProject(
    `agent-diagnostic-${suffix}`,
    title,
    { allowRequestRole: true },
  );
  let foreignProject: { name: string } | undefined;
  try {
    foreignProject = await env.api.createProject(
      `agent-diagnostic-other-${suffix}`,
      "Other diagnostic project fixture",
    );
    const issue = await env.api.createRoleGrantIssue(
      project.name,
      { role: "roles/projectViewer", user: `users/${env.adminEmail}` },
      "Diagnostic scope fixture",
    );
    const capability = {
      kind: "diagnostic",
      resource: {
        kind: "issue",
        name: issue.name,
        project: project.name,
        label: issue.name,
      },
    };
    const chatId = `diagnostic-api-${suffix}`;
    const workspaceId = workspace.split("/").at(-1)!;
    const storageKey = `bb-agent-state-v3.${workspaceId.length}:${workspaceId}.${env.adminEmail}`;
    await page.addInitScript(
      ({ storageKey, workspace, project, capability, chatId }) => {
        const now = Date.now();
        localStorage.setItem(
          storageKey,
          JSON.stringify({
            currentChatId: chatId,
            chats: [
              {
                id: chatId,
                title: "Diagnostic API permission",
                context: project,
                workspaceName: workspace,
                capability,
                createdTs: now,
                updatedTs: now,
                status: "idle",
                totalTokensUsed: 0,
                archived: false,
              },
            ],
            messagesByChatId: { [chatId]: [] },
            pendingAskByChatId: {},
            draftsByChatId: {},
            queuedUserMessagesByChatId: {},
          }),
        );
        localStorage.setItem(
          "bb.release",
          JSON.stringify({
            ignoreRemindModalTillNextRelease: true,
            nextCheckTs: now + 86_400_000,
          }),
        );
      },
      { storageKey, workspace, project: project.name, capability, chatId },
    );
    const state = () =>
      page.evaluate(
        ({ storageKey, chatId }) => {
          const saved = JSON.parse(localStorage.getItem(storageKey) ?? "{}");
          const chat = saved.chats?.find(
            (item: { id: string }) => item.id === chatId,
          );
          return {
            currentChatId: saved.currentChatId,
            capability: chat?.capability,
            pending: saved.pendingAskByChatId?.[chatId],
            drafts: saved.draftsByChatId?.[chatId],
          };
        },
        { storageKey, chatId },
      );

    const providerRequests: AIChatRequest[] = [];
    const updates: unknown[] = [];
    const reads: unknown[] = [];
    context.on("request", (request) => {
      if (new URL(request.url()).pathname !== "/bytebase.v1.AIService/Chat")
        return;
      const body = request.postDataBuffer();
      if (!body) throw new Error("Missing Chat request body");
      providerRequests.push(
        request.headers()["content-type"]?.startsWith("application/proto")
          ? fromBinary(AIChatRequestSchema, body)
          : fromJsonString(AIChatRequestSchema, body.toString()),
      );
    });
    await context.route(
      "**/bytebase.v1.ProjectService/UpdateProject",
      async (route) => {
        const body = route.request().postDataJSON();
        updates.push(body);
        expect(body).toEqual({
          project: { name: project.name, title: "Approved diagnostic title" },
          updateMask: "title",
        });
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            name: project.name,
            title: "Approved diagnostic title",
            state: "ACTIVE",
          }),
        });
      },
    );
    context.on("request", (request) => {
      if (
        new URL(request.url()).pathname ===
          "/bytebase.v1.ProjectService/GetProject" &&
        request.headers()["content-type"]?.startsWith("application/json")
      )
        reads.push(request.postDataJSON());
    });

    await page.goto(`${env.baseURL}/landing`);
    await page.getByRole("button", { name: "Agent", exact: true }).click();
    const agent = windowOf(page);
    await expect(agent).toBeVisible();
    const row = agent.getByRole("button", {
      name: "Diagnostic API permission",
      exact: true,
    });
    if (!(await row.isVisible())) await button(page, "Chats").first().click();
    await row.click();

    provider.respond(
      tool("safe-read", "call_api", {
        operationId: projectRead,
        body: { name: project.name },
      }),
    );
    provider.respond(answer(10, "The diagnostic project read completed."));
    await send(page, "Read this diagnostic project's current title.");
    await expect(
      agent.getByText("The diagnostic project read completed.", {
        exact: true,
      }),
    ).toBeVisible();
    await expect(button(page, "Allow once")).toBeHidden();
    expect(reads).toEqual([{ name: project.name }]);
    expect(resultOf(providerRequests, "safe-read")).toMatchObject({
      status: 200,
    });

    const approvedBody = {
      project: { name: project.name, title: "Approved diagnostic title" },
      updateMask: "title",
    };
    provider.respond(
      tool("update-schema", "search_api", { operationId: projectUpdate }),
    );
    provider.respond(
      tool("approved-update", "call_api", {
        operationId: projectUpdate,
        body: approvedBody,
      }),
    );
    provider.respond(
      answer(10, "The approved request completed in this diagnostic chat."),
    );
    await send(
      page,
      "Change this diagnostic project's title to Approved diagnostic title after my one-time approval.",
    );
    const allow = button(page, "Allow once");
    await expect(allow).toBeEnabled();
    await expect(
      agent.getByText("Request permission", { exact: true }),
    ).toBeVisible();
    const preview = agent
      .locator("pre")
      .filter({ hasText: '"title": "Approved diagnostic title"' });
    await expect(preview).toContainText(`"name": "${project.name}"`);
    await expect(preview).toContainText('"updateMask": "title"');
    await expect
      .poll(async () => (await state()).pending?.apiPermission?.operationId)
      .toBe(projectUpdate);
    const approvedTicket = (await state()).pending;
    expect(approvedTicket.toolCallId).toBe("approved-update");
    expect(approvedTicket.apiPermission.actionId).toMatch(/^[a-f0-9]{32}$/);
    expect(updates).toEqual([]);
    expect((await state()).capability).toEqual(capability);
    await agent.screenshot({ path: testInfo.outputPath("diagnostic-api-confirmation.png") });

    await allow.click();
    await expect(
      agent.getByText(
        "The approved request completed in this diagnostic chat.",
        { exact: true },
      ),
    ).toBeVisible();
    await expect(allow).toBeHidden();
    expect(updates).toEqual([approvedBody]);
    expect(resultOf(providerRequests, "approved-update")).toMatchObject({
      status: 200,
      agentExecution: { reason: "rpc-success" },
    });
    await expect.poll(async () => (await state()).pending).toBeUndefined();
    expect((await state()).capability).toEqual(capability);

    provider.respond(
      tool("denied-update", "call_api", {
        operationId: projectUpdate,
        body: {
          project: { name: project.name, title: "Denied diagnostic title" },
          updateMask: "title",
        },
      }),
    );
    provider.respond(answer(10, "The denied request was not sent."));
    await send(
      page,
      "Prepare another title change, but wait for my permission again.",
    );
    await expect(allow).toBeEnabled();
    await expect(
      agent
        .locator("pre")
        .filter({ hasText: '"title": "Denied diagnostic title"' }),
    ).toBeVisible();
    await expect
      .poll(async () => (await state()).pending?.toolCallId)
      .toBe("denied-update");
    const deniedTicket = (await state()).pending;
    expect(deniedTicket.apiPermission.actionId).not.toBe(
      approvedTicket.apiPermission.actionId,
    );
    expect(updates).toEqual([approvedBody]);
    await button(page, "Deny").click();
    await expect(
      agent.getByText("The denied request was not sent.", { exact: true }),
    ).toBeVisible();
    expect(resultOf(providerRequests, "denied-update")).toMatchObject({
      agentExecution: { reason: "permission-rejected" },
    });
    expect(updates).toEqual([approvedBody]);

    provider.respond(
      tool("foreign-update", "call_api", {
        operationId: projectUpdate,
        body: {
          project: {
            name: foreignProject.name,
            title: "Forbidden foreign title",
          },
          updateMask: "title",
        },
      }),
    );
    provider.respond(
      answer(10, "The other project remains outside this diagnostic chat."),
    );
    await send(page, "Try the same title change in the other project.");
    await expect(
      agent.getByText(
        "The other project remains outside this diagnostic chat.",
        { exact: true },
      ),
    ).toBeVisible();
    await expect(allow).toBeHidden();
    await expect(
      agent.getByText("Request permission", { exact: true }),
    ).toBeHidden();
    await expect.poll(async () => (await state()).pending).toBeUndefined();
    expect(resultOf(providerRequests, "foreign-update")).toMatchObject({
      agentExecution: { reason: "preflight-rejected" },
    });
    expect(updates).toEqual([approvedBody]);

    provider.respond(
      tool("continued-read", "call_api", {
        operationId: projectRead,
        body: { name: project.name },
      }),
    );
    provider.respond(
      tool("blocked-dom", "dom_action", { type: "click", ref: "e1" }),
    );
    provider.respond(
      tool("blocked-draft", "prepare_access_grant_draft", {
        project: project.name,
        database: env.database,
        statement: "SELECT 1",
      }),
    );
    provider.respond(
      answer(
        10,
        "The diagnosis continues without enabling page or draft actions.",
      ),
    );
    await send(
      page,
      "Continue diagnosing this project without making another change.",
    );
    await expect(
      agent.getByText(
        "The diagnosis continues without enabling page or draft actions.",
        { exact: true },
      ),
    ).toBeVisible();
    await expect(allow).toBeHidden();
    expect(reads).toEqual([{ name: project.name }, { name: project.name }]);
    for (const id of ["blocked-dom", "blocked-draft"]) {
      expect(resultOf(providerRequests, id)).toMatchObject({
        error: expect.stringContaining("not available in a diagnostic chat"),
      });
    }
    expect(await state()).toMatchObject({ currentChatId: chatId, capability });
    expect((await state()).drafts).toBeUndefined();
    expect(updates).toEqual([approvedBody]);
    expect((await env.api.getProject(project.name)).title).toBe(title);
    expect((await env.api.getProject(foreignProject.name)).title).toBe(
      "Other diagnostic project fixture",
    );
    expect(provider.calls).toBe(13);
  } finally {
    if (foreignProject) await env.api.deleteProject(foreignProject.name);
    await env.api.deleteProject(project.name);
  }
});
