import { fromBinary, fromJsonString, toBinary } from "@bufbuild/protobuf";
import type { Locator, Page, Request } from "@playwright/test";
import { AIChatRequestSchema, AIChatResponseSchema } from "../../../src/types/proto-es/v1/ai_service_pb";
import {
  QueryRequestSchema,
  QueryResponseSchema,
} from "../../../src/types/proto-es/v1/sql_service_pb";
import { storageKeyAiConversations } from "../../../src/utils/storage-keys";
import { answer, expect, test } from "../agent-token-budget/token-budget.fixture";
import { loadTestEnv } from "../framework/env";
import { loginBrowserContext } from "../framework/sign-in";
import { SqlEditorPage } from "./sql-editor.page";

const chatEndpoint =
  /\/(?:bytebase\.v1\.AIService\/Chat|v1\/ai\/chat)(?:\?|$)/;
const queryEndpoint = /\/bytebase\.v1\.SQLService\/Query(?:\?|$)/;
const initialSql = "SELECT 20;";
const proposedSql = "SELECT 731 AS agent_result;";
const exampleSql = "SELECT 732;";
const suggestedQuestions = ["Show row counts by table", "Which columns can join these tables?", "Show recent records"];
const proposal = {
  toolCalls: [
    {
      id: "editor-migration-done",
      name: "done",
      arguments: JSON.stringify({
        text: [
          "Proposed query",
          "```sql",
          proposedSql,
          "```",
          "Unmarked example",
          "```sql",
          exampleSql,
          "```",
        ].join("\n"),
        success: true,
        actions: [{ kind: "sql-editor-block", blockIndex: 0 }],
      }),
    },
  ],
  stopReason: "TURN_COMPLETED",
};

function requestBody(request: Request) {
  return request.postDataBuffer()!;
}

function isProto(request: Request) {
  return request.headers()["content-type"]?.startsWith("application/proto");
}

function agentTurn(request: Request) {
  if (!chatEndpoint.test(request.url())) return false;
  const body = requestBody(request);
  const chat = isProto(request) ? fromBinary(AIChatRequestSchema, body) : fromJsonString(AIChatRequestSchema, body.toString());
  return chat.messages.some(message => message.content.includes("- Frozen SQL Editor context for this turn"));
}

// Suggestion generation must not consume the conversation provider's queued replies.
test.beforeEach(async ({ context, provider: _provider }) => {
  await context.route(chatEndpoint, async route => {
    const request = route.request();
    const body = requestBody(request);
    const chat = isProto(request) ? fromBinary(AIChatRequestSchema, body) : fromJsonString(AIChatRequestSchema, body.toString());
    if (!chat.messages[0]?.content.startsWith("Generate three concise, useful read-only analysis/query questions for the SQL Editor,")) return route.fallback();
    const response = JSON.stringify({ content: JSON.stringify({ questions: suggestedQuestions }), stopReason: "TURN_COMPLETED" });
    await route.fulfill({ status: 200, contentType: isProto(request) ? "application/proto" : "application/json", body: isProto(request) ? Buffer.from(toBinary(AIChatResponseSchema, fromJsonString(AIChatResponseSchema, response))) : response });
  });
});

function frozenContext(request: Request) {
  const body = requestBody(request);
  const chat = isProto(request)
    ? fromBinary(AIChatRequestSchema, body)
    : fromJsonString(AIChatRequestSchema, body.toString());
  const line = chat.messages
    .flatMap((message) => message.content.split("\n"))
    .find((value) => value.startsWith("- Frozen SQL Editor context for this turn"));
  expect(line).toBeDefined();
  return JSON.parse(line!.slice(line!.indexOf("{"))) as {
    project: string;
    tab?: { id: string; mode: string };
    connection?: { database: string };
  };
}

function agentWindow(page: Page) {
  return page.locator("[data-agent-window][data-agent-presentation]");
}

function launcher(page: Page) {
  return page.getByRole("button", { name: "AI Assistant", exact: true });
}

async function openAgent(page: Page) {
  const agent = agentWindow(page);
  await launcher(page).click();
  await expect(agent).toBeVisible();
  await expect(agent).toHaveCount(1);
  return agent;
}

