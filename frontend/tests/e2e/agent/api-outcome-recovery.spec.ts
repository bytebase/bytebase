import { fromJsonString } from "@bufbuild/protobuf";
import { expect, type Page, type Request, test } from "@playwright/test";
import { QueryResponseSchema } from "../../../src/types/proto-es/v1/sql_service_pb";
import {
  answer,
  DeterministicProvider,
} from "../agent-token-budget/token-budget.fixture";
import { TokenBudgetPage } from "../agent-token-budget/token-budget.page";
import { loadTestEnv } from "../framework/env";

const aiMask =
  "value.ai.enabled,value.ai.provider,value.ai.endpoint,value.ai.api_key,value.ai.model";
let installed = false;
test.beforeAll(async () => {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const previous = (await env.api.getSetting("AI"))?.value as
    | {
        ai?: {
          enabled?: boolean;
          endpoint?: string;
          model?: string;
          apiKey?: string;
        };
      }
    | undefined;
  if (
    previous?.ai?.enabled ||
    previous?.ai?.endpoint ||
    previous?.ai?.apiKey ||
    previous?.ai?.model
  )
    throw new Error("Expected an unconfigured disposable AI provider");
  await env.api.upsertSetting(
    "AI",
    {
      ai: {
        enabled: true,
        provider: "OPEN_AI",
        endpoint: "http://127.0.0.1:1",
        model: "e2e-scripted",
        apiKey: "e2e-placeholder",
      },
    },
    aiMask,
  );
  installed = true;
});
test.afterAll(async () => {
  if (!installed) return;
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  await env.api.upsertSetting(
    "AI",
    {
      ai: {
        enabled: false,
        provider: "PROVIDER_UNSPECIFIED",
        endpoint: "",
        model: "",
        apiKey: "",
      },
    },
    aiMask,
  );
});

function latch() {
  let release!: () => void;
  const promise = new Promise<void>((resolve) => {
    release = resolve;
  });
  return { promise, release };
}

const windowOf = (page: Page) => page.locator("[data-agent-window]");
const editor = (page: Page) => windowOf(page).locator("textarea").last();
const button = (page: Page, name: string) =>
  windowOf(page).getByRole("button", { name, exact: true });
async function open(page: Page, url: string) {
  await page.goto(`${url}/landing`);
  if (!(await windowOf(page).isVisible()))
    await page.getByRole("button", { name: "Agent", exact: true }).click();
  await expect(windowOf(page)).toBeVisible();
}
async function selectChat(page: Page, title: string) {
  const row = windowOf(page)
    .locator("[data-agent-chat-row]")
    .filter({ hasText: title });
  if (!(await row.isVisible())) await button(page, "Chats").first().click();
  await row.click();
}
async function send(page: Page, text: string, command = "Send") {
  await editor(page).fill(text);
  await button(page, command).click();
}

function toolCall(id: string, name: string, args: object) {
  return {
    toolCalls: [{ id, name, arguments: JSON.stringify(args) }],
    stopReason: "TURN_COMPLETED",
  };
}

const matchesPost = (request: Request, path: string) =>
  request.method() === "POST" && new URL(request.url()).pathname === path;
const queryPath = "/bytebase.v1.SQLService/Query";
const updatePath = "/bytebase.v1.DatabaseService/UpdateDatabase";
const getDatabasePath = "/bytebase.v1.DatabaseService/GetDatabase";
const repeatRisk = "An earlier identical request may already have taken effect.";
const recovery = (page: Page) =>
  windowOf(page).getByRole("region", { name: "Request recovery" });
async function expectNoRecoveryActions(page: Page) {
  const region = recovery(page);
  await expect(region.getByRole("button")).toHaveCount(0);
  await expect(region.getByRole("link")).toHaveCount(0);
  await expect(region.getByRole("group")).toHaveCount(0);
  await expect(
    page.getByRole("alertdialog", { name: "Confirm external review" }),
  ).toHaveCount(0);
}

async function runReferencedSelect(
  page: Page,
  provider: DeterministicProvider,
  env: ReturnType<typeof loadTestEnv>,
) {
  const statement = "SELECT 1 AS value";
  const body = { name: env.database, statement, limit: 1 };
  const operationId = "bytebase.v1.SQLService.Query";
  const completed = "The SELECT completed.";
  provider.respond(toolCall("query-skill", "get_skill", { name: "query" }));
  provider.respond(toolCall("query-api", "search_api", { operationId }));
  provider.respond(toolCall("query-call", "call_api", { operationId, body }));
  provider.respond(answer(100, completed));

  const posts: unknown[] = [];
  page.on("request", (request) => {
    if (matchesPost(request, queryPath)) posts.push(request.postDataJSON());
  });

  await editor(page).fill(`Run ${statement} on @`);
  const picker = page.locator("[data-agent-resource-picker]");
  await picker.getByRole("button", { name: "Databases", exact: true }).click();
  await picker
    .getByLabel("Search databases", { exact: true })
    .fill(env.databaseId);
  await picker
    .locator(`[data-agent-resource-database="${env.database}"]`)
    .click();
  await button(page, "Send").click();
  await expect(
    windowOf(page).locator("[data-agent-message-mention]"),
  ).toContainText(env.databaseId);

  const approve = button(page, "Allow once");
  await expect(approve).toBeEnabled();
  await expect(
    windowOf(page).getByText("Request permission", { exact: true }),
  ).toBeVisible();
  const preview = windowOf(page).locator("pre").filter({
    hasText: `"statement": "${statement}"`,
  });
  await expect(preview).toContainText(`"name": "${env.database}"`);
  await expect(preview).toContainText('"limit": 1');
  expect(posts).toEqual([]);

  const responsePromise = page.waitForResponse((response) =>
    matchesPost(response.request(), queryPath),
  );
  await approve.click();
  const response = await responsePromise;
  expect(response.status()).toBe(200);
  const query = fromJsonString(QueryResponseSchema, await response.text());
  expect(query.results).toHaveLength(1);
  expect(query.results[0]).toMatchObject({
    columnNames: ["value"],
    error: "",
    rowsCount: 1n,
  });
  expect(query.results[0].rows).toHaveLength(1);
  expect(query.results[0].rows[0].values).toHaveLength(1);
  expect(String(query.results[0].rows[0].values[0].kind.value)).toBe("1");
  const result = windowOf(page).locator("[data-agent-query-result]");
  await expect(
    result.getByRole("columnheader", { name: "value", exact: true }),
  ).toBeVisible();
  await expect(result.getByTitle("1", { exact: true })).toHaveText("1");
  await expect(
    windowOf(page).getByText(completed, { exact: true }),
  ).toBeVisible();
  await expect(approve).toBeHidden();
  await expect(
    windowOf(page).getByText(/unavailable during API recovery/i),
  ).toHaveCount(0);
  expect(posts).toEqual([body]);
}

test(
  "a database-referenced SELECT returns real rows after exact approval",
  async ({ page, context }) => {
    const env = loadTestEnv();
    const provider = new DeterministicProvider();
    await provider.install(context);
    await open(page, env.baseURL);

    await runReferencedSelect(page, provider, env);

    await expect(recovery(page)).toBeHidden();
    await expectNoRecoveryActions(page);
    expect(provider.calls).toBe(4);
    expect(provider.unexpectedCalls).toBe(0);
  },
);