async function send(agent: Locator, prompt: string) {
  await agent.locator("textarea").fill(prompt);
  await agent.getByRole("button", { name: "Send", exact: true }).click();
}

async function openSavedQuery(page: Page, run: (editor: SqlEditorPage) => Promise<void>) {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const savedQuery = await env.api.createSavedQuery(
    env.project,
    "Agent migration query",
    env.database,
    initialSql
  );
  try {
    await page.goto(
      `${env.baseURL}/sql-editor/${env.project}/savedQueries/${savedQuery.name.split("/").at(-1)}`
    );
    const editor = new SqlEditorPage(page, env.baseURL);
    await expect(editor.activeTab()).toContainText("Agent migration query");
    await expect.poll(() => editor.readEditorContent()).toBe(initialSql);
    await run(editor);
  } finally {
    await page.close();
    await env.api.deleteSavedQuery(savedQuery.name);
  }
}

for (const width of [1440, 390]) {
  test(`Project-only SQL Editor opens and restores one Agent at ${width}px`, async ({
    page,
    provider,
  }, testInfo) => {
    const env = loadTestEnv();
    await page.setViewportSize({ width, height: 900 });
    provider.respond(answer(undefined, "Project-only answer without a database."));
    await page.goto(`${env.baseURL}/sql-editor/${env.project}`);
    const agent = await openAgent(page);
    await expect(agent).toHaveAttribute(
      "data-agent-presentation",
      width === 1440 ? "sql-editor-docked" : "sql-editor-overlay"
    );
    const context = agent.locator("[data-agent-context-trigger]");
    await expect(context).toBeDisabled();

    const outgoing = page.waitForRequest(agentTurn);
    await send(agent, "Explain what a SQL transaction is.");
    const snapshot = frozenContext(await outgoing);
    expect(snapshot.project).toBe(env.project);
    expect(snapshot.connection).toBeUndefined();
    await expect(agent.getByText("Project-only answer without a database.")).toBeVisible();
    expect(provider.calls).toBe(1);
    const separator = page.getByRole("separator", {
      name: "AI Assistant",
      exact: true,
    });
    let settledWidth: number | undefined;
    if (width === 1440) {
      await expect(separator).toHaveAttribute("aria-orientation", "vertical");
      await separator.press("End");
      const maximum = Number(await separator.getAttribute("aria-valuemax"));
      await expect(separator).toHaveAttribute("aria-valuenow", String(maximum));
      await separator.press("ArrowRight");
      settledWidth = maximum - 16;
      await expect(separator).toHaveAttribute("aria-valuenow", String(settledWidth));
      await expect
        .poll(async () => Math.round((await agent.boundingBox())!.width))
        .toBe(settledWidth);
    } else {
      await expect(separator).toHaveCount(0);
    }
    const bounds = (await agent.boundingBox())!;
    expect(bounds.x).toBeGreaterThanOrEqual(0);
    expect(bounds.x + bounds.width).toBeLessThanOrEqual(width + 1);
    expect(bounds.y + bounds.height).toBeLessThanOrEqual(901);
    await page.screenshot({ path: testInfo.outputPath(`editor-agent-${width}.png`) });

    await agent.getByTitle("Minimize", { exact: true }).click();
    await expect(agent).toHaveCount(0);
    await expect(separator).toHaveCount(0);
    await page.locator("button[data-agent-window]").click();
    await expect(agent).toHaveCount(1);
    await expect(agent.getByText("Project-only answer without a database.")).toBeVisible();
    if (settledWidth !== undefined) {
      await page.reload();
      await openAgent(page);
      await expect(separator).toHaveAttribute("aria-valuenow", String(settledWidth));
      expect(provider.calls).toBe(1);
    }
  });
}