test("a diagnostic read with a reused tool ID survives reload and resume", async ({
  page,
  context,
}, info) => {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const original = (await env.api.getDatabase(env.database)).labels ?? {};
  const labels = { ...original, "e2e-recovery": "same-run-reuse" };
  const body = {
    database: { name: env.database, labels },
    updateMask: "labels",
  };
  const toolCallId = "reused-api-call";
  const prompt = "Label QA same-run tool ID reuse";
  const completed = "The saved diagnostic read survived resume.";
  const provider = new DeterministicProvider();
  await provider.install(context);
  provider.respond(
    toolCall(toolCallId, "call_api", {
      operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
      body,
    }),
  );
  provider.respond({
    ...toolCall(toolCallId, "call_api", {
      operationId: "bytebase.v1.DatabaseService.GetDatabase",
      body: { name: env.database },
    }),
    usage: { totalTokens: 1100, inputTokens: 1100, outputTokens: 0 },
  });
  provider.respond(answer(100, completed));
  let posts = 0;
  let forwards = 0;
  let diagnosticReads = 0;
  page.on("request", (request) => {
    if (
      matchesPost(request, getDatabasePath) &&
      request.headers()["content-type"]?.startsWith("application/json")
    ) {
      expect(request.postDataJSON()).toEqual({ name: env.database });
      diagnosticReads++;
    }
  });
  await context.route(
    /\/bytebase\.v1\.DatabaseService\/UpdateDatabase(?:\?|$)/,
    async (route) => {
      posts++;
      if (posts > 1) return route.abort();
      expect(route.request().method()).toBe("POST");
      expect(route.request().postDataJSON()).toEqual(body);
      forwards++;
      const upstream = await route.fetch({ maxRedirects: 0, maxRetries: 0 });
      expect(upstream.status()).toBe(200);
      await upstream.dispose();
      await route.abort("failed");
    },
  );
  const persistedResults = (ownedChatId?: string) =>
    page.evaluate(({ id, ownedChatId }) => {
      const key = Object.keys(localStorage).find((key) =>
        key.startsWith("bb-agent-state-v3."),
      );
      if (!key) return null;
      const state = JSON.parse(localStorage.getItem(key)!);
      const chat = state.chats.find(
        (chat: { id: string }) =>
          chat.id === (ownedChatId ?? state.currentChatId),
      );
      const messages = chat ? state.messagesByChatId[chat.id] ?? [] : [];
      return {
        chatId: chat?.id,
        runId: chat?.runId,
        results: messages
          .filter(
            (message: { role: string; toolCallId?: string }) =>
              message.role === "tool" && message.toolCallId === id,
          )
          .map(
            (message: {
              content: string;
              metadata?: { apiAttemptId?: string };
            }) => ({
              content: message.content,
              apiAttemptId: message.metadata?.apiAttemptId,
            }),
          ),
      };
    }, { id: toolCallId, ownedChatId });
  try {
    await open(page, env.baseURL);
    const budget = new TokenBudgetPage(page, env.baseURL, info);
    await budget.setDefaultLimit("1");
    await send(page, prompt);
    await expect(button(page, "Allow once")).toBeEnabled();
    await button(page, "Allow once").click();
    await budget.paused();
    await expect(
      windowOf(page).getByText("Outcome unknown", { exact: true }),
    ).toBeVisible();
    await expectNoRecoveryActions(page);
    await expect
      .poll(async () => (await persistedResults())?.results.length)
      .toBe(2);
    const before = (await persistedResults())!;
    expect(before.chatId).toBeTruthy();
    expect(before.results[0].apiAttemptId).toBeTruthy();
    expect(before.results[1].apiAttemptId).toBeUndefined();
    expect(before.results[1].content).toContain("same-run-reuse");
    expect((await env.api.getDatabase(env.database)).labels ?? {}).toEqual(
      labels,
    );
    expect(provider.calls).toBe(2);
    expect(diagnosticReads).toBe(1);
    expect(posts).toBe(1);
    expect(forwards).toBe(1);

    await page.reload();
    if (!(await windowOf(page).isVisible()))
      await page.getByRole("button", { name: "Agent", exact: true }).click();
    if (!(await budget.region.isVisible())) await selectChat(page, prompt);
    await budget.paused();
    expect(await persistedResults(before.chatId)).toEqual(before);
    await budget.increase("2", { resume: true });
    await expect(
      windowOf(page).getByText(completed, { exact: true }),
    ).toBeVisible();
    expect(provider.toolResults).toBe(2);
    expect(provider.calls).toBe(3);
    expect(provider.unexpectedCalls).toBe(0);
    expect((await persistedResults(before.chatId))?.results).toEqual(
      before.results,
    );
    expect(diagnosticReads).toBe(1);
    expect(posts).toBe(1);
    expect(forwards).toBe(1);
    await expect(button(page, "Allow once")).toBeHidden();
    await expectNoRecoveryActions(page);
    const path = info.outputPath("reused-tool-id-restored.png");
    await page.screenshot({ path });
    await info.attach("reused-tool-id-restored", {
      path,
      contentType: "image/png",
    });
  } finally {
    if (forwards) await env.api.updateDatabaseLabels(env.database, original);
    expect((await env.api.getDatabase(env.database)).labels ?? {}).toEqual(
      original,
    );
  }
});

test("a later write with a reused tool ID never borrows an earlier acknowledgment", async ({
  page,
  context,
}, info) => {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const original = (await env.api.getDatabase(env.database)).labels ?? {};
  const firstLabels = { ...original, "e2e-recovery": "first-acknowledged" };
  const secondLabels = { ...original, "e2e-recovery": "second-uncertain" };
  const bodies = [firstLabels, secondLabels].map((labels) => ({
    database: { name: env.database, labels },
    updateMask: "labels",
  }));
  const toolCallId = "reused-sequential-write";
  const provider = new DeterministicProvider();
  await provider.install(context);
  for (const body of bodies)
    provider.respond(
      toolCall(toolCallId, "call_api", {
        operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
        body,
      }),
    );
  const secondCommitted = latch();
  const secondDelivery = latch();
  let posts = 0;
  let forwards = 0;
  await context.route(
    /\/bytebase\.v1\.DatabaseService\/UpdateDatabase(?:\?|$)/,
    async (route) => {
      const index = posts++;
      if (index > 1) return route.abort();
      expect(route.request().method()).toBe("POST");
      expect(route.request().postDataJSON()).toEqual(bodies[index]);
      forwards++;
      const upstream = await route.fetch({ maxRedirects: 0, maxRetries: 0 });
      expect(upstream.status()).toBe(200);
      if (index === 0) {
        try {
          await route.fulfill({ response: upstream });
        } finally {
          await upstream.dispose();
        }
        return;
      }
      await upstream.dispose();
      secondCommitted.release();
      await secondDelivery.promise;
      await route.abort("failed");
    },
  );
  const calls = windowOf(page).getByRole("button", { name: /^call_api\b/ });
  try {
    await open(page, env.baseURL);
    await send(page, "Label QA acknowledge the first write, then change it again");
    const approve = button(page, "Allow once");
    await expect(approve).toBeEnabled();
    await expect(
      windowOf(page).locator("pre").filter({
        hasText: '"e2e-recovery": "first-acknowledged"',
      }),
    ).toBeVisible();
    expect(posts).toBe(0);
    await approve.click();
    await expect(calls.first()).toContainText("RPC acknowledged");
    await expect(approve).toBeEnabled();
    await expect(
      windowOf(page).locator("pre").filter({
        hasText: '"e2e-recovery": "second-uncertain"',
      }),
    ).toBeVisible();
    expect(posts).toBe(1);
    expect(forwards).toBe(1);
    expect((await env.api.getDatabase(env.database)).labels ?? {}).toEqual(
      firstLabels,
    );
    await approve.click();
    await secondCommitted.promise;
    expect((await env.api.getDatabase(env.database)).labels ?? {}).toEqual(
      secondLabels,
    );
    const before = await page.evaluate(() => {
      const key = Object.keys(localStorage).find((key) =>
        key.startsWith("bb-agent-state-v3."),
      );
      const state = JSON.parse(localStorage.getItem(key!)!);
      const chat = state.chats.find(
        (chat: { id: string }) => chat.id === state.currentChatId,
      );
      const messages = state.messagesByChatId[chat.id] ?? [];
      return {
        chatId: chat.id as string,
        runId: chat.runId as string,
        attempts: chat.apiAttempts.map(
          (attempt: {
            id: string;
            rootUserMessageId: string;
            toolCallId: string;
            phase: string;
            execution?: { result: string };
          }) => ({
            id: attempt.id,
            rootUserMessageId: attempt.rootUserMessageId,
            toolCallId: attempt.toolCallId,
            phase: attempt.phase,
            result: attempt.execution?.result,
          }),
        ),
        userMessages: messages.filter(
          (message: { role: string }) => message.role === "user",
        ).length,
      };
    });
    expect(before.chatId).toBeTruthy();
    expect(before.runId).toBeTruthy();
    expect(before.userMessages).toBe(1);
    expect(before.attempts).toHaveLength(2);
    expect(before.attempts[0].toolCallId).toBe(toolCallId);
    expect(before.attempts[1].toolCallId).toBe(toolCallId);
    expect(before.attempts[0].id).not.toBe(before.attempts[1].id);
    expect(before.attempts[0].rootUserMessageId).toBe(
      before.attempts[1].rootUserMessageId,
    );
    expect(before.attempts[0]).toMatchObject({
      phase: "settled",
      result: "succeeded",
    });
    expect(before.attempts[1].phase).toBe("dispatching");
    expect(posts).toBe(2);
    expect(forwards).toBe(2);

    await page.reload();
    secondDelivery.release();
    if (!(await windowOf(page).isVisible()))
      await page.getByRole("button", { name: "Agent", exact: true }).click();
    const row = windowOf(page).locator(
      `[data-agent-chat-row="${before.chatId}"]`,
    );
    if (!(await row.isVisible())) await button(page, "Chats").first().click();
    await row.click();
    await expect(calls).toHaveCount(2);
    await expect(calls.nth(0)).toContainText("RPC acknowledged");
    await expect(calls.nth(1)).toContainText("Outcome unknown");
    await expect(calls.nth(1)).not.toContainText("RPC acknowledged");
    await expect(approve).toBeHidden();
    await expectNoRecoveryActions(page);
    await expect(recovery(page)).toBeHidden();
    expect(provider.calls).toBe(2);
    expect(provider.unexpectedCalls).toBe(0);
    expect(posts).toBe(2);
    expect(forwards).toBe(2);
    for (const [device, width, height] of [
      ["desktop", 1440, 960],
      ["mobile", 390, 844],
    ] as const) {
      await page.setViewportSize({ width, height });
      await expect(calls.nth(0)).toBeInViewport();
      await expect(calls.nth(1)).toBeInViewport();
      const path = info.outputPath(`sequential-write-outcomes-${device}.png`);
      await page.screenshot({ path });
      await info.attach(device, { path, contentType: "image/png" });
    }
  } finally {
    secondDelivery.release();
    if (forwards) await env.api.updateDatabaseLabels(env.database, original);
    expect((await env.api.getDatabase(env.database)).labels ?? {}).toEqual(
      original,
    );
  }
});