test("only the marked SQL block can request a native Query, once allowed", async ({
  page,
  provider,
}, testInfo) => {
  const env = loadTestEnv();
  await page.setViewportSize({ width: 1440, height: 900 });
  provider.respond(proposal);
  await openSavedQuery(page, async (editor) => {
    const observed: Request[] = [];
    page.on("request", (request) => {
      if (queryEndpoint.test(request.url())) observed.push(request);
    });
    const agent = await openAgent(page);
    const outgoing = page.waitForRequest(agentTurn);
    await send(agent, "Suggest a harmless query for checking the connection.");
    expect(frozenContext(await outgoing).connection?.database).toBe(env.database);
    const marked = agent.locator("[data-agent-sql-block]").filter({ hasText: proposedSql });
    const example = agent.locator("[data-agent-sql-block]").filter({ hasText: exampleSql });
    await expect(example.getByRole("button", { name: "Copy SQL", exact: true })).toBeVisible();
    await expect(example.getByRole("button", { name: /^(Run query|Insert|Replace selection)$/ })).toHaveCount(0);
    await expect(marked.getByRole("button", { name: "Insert", exact: true })).toBeEnabled();
    await marked.getByRole("button", { name: "Run query", exact: true }).click();
    await expect(agent.getByText("Request permission", { exact: true })).toBeVisible();
    expect(observed).toHaveLength(0);
    await expect(agent.locator("[data-agent-pending-preview]")).toContainText(proposedSql);
    const query = page.waitForResponse((response) => queryEndpoint.test(response.url()));
    await agent.getByRole("button", { name: "Allow once", exact: true }).click();
    expect((await query).ok()).toBe(true);
    const row = page
      .locator('[data-row-index="0"]:not([data-agent-window] *)')
      .filter({ hasText: "731" });
    await expect(row).toBeVisible();
    const resultBounds = (await row.boundingBox())!;
    const agentBounds = (await agent.boundingBox())!;
    const tabs = page.locator(".bb-sql-editor-tab-list");
    const tabBounds = (await tabs.boundingBox())!;
    expect(tabBounds.x + tabBounds.width).toBeLessThanOrEqual(agentBounds.x + 1);
    expect(Math.abs(tabBounds.y - agentBounds.y)).toBeLessThanOrEqual(1);
    expect(resultBounds.y + resultBounds.height).toBeLessThanOrEqual(agentBounds.y + agentBounds.height + 1);
    expect(resultBounds.x + resultBounds.width).toBeLessThanOrEqual(agentBounds.x + 1);
    expect(observed).toHaveLength(1);
    const request = observed[0]!;
    const body = requestBody(request);
    const parsed = isProto(request)
      ? fromBinary(QueryRequestSchema, body)
      : fromJsonString(QueryRequestSchema, body.toString());
    expect(parsed).toMatchObject({ name: env.database, statement: proposedSql });
    expect(parsed.limit).toBeGreaterThan(0);
    expect(provider.calls).toBe(1);
    await expect.poll(() => editor.readEditorContent()).toBe(initialSql);
    await page.screenshot({ path: testInfo.outputPath("approved-query-results.png") });
  });
});

test("Project Viewer cannot execute the marked SQL through one-time Agent approval", async ({
  context,
  page,
  provider,
}) => {
  const env = loadTestEnv();
  const email = `agent-migration-viewer-${Date.now()}@example.com`;
  const password = "agent-migration-test-123!";
  await env.api.login(env.adminEmail, env.adminPassword);
  await env.api.createUser(email, password, "SQL Editor Agent test viewer");
  try {
    await env.api.appendProjectBinding(env.project, "roles/projectViewer", [`user:${email}`]);
    await loginBrowserContext(context, env.baseURL, email, password);
    provider.respond(proposal);
    await page.goto(`${env.baseURL}/sql-editor/${env.project}/instances/${env.instanceId}/databases/${env.databaseId}`);
    await expect(new SqlEditorPage(page).codeEditor).toBeVisible();
    const sqlRequests: string[] = [];
    page.on("request", (request) => {
      if (/\/bytebase\.v1\.SQLService\/(?:Query|Execute|AdminExecute)(?:\?|$)/.test(request.url())) {
        sqlRequests.push(request.url());
      }
    });
    const agent = await openAgent(page);
    await send(agent, "Propose a sample SQL query for this database.");
    const block = agent.locator("[data-agent-sql-block]").filter({ hasText: proposedSql });
    await block.getByRole("button", { name: "Run query", exact: true }).click();
    const allow = agent.getByRole("button", { name: "Allow once", exact: true });
    await expect(allow).toBeVisible();
    expect(sqlRequests).toHaveLength(0);
    const deniedQuery = page.waitForResponse((response) => queryEndpoint.test(response.url()));
    await allow.click();
    const response = await deniedQuery;
    if (response.ok()) {
      const body = await response.body();
      const query = response.headers()["content-type"]?.startsWith("application/proto")
        ? fromBinary(QueryResponseSchema, body)
        : fromJsonString(QueryResponseSchema, body.toString());
      expect(query.results.some(({ detailedError }) => detailedError.case === "permissionDenied")).toBe(true);
      expect(query.results.flatMap(({ rows }) => rows)).toHaveLength(0);
    } else {
      expect(await response.json()).toMatchObject({ code: "permission_denied" });
    }
    await expect(allow).toBeHidden();
    await expect(page.locator('[data-row-index="0"]').filter({ hasText: "731" })).toHaveCount(0);
    expect(sqlRequests).toEqual([response.url()]);
    expect(provider.calls).toBe(1);
  } finally {
    await page.close();
    try {
      const policy = await env.api.getProjectIamPolicy(env.project);
      policy.bindings = policy.bindings.flatMap((binding) => {
        if (!binding.members.includes(`user:${email}`)) return [binding];
        const members = binding.members.filter((member) => member !== `user:${email}`);
        return members.length ? [{ ...binding, members }] : [];
      });
      await env.api.setProjectIamPolicy(env.project, policy);
    } finally {
      await env.api.deleteUser(email);
    }
  }
});