for (const fault of [
  "commit-drop",
  "reload-before-settlement",
  "protected-401",
] as const) {
  test(
    fault === "commit-drop"
      ? "a lost write response allows diagnosis, queued follow-up, and a new SELECT"
      : fault === "reload-before-settlement"
        ? "reload never replays the old write; a fresh repeat still needs its own approval"
        : "a protected 401 never reposts the write",
    async ({ page, context }, info) => {
      const env = loadTestEnv();
      await env.api.login(env.adminEmail, env.adminPassword);
      const original = (await env.api.getDatabase(env.database)).labels ?? {};
      const labels = { ...original, "e2e-recovery": fault };
      const body = {
        database: { name: env.database, labels },
        updateMask: "labels",
      };
      const operationId = "bytebase.v1.DatabaseService.UpdateDatabase";
      const response = (id: string) =>
        toolCall(id, "call_api", { operationId, body });
      const provider = new DeterministicProvider();
      await provider.install(context);
      const startApproval = provider.hold(response("initial-write"));
      const diagnosis =
        fault === "protected-401"
          ? "The RPC failed; changes may have occurred. Current labels were read without replaying the request."
          : "The request outcome is unknown. Current labels were read without replaying the request.";
      if (fault !== "reload-before-settlement") {
        provider.respond(
          toolCall("diagnose-current-state", "call_api", {
            operationId: "bytebase.v1.DatabaseService.GetDatabase",
            body: { name: env.database },
          }),
        );
        provider.respond(answer(100, diagnosis));
        provider.respond(answer(101, "The queued follow-up completed."));
      }
      const sent = latch();
      const delivery = latch();
      let posts = 0;
      let forwards = 0;
      let upstreamStatus: number | undefined;
      let allowFreshRepeat = false;
      let diagnosticReads = 0;
      page.on("request", (request) => {
        if (
          matchesPost(request, getDatabasePath) &&
          request.headers()["content-type"]?.startsWith("application/json")
        ) {
          expect(request.postDataJSON()).toEqual({ name: env.database });
          diagnosticReads++;
        }
      });
      await context.route(
        /\/bytebase\.v1\.DatabaseService\/UpdateDatabase(?:\?|$)/,
        async (route) => {
          ++posts;
          const repeat = posts === 2 && allowFreshRepeat;
          if (posts > 1 && !repeat) return route.abort();
          expect(route.request().method()).toBe("POST");
          expect(route.request().postDataJSON()).toEqual(body);
          if (fault === "protected-401") {
            await route.fulfill({
              status: 401,
              contentType: "application/json",
              body: JSON.stringify({
                code: "unauthenticated",
                message: "Controlled session fault",
              }),
            });
            sent.release();
            return;
          }
          ++forwards;
          const upstream = await route.fetch({
            maxRedirects: 0,
            maxRetries: 0,
          });
          if (repeat) {
            try {
              await route.fulfill({ response: upstream });
            } finally {
              await upstream.dispose();
            }
            return;
          }
          upstreamStatus = upstream.status();
          await upstream.dispose();
          sent.release();
          if (fault === "reload-before-settlement") await delivery.promise;
          await route.abort("failed");
        },
      );
      const prompt = `Label QA ${fault}`;
      const queued = "Summarize this change later";
      const draft = "Unsent recovery notes";
      try {
        await open(page, env.baseURL);
        await send(page, prompt);
        await expect.poll(() => provider.calls).toBe(1);
        await send(page, queued, "Queue");
        await editor(page).fill(draft);
        startApproval();
        const approve = button(page, "Allow once");
        await expect(approve).toBeEnabled();
        expect(posts).toBe(0);
        await approve.click();
        await sent.promise;
        expect(upstreamStatus).toBe(
          fault === "protected-401" ? undefined : 200,
        );
        expect((await env.api.getDatabase(env.database)).labels ?? {}).toEqual(
          fault === "protected-401" ? original : labels,
        );

        if (fault === "reload-before-settlement") {
          await expect(
            windowOf(page).getByText("call_api", { exact: true }),
          ).toBeVisible();
          await expect(recovery(page)).toBeHidden();
          await expectNoRecoveryActions(page);
          await page.reload();
          delivery.release();
          if (!(await windowOf(page).isVisible()))
            await page
              .getByRole("button", { name: "Agent", exact: true })
              .click();
          if (
            !(await windowOf(page)
              .getByText("Outcome unknown", { exact: true })
              .isVisible())
          )
            await selectChat(page, prompt);
        }
        const region = recovery(page);
        const outcomeLabel =
          fault === "protected-401"
            ? "RPC failed; changes may have occurred"
            : "Outcome unknown";
        await expectNoRecoveryActions(page);
        await expect(approve).toBeHidden();
        await expect(editor(page)).toHaveValue(draft);
        expect(posts).toBe(1);

        if (fault !== "reload-before-settlement") {
          await expect(
            windowOf(page).getByText(diagnosis, { exact: true }),
          ).toBeVisible();
          await expect(
            windowOf(page).getByText("The queued follow-up completed.", {
              exact: true,
            }),
          ).toBeVisible();
          await expect(button(page, "Resume queue")).toBeHidden();
          await expect(editor(page)).toHaveValue(draft);
          await expect(region).toBeHidden();
          await expect(
            windowOf(page).getByText(outcomeLabel, { exact: true }),
          ).toBeVisible();
          await expect(button(page, "Send")).toBeEnabled();
          expect(diagnosticReads).toBe(1);
          expect(provider.calls).toBe(4);
        }

        if (fault === "commit-drop") {

          await runReferencedSelect(page, provider, env);
          await expect(region).toBeHidden();
          const outcome = windowOf(page).getByText("Outcome unknown", {
            exact: true,
          });
          await expect(outcome).toBeVisible();
          await expectNoRecoveryActions(page);
          expect(posts).toBe(1);
          expect(provider.calls).toBe(8);

          const north = (await windowOf(page)
            .locator('[data-agent-window-resize-zone="n"]')
            .boundingBox())!;
          await page.mouse.move(
            north.x + north.width / 2,
            north.y + north.height / 2,
          );
          await page.mouse.down();
          await page.mouse.move(north.x + north.width / 2, 40, { steps: 5 });
          await page.mouse.up();
          for (const [device, width, height] of [
            ["desktop", 1440, 960],
            ["mobile", 390, 844],
          ] as const) {
            await page.setViewportSize({ width, height });
            if (device === "mobile")
              await expect(
                windowOf(page)
                  .locator("[data-agent-window-header]")
                  .getByRole("button", { name: "Chats", exact: true }),
              ).toBeVisible();
            if (!(await outcome.isVisible())) await selectChat(page, prompt);
            await outcome.scrollIntoViewIfNeeded();
            await expect(outcome).toBeInViewport();
            await expect(region).toBeHidden();
            await expectNoRecoveryActions(page);
            const box = (await outcome.boundingBox())!;
            expect(box.x).toBeGreaterThanOrEqual(0);
            expect(box.x + box.width).toBeLessThanOrEqual(width);
            const path = info.outputPath(`inline-api-outcome-${device}.png`);
            await page.screenshot({ path });
            await info.attach(device, { path, contentType: "image/png" });
          }
          expect(provider.calls).toBe(8);
          await expectNoRecoveryActions(page);
          await expect(region).toBeHidden();
        }
        if (fault === "reload-before-settlement") {
          await expect(
            windowOf(page).getByText("Outcome unknown", { exact: true }),
          ).toBeVisible();
          await expect(region).toBeHidden();
          await expect(button(page, "Resume queue")).toBeEnabled();
          await expect(
            windowOf(page).getByText(queued, { exact: true }),
          ).toBeVisible();
          expect(provider.calls).toBe(1);
          expect(diagnosticReads).toBe(0);
          const peer = await context.newPage();
          try {
            await open(peer, env.baseURL);
            if (
              !(await windowOf(peer)
                .getByText("Outcome unknown", { exact: true })
                .isVisible())
            )
              await selectChat(peer, prompt);
            await expect(
              windowOf(peer).getByText("Outcome unknown", { exact: true }),
            ).toBeVisible();
            await expect(recovery(peer)).toBeHidden();
            await expectNoRecoveryActions(peer);
            await expect(button(peer, "Allow once")).toBeHidden();
            expect(posts).toBe(1);
            expect(forwards).toBe(1);

            const interveningLabels = {
              ...original,
              "e2e-recovery": "later-edit",
            };
            await env.api.updateDatabaseLabels(env.database, interveningLabels);
            if (!(await button(peer, "New Chat").first().isVisible()))
              await button(peer, "Chats").first().click();
            await button(peer, "New Chat").first().click();
            provider.respond(response("peer-matching-write"));
            provider.respond(
              answer(100, "The newly approved write completed."),
            );
            await send(peer, "Request the same label change again");
            const repeatApproval = button(peer, "Allow once");
            await expect(repeatApproval).toBeEnabled();
            await expect(
              windowOf(peer).getByText("Request permission", { exact: true }),
            ).toBeVisible();
            await expect(windowOf(peer)).toContainText(repeatRisk);
            expect(posts).toBe(1);
            expect(forwards).toBe(1);
            expect(
              (await env.api.getDatabase(env.database)).labels ?? {},
            ).toEqual(interveningLabels);

            const repeatedResponse = peer.waitForResponse((response) =>
              matchesPost(response.request(), updatePath),
            );
            allowFreshRepeat = true;
            await repeatApproval.click();
            expect((await repeatedResponse).status()).toBe(200);
            await expect(
              windowOf(peer).getByText("The newly approved write completed.", {
                exact: true,
              }),
            ).toBeVisible();
            await expect(repeatApproval).toBeHidden();
            expect(
              (await env.api.getDatabase(env.database)).labels ?? {},
            ).toEqual(labels);
            await expect(
              windowOf(page).getByText("Outcome unknown", { exact: true }),
            ).toBeVisible();
            await expect(region).toBeHidden();
          } finally {
            await peer.close();
          }
        }
        if (fault === "reload-before-settlement") {
          await expect(button(page, "Resume queue")).toBeEnabled();
          await expect(button(page, "Resend message")).toBeEnabled();
        }
        expect(posts).toBe(fault === "reload-before-settlement" ? 2 : 1);
        expect(forwards).toBe(
          fault === "protected-401" ? 0 : fault === "commit-drop" ? 1 : 2,
        );
        expect(provider.unexpectedCalls).toBe(0);
        expect(provider.calls).toBe(
          fault === "commit-drop"
            ? 8
            : fault === "reload-before-settlement"
              ? 3
              : 4,
        );
        await info.attach("recovery-counts", {
          contentType: "application/json",
          body: JSON.stringify({
            posts,
            forwards,
            upstreamStatus,
            modelCalls: provider.calls,
            diagnosticReads,
            fixtureLabel:
              fault === "protected-401" ? null : labels["e2e-recovery"],
          }),
        });
      } finally {
        startApproval();
        delivery.release();
        if (forwards)
          await env.api.updateDatabaseLabels(env.database, original);
        expect((await env.api.getDatabase(env.database)).labels ?? {}).toEqual(
          original,
        );
      }
    },
  );
}