test("an in-flight Agent conversation and draft survive an app route change and return", async ({
  page,
  provider,
}) => {
  const env = loadTestEnv();
  const editorUrl = `${env.baseURL}/sql-editor/${env.project}/instances/${env.instanceId}/databases/${env.databaseId}`;
  const prompt = "Explain the current database connection.";
  const draft = "This unsent follow-up stays in the composer.";
  const release = provider.hold(answer(undefined, "The original turn completed."));
  await page.setViewportSize({ width: 1440, height: 900 });
  try {
    await page.goto(editorUrl);
    const agent = await openAgent(page);
    await expect(agent).toHaveAttribute("data-agent-presentation", "sql-editor-docked");
    const outgoing = page.waitForRequest(agentTurn);
    await send(agent, prompt);
    expect(frozenContext(await outgoing).connection?.database).toBe(env.database);
    const input = agent.locator("textarea");
    const submitted = agent
      .locator("[data-agent-user-message]")
      .getByText(prompt, { exact: true });
    await input.fill(draft);

    const profile = page.locator("header").getByRole("button", {
      name: "D",
      exact: true,
    });
    await profile.click();
    const account = page.getByRole("menuitem").filter({ hasText: env.adminEmail });
    await expect(account).toBeVisible();
    await expect(agent).toHaveCount(0);
    await account.click({ timeout: 5_000 });
    await expect(page).toHaveURL(/\/account(?:\?|$)/);
    await expect(agent).toHaveAttribute("data-agent-presentation", "floating");
    await expect(agent).toHaveCount(1);
    await expect(input).toHaveValue(draft);
    await expect(submitted).toHaveCount(1);

    await page.goBack();
    await expect(page).toHaveURL(/\/sql-editor\//);
    await expect(agent).toHaveAttribute("data-agent-presentation", "sql-editor-docked");
    await expect(agent).toHaveCount(1);
    await expect(input).toHaveValue(draft);
    release();
    await expect(submitted).toHaveCount(1);
    await expect(agent.getByText("The original turn completed.")).toBeVisible();
    await expect(input).toHaveValue(draft);
    expect(provider.calls).toBe(1);
  } finally {
    release();
  }
});

for (const width of [1440, 390]) {
  test(`switching the live database changes only the next Agent turn at ${width}px`, async ({
    page,
    provider,
  }) => {
    const env = loadTestEnv();
    const nextDatabase = env.databaseId === "hr_prod" ? "hr_test" : "hr_prod";
    const release = provider.hold(proposal);
    const presentation = width === 1440 ? "sql-editor-docked" : "sql-editor-overlay";
    const prompt = "Propose a query for the current database.";
    await page.setViewportSize({ width, height: 900 });
    try {
      await openSavedQuery(page, async () => {
        const agent = await openAgent(page);
        await expect(agent).toHaveAttribute("data-agent-presentation", presentation);
        const first = page.waitForRequest(agentTurn);
        await send(agent, prompt);
        expect(frozenContext(await first).connection?.database).toBe(env.database);

        await page.getByRole("button").filter({ hasText: env.databaseId }).first().click({
          timeout: 5_000,
        });
        const chooser = page.getByRole("dialog", { name: "Select database", exact: true });
        await expect(chooser).toBeVisible();
        await expect(agent).toHaveCount(0);
        await chooser.getByText(nextDatabase, { exact: true }).click({
          timeout: 5_000,
        });
        await expect(chooser).toBeHidden();
        await expect(page.getByRole("button").filter({ hasText: nextDatabase }).first()).toBeVisible();
        await expect(agent).toHaveAttribute("data-agent-presentation", presentation);
        await expect(agent).toHaveCount(1);
        await expect(agent.getByText(prompt, { exact: true })).toHaveCount(1);
        release();

        const prior = agent.locator("[data-agent-sql-block]").filter({ hasText: proposedSql });
        await expect(prior.getByRole("button", { name: "Copy SQL", exact: true })).toBeVisible();
        await expect(prior.getByRole("button", { name: "Run query", exact: true })).toBeDisabled();
        provider.respond(answer(undefined, "Second database context received."));
        const second = page.waitForRequest(agentTurn);
        await send(agent, "Which database is selected for this turn?");
        expect(frozenContext(await second).connection?.database).toBe(
          `${env.database.replace(/\/databases\/[^/]+$/, "")}/databases/${nextDatabase}`
        );
        await expect(agent.getByText("Second database context received.")).toBeVisible();
        await expect(agent).toHaveCount(1);
        expect(provider.calls).toBe(2);
      });
    } finally {
      release();
    }
  });
}

for (const action of ["Insert", "Replace selection"] as const) {
  test(`${action} preserves Undo and keeps fresh Insert and Run available`, async ({
    page,
    provider,
  }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    provider.respond(proposal);
    await openSavedQuery(page, async (editor) => {
      await editor.codeEditor.click();
      const replace = action === "Replace selection";
      await page.keyboard.press(replace ? "Control+a" : "End");
      const agent = await openAgent(page);
      await send(agent, "Propose a sample SQL query for this editor.");
      const marked = agent.locator("[data-agent-sql-block]").filter({ hasText: proposedSql });
      await marked.getByRole("button", { name: action, exact: true }).click();
      await expect
        .poll(() => editor.readEditorContent())
        .toBe(replace ? proposedSql : `${initialSql}\n${proposedSql}`);
      await expect(marked.getByRole("button", { name: "Insert", exact: true })).toBeEnabled();
      await expect(marked.getByRole("button", { name: "Run query", exact: true })).toBeEnabled();
      if (replace) await expect(marked.getByRole("button", { name: "Replace selection", exact: true })).toBeDisabled();
      await editor.codeEditor.first().click();
      await page.keyboard.press("Control+z");
      await expect.poll(() => editor.readEditorContent()).toBe(initialSql);
      expect(provider.calls).toBe(1);
    });
  });
}

for (const decision of ["deny", "database-change", "editor-change"] as const) {
  test(`SQL response approval handles ${decision} without retargeting or replay`, async ({ page, provider }) => {
    const env = loadTestEnv();
    await page.setViewportSize({ width: 1440, height: 900 });
    provider.respond(proposal);
    await openSavedQuery(page, async (editor) => {
      const requests: Request[] = [];
      page.on("request", request => {
        if (/\/bytebase\.v1\.SQLService\/(?:Query|Execute|AdminExecute)(?:\?|$)/.test(request.url())) requests.push(request);
      });
      const agent = await openAgent(page);
      await send(agent, "Propose a sample SQL query for this database.");
      const block = agent.locator("[data-agent-sql-block]").filter({ hasText: proposedSql });
      await block.getByRole("button", { name: "Run query", exact: true }).click();
      const allow = agent.getByRole("button", { name: "Allow once", exact: true });
      await expect(allow).toBeEnabled();
      expect(requests).toHaveLength(0);
      if (decision === "deny") {
        await agent.getByRole("button", { name: "Deny", exact: true }).click();
      } else if (decision === "database-change") {
        const nextDatabase = env.databaseId === "hr_prod" ? "hr_test" : "hr_prod";
        await page.getByRole("button").filter({ hasText: env.databaseId }).first().click();
        const chooser = page.getByRole("dialog", { name: "Select database", exact: true });
        await expect(chooser).toBeVisible();
        await chooser.getByText(nextDatabase, { exact: true }).click();
        await expect(chooser).toBeHidden();
        await expect(allow).toBeDisabled();
        await agent.getByRole("button", { name: "Dismiss", exact: true }).click();
      } else {
        const newerSql = `${initialSql} -- newer editor text`;
        await editor.codeEditor.first().click();
        await page.keyboard.press("Control+a");
        await page.keyboard.type(newerSql);
        await expect.poll(() => editor.readEditorContent()).toBe(newerSql);
        await expect(allow).toBeEnabled();
        const response = page.waitForResponse(value => queryEndpoint.test(value.url()));
        await allow.click();
        expect((await response).ok()).toBe(true);
        await expect.poll(() => editor.readEditorContent()).toBe(newerSql);
      }
      await expect(allow).toBeHidden();
      expect(requests).toHaveLength(decision === "editor-change" ? 1 : 0);
      if (requests.length) {
        const request = requests[0]!;
        const body = requestBody(request);
        const parsed = isProto(request) ? fromBinary(QueryRequestSchema, body) : fromJsonString(QueryRequestSchema, body.toString());
        expect(parsed).toMatchObject({ name: env.database, statement: proposedSql });
      }
      expect(provider.calls).toBe(1);
    });
  });
}

test("switching Project captures the new Project without leaking the old database", async ({ page, provider }) => {
  const env = loadTestEnv();
  const title = `Agent migration Project ${Date.now()}`;
  await env.api.login(env.adminEmail, env.adminPassword);
  const project = await env.api.createProject(`agent-migration-${Date.now()}`, title);
  try {
    provider.respond(answer(undefined, "Original Project answer."));
    await page.goto(`${env.baseURL}/sql-editor/${env.project}/instances/${env.instanceId}/databases/${env.databaseId}`);
    const agent = await openAgent(page);
    const first = page.waitForRequest(agentTurn);
    await send(agent, "Which database is selected?");
    expect(frozenContext(await first).connection?.database).toBe(env.database);
    await expect(agent.getByText("Original Project answer.")).toBeVisible();
    await page.getByRole("button", { name: "Select project" }).first().click();
    await page.getByPlaceholder("Filter by name").fill(title);
    await page.getByText(title, { exact: true }).first().click();
    await expect(page.locator("header").getByText(title, { exact: true })).toBeVisible();
    await expect(agent.getByText("Original Project answer.")).toHaveCount(0);
    provider.respond(answer(undefined, "New Project answer without a database."));
    const second = page.waitForRequest(agentTurn);
    await send(agent, "Show databases in this project");
    const snapshot = frozenContext(await second);
    expect(snapshot.project).toBe(project.name);
    expect(snapshot.connection).toBeUndefined();
    await expect(agent.getByText("New Project answer without a database.")).toBeVisible();
    expect(provider.calls).toBe(2);
  } finally {
    await page.close();
    await env.api.deleteProject(project.name);
  }
});

test("live resizing preserves one running Agent and its draft across the dock boundary", async ({ page, provider }, testInfo) => {
  const env = loadTestEnv();
  const prompt = "Explain the current connection while I resize the editor.";
  const draft = "Keep this unsent SQL follow-up.";
  const release = provider.hold(answer(undefined, "The resized conversation completed."));
  await page.setViewportSize({ width: 1600, height: 900 });
  try {
    await page.goto(`${env.baseURL}/sql-editor/${env.project}/instances/${env.instanceId}/databases/${env.databaseId}`);
    const agent = await openAgent(page);
    const column = page.getByTestId("sql-editor-agent-main-column");
    const outgoing = page.waitForRequest(agentTurn);
    await send(agent, prompt);
    await outgoing;
    const input = agent.locator("textarea");
    await input.fill(draft);
    for (const target of [961, 959, 961]) {
      for (let attempt = 0; attempt < 8; attempt++) {
        const viewport = page.viewportSize()!;
        const width = (await column.boundingBox())!.width;
        if (Math.abs(width - target) < 0.75) break;
        await page.setViewportSize({ width: viewport.width + Math.round(target - width), height: 900 });
        await expect.poll(async () => (await column.boundingBox())!.width).not.toBe(width);
      }
      const width = (await column.boundingBox())!.width;
      expect(Math.abs(width - target)).toBeLessThan(0.75);
      expect(width >= 960).toBe(target >= 960);
      await expect(agent).toHaveAttribute("data-agent-presentation", target >= 960 ? "sql-editor-docked" : "sql-editor-overlay");
      await expect(agent).toHaveCount(1);
      await expect(input).toHaveValue(draft);
      await expect(agent.locator("[data-agent-user-message]").getByText(prompt, { exact: true })).toHaveCount(1);
      await page.screenshot({ path: testInfo.outputPath(`live-resize-${target}.png`) });
    }
    await page.setViewportSize({ width: 390, height: 900 });
    await expect(agent).toHaveAttribute("data-agent-presentation", "sql-editor-overlay");
    await expect(input).toHaveValue(draft);
    const bounds = (await agent.boundingBox())!;
    expect(bounds.x).toBeGreaterThanOrEqual(0);
    expect(bounds.x + bounds.width).toBeLessThanOrEqual(391);
    await page.screenshot({ path: testInfo.outputPath("live-resize-mobile.png") });
    await page.setViewportSize({ width: 1600, height: 900 });
    await expect(agent).toHaveAttribute("data-agent-presentation", "sql-editor-docked");
    await expect(input).toHaveValue(draft);
    release();
    await expect(agent.getByText("The resized conversation completed.")).toBeVisible();
    await expect(input).toHaveValue(draft);
    expect(provider.calls).toBe(1);
  } finally {
    release();
  }
});

test("legacy SQL chat is imported into shared history as a read-only transcript", async ({
  page,
  provider,
}) => {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const { workspace } = await env.api.getActuatorInfo();
  const key = storageKeyAiConversations(workspace, env.adminEmail);
  const title = "Legacy sample SQL chat";
  const inaccessibleTitle = "Legacy inaccessible SQL chat";
  const prompt = "Show a sample query";
  const content = ["Historical example", "", "```sql", exampleSql, "```"].join("\n");
  const createdTs = Date.now() - 60_000;
  const source = JSON.stringify({
    version: 1,
    conversations: [{
      id: "legacy-sql-migration-e2e",
      created_ts: createdTs,
      name: title,
      database: env.database,
      instance: env.instance,
      messageList: [
        { id: "legacy-user", created_ts: createdTs, author: "USER", content: prompt, status: "DONE", error: "" },
        { id: "legacy-answer", created_ts: createdTs + 1, author: "AI", content, status: "DONE", error: "" },
      ],
    }, {
      id: "legacy-sql-inaccessible-e2e",
      created_ts: createdTs,
      name: inaccessibleTitle,
      database: `${env.instance}/databases/agent_migration_missing_database`,
      instance: env.instance,
      messageList: [],
    }],
  });
  await page.addInitScript(({ key, source }) => localStorage.setItem(key, source), { key, source });
  await page.goto(`${env.baseURL}/sql-editor/${env.project}/instances/${env.instanceId}/databases/${env.databaseId}`);
  const agent = await openAgent(page);
  const historyRow = agent.getByRole("button", { name: title, exact: true });
  const archived = agent.getByRole("button", { name: "View archived chats", exact: true });
  if (!(await archived.isVisible())) {
    await agent.getByRole("button", { name: "Chats", exact: true }).click();
  }
  await archived.click();
  await expect(agent.getByRole("button", { name: inaccessibleTitle, exact: true })).toHaveCount(0);
  await historyRow.click();
  await expect(agent.getByText(prompt, { exact: true })).toBeVisible();
  await expect(agent.getByText("Historical example", { exact: true })).toBeVisible();
  await expect(agent.getByRole("button", { name: /^(Send|Resend|Insert|Replace selection|Run query)$/ })).toHaveCount(0);
  expect(await page.evaluate((key) => localStorage.getItem(key), key)).toBe(source);
  await page.reload();
  await openAgent(page);
  if (!(await archived.isVisible())) await agent.getByRole("button", { name: "Chats", exact: true }).click();
  await archived.click();
  await expect(historyRow).toHaveCount(1);
  await expect(agent.getByRole("button", { name: inaccessibleTitle, exact: true })).toHaveCount(0);
  await historyRow.click();
  await expect(agent.getByText("Historical example", { exact: true })).toBeVisible();
  await expect(agent.getByRole("button", { name: /^(Send|Resend|Insert|Replace selection|Run query)$/ })).toHaveCount(0);
  expect(provider.calls).toBe(0);
});

test("connected database suggestions remain stable when the Agent is reopened", async ({ page, provider }) => {
  const env = loadTestEnv();
  const requests: Request[] = [];
  page.on("request", request => { if (chatEndpoint.test(request.url())) requests.push(request); });
  await page.goto(`${env.baseURL}/sql-editor/${env.project}/instances/${env.instanceId}/databases/${env.databaseId}`);
  const agent = await openAgent(page);
  for (const question of suggestedQuestions) await expect(agent.getByRole("button", { name: question, exact: true })).toBeVisible();
  expect(requests).toHaveLength(1);
  await launcher(page).click();
  await expect(agent).toHaveCount(0);
  await launcher(page).click();
  for (const question of suggestedQuestions) await expect(agent.getByRole("button", { name: question, exact: true })).toBeVisible();
  expect(requests).toHaveLength(1);
  expect(provider.calls).toBe(0);
});

test("queued SQL questions retain their captured database after a connection switch", async ({ page, provider }) => {
  const env = loadTestEnv();
  const nextDatabase = env.databaseId === "hr_prod" ? "hr_test" : "hr_prod";
  const release = provider.hold(answer(undefined, "The original SQL question completed."));
  provider.respond(answer(undefined, "The queued SQL question completed."));
  try {
    await page.goto(`${env.baseURL}/sql-editor/${env.project}/instances/${env.instanceId}/databases/${env.databaseId}`);
    const agent = await openAgent(page);
    const first = page.waitForRequest(agentTurn);
    await send(agent, "Explain the current database.");
    await first;
    await agent.locator("textarea").fill("Explain another feature of the same database.");
    await agent.getByRole("button", { name: "Queue", exact: true }).click();
    await expect(agent.getByText("1 queued", { exact: true })).toBeVisible();
    await page.getByRole("button").filter({ hasText: env.databaseId }).first().click();
    const chooser = page.getByRole("dialog", { name: "Select database", exact: true });
    await chooser.getByText(nextDatabase, { exact: true }).click();
    await expect(chooser).toBeHidden();
    const queued = page.waitForRequest(agentTurn);
    release();
    expect(frozenContext(await queued).connection?.database).toBe(env.database);
    await expect(agent.getByText("The queued SQL question completed.")).toBeVisible();
    provider.respond(answer(undefined, "The newly selected database is used."));
    const live = page.waitForRequest(agentTurn);
    await send(agent, "Explain the database selected now.");
    expect(frozenContext(await live).connection?.database).toBe(`${env.database.replace(/\/databases\/[^/]+$/, "")}/databases/${nextDatabase}`);
    await expect(agent.getByText("The newly selected database is used.")).toBeVisible();
    expect(provider.calls).toBe(3);
  } finally {
    release();
  }
});

test("Admin mode retains Project help without native SQL response actions", async ({ page, provider }) => {
  const env = loadTestEnv();
  provider.respond(proposal);
  await openSavedQuery(page, async editor => {
    await editor.adminModeButton.click();
    const agent = await openAgent(page);
    const outgoing = page.waitForRequest(agentTurn);
    await send(agent, "Explain this connection without executing any SQL.");
    const snapshot = frozenContext(await outgoing);
    expect(snapshot.project).toBe(env.project);
    expect(snapshot.tab?.mode).toBe("ADMIN");
    const block = agent.locator("[data-agent-sql-block]").filter({ hasText: proposedSql });
    await expect(block.getByRole("button", { name: "Copy SQL", exact: true })).toBeVisible();
    await expect(agent.getByRole("button", { name: /^(Insert|Replace selection|Run query)$/ })).toHaveCount(0);
    expect(provider.calls).toBe(1);
  });
});
