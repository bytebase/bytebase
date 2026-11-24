import { writeFileSync } from "node:fs";
import type { Locator, Page, TestInfo } from "@playwright/test";
import type {
  AgentChat,
  AgentMessage,
  AgentMessageMetadata,
} from "../../../src/modules/agent/logic/types";
import {
  answer as completedAnswer,
  expect,
  test,
} from "../agent-token-budget/token-budget.fixture";
import { loadTestEnv } from "../framework/env";

interface SavedAnswer {
  title: string;
  content?: string;
  metadata: AgentMessageMetadata;
  toolCalls?: AgentMessage["toolCalls"];
}

function captureScreenshots(testInfo: TestInfo, width: number) {
  const screenshots: { file: string; label: string }[] = [];
  return async (surface: Locator, file: string, label: string) => {
    await surface.screenshot({ path: testInfo.outputPath(file) });
    screenshots.push({ file, label });
    writeFileSync(testInfo.outputPath("screenshots.json"), JSON.stringify({
      title: testInfo.title,
      width,
      screenshots,
    }, null, 2));
  };
}

async function openFixture(
  page: Page,
  id: string,
  answers: SavedAnswer[],
  extra: Partial<AgentChat> = {}
) {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const { workspace } = await env.api.getActuatorInfo();
  const workspaceId = workspace.split("/").at(-1)!;
  await page.addInitScript(
    ({ key, workspace, project, id, answers, extra }) => {
      const marker = `agent-draft-layout-seeded.${id}`;
      if (sessionStorage.getItem(marker)) return;
      const now = Date.now();
      localStorage.setItem(key, JSON.stringify({
        currentChatId: id,
        chats: [{
          id,
          title: "Draft action layout",
          context: project,
          workspaceName: workspace,
          createdTs: now,
          updatedTs: now,
          status: "idle",
          totalTokensUsed: 0,
          archived: false,
          ...extra,
        }],
        messagesByChatId: {
          [id]: [
            { id: `${id}-user`, chatId: id, role: "user", content: "Review these unsaved proposals.", createdTs: now },
            ...answers.map((item, index) => ({
              id: `${id}-answer-${index}`,
              chatId: id,
              role: "assistant",
              content: item.content ?? item.title,
              metadata: item.metadata,
              toolCalls: item.toolCalls,
              createdTs: now + index + 1,
            })),
          ],
        },
        pendingAskByChatId: {},
        draftsByChatId: {},
        queuedUserMessagesByChatId: {},
      }));
      sessionStorage.setItem(marker, "1");
    },
    {
      key: `bb-agent-state-v3.${workspaceId.length}:${workspaceId}.${env.adminEmail}`,
      workspace,
      project: env.project,
      id,
      answers,
      extra,
    }
  );
  await page.goto(`${env.baseURL}/landing`);
  await page.getByRole("button", { name: "Agent", exact: true }).click();
  const agent = page.locator("[data-agent-window]");
  await expect(agent).toBeVisible();
  const row = agent.getByRole("button", { name: "Draft action layout", exact: true });
  if (!(await row.isVisible())) {
    await agent.getByRole("button", { name: "Chats", exact: true }).first().click();
  }
  await row.click();
  return { env, agent };
}

async function expectContained(element: Locator, container: Locator) {
  await expect(container).toBeVisible();
  await expect(element).toBeVisible();
  await expect.poll(() => element.boundingBox()).not.toBeNull();
  const bounds = await element.boundingBox();
  const parent = await container.boundingBox();
  expect(bounds).not.toBeNull();
  expect(parent).not.toBeNull();
  expect(bounds!.x).toBeGreaterThanOrEqual(parent!.x - 1);
  expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(parent!.x + parent!.width + 1);
  expect(await element.evaluate((node) => node.scrollWidth - node.clientWidth)).toBeLessThanOrEqual(1);
}

async function expectFinalActionRow(action: Locator, answer: Locator) {
  const group = action.locator("xpath=ancestor::*[@data-agent-action-group][1]");
  await expect(group).toBeVisible();
  await expectContained(group, answer);
  await expectContained(action, group);
  const placement = await group.evaluate((node) => {
    const bounds = node.getBoundingClientRect();
    const preceding = node.previousElementSibling?.getBoundingClientRect();
    return {
      final: node.parentElement?.lastElementChild === node,
      top: bounds.top,
      precedingBottom: preceding?.bottom,
      right: bounds.right,
    };
  });
  expect(placement.final).toBe(true);
  if (placement.precedingBottom !== undefined) {
    expect(placement.top).toBeGreaterThanOrEqual(placement.precedingBottom - 1);
  }
  const bounds = (await action.boundingBox())!;
  expect(Math.abs(bounds.x + bounds.width - placement.right)).toBeLessThanOrEqual(1);
}

function effectfulCalls(page: Page) {
  const calls: string[] = [];
  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    if (request.method() === "POST" && /\/(?:Create|Update|Delete|BatchUpdate|Query)[^/]*$/.test(path)) {
      calls.push(path);
    }
  });
  return calls;
}

const savedDrafts: SavedAnswer[] = [
  {
    title: "Unavailable rule draft",
    metadata: { actions: [{ kind: "draft-masking-rule", draftId: "rule-inactive" }] },
  },
  {
    title: "Opened rule draft",
    metadata: {
      actions: [{ kind: "draft-masking-rule", draftId: "rule-opened" }],
      maskingRuleDraftHandoff: { draftId: "rule-opened", outcome: "opened" },
    },
  },
  {
    title: "Applied rule draft",
    metadata: {
      actions: [{ kind: "draft-masking-rule", draftId: "rule-applied" }],
      maskingRuleDraftHandoff: { draftId: "rule-applied", outcome: "applied" },
    },
  },
  {
    title: "Unavailable exemption draft",
    metadata: { actions: [{ kind: "draft-masking-exemption", draftId: "exemption-inactive" }] },
  },
  {
    title: "Opened exemption draft",
    metadata: {
      actions: [{ kind: "draft-masking-exemption", draftId: "exemption-opened" }],
      maskingExemptionDraftHandoff: { draftId: "exemption-opened", outcome: "opened" },
    },
  },
  {
    title: "Applied exemption draft",
    metadata: {
      actions: [{ kind: "draft-masking-exemption", draftId: "exemption-applied" }],
      maskingExemptionDraftHandoff: { draftId: "exemption-applied", outcome: "applied" },
    },
  },
  {
    title: "Created exemption receipt",
    metadata: {
      actions: [{ kind: "draft-masking-exemption", draftId: "exemption-created" }],
      maskingExemptionDraftHandoff: { draftId: "exemption-created", outcome: "opened", created: true },
    },
  },
  {
    title: "Unavailable semantic type draft",
    metadata: { actions: [{ kind: "draft-semantic-type", draftId: "semantic-inactive" }] },
  },
  {
    title: "Opened semantic type draft",
    metadata: {
      actions: [{ kind: "draft-semantic-type", draftId: "semantic-opened" }],
      semanticTypeDraftHandoff: { draftId: "semantic-opened", outcome: "opened" },
    },
  },
  {
    title: "Created semantic type receipt",
    metadata: {
      actions: [{ kind: "draft-semantic-type", draftId: "semantic-created" }],
      semanticTypeDraftHandoff: { draftId: "semantic-created", outcome: "opened", created: true },
    },
  },
];

for (const width of [320, 390]) {
  test(`saved draft actions remain final and right-aligned at ${width}px`, async ({ page, provider }, testInfo) => {
    const capture = captureScreenshots(testInfo, width);
    await page.setViewportSize({ width, height: 900 });
    const calls = effectfulCalls(page);
    const { agent } = await openFixture(page, `saved-drafts-${width}`, savedDrafts);
    for (const [index, item] of savedDrafts.entries()) {
      const answer = agent.locator("[data-agent-answer]").filter({ hasText: item.title });
      const action = answer.locator("[data-agent-primary-action]");
      await action.scrollIntoViewIfNeeded();
      await expect(action).toBeDisabled();
      await expectFinalActionRow(action, answer);
      await capture(answer, `saved-draft-${index}-${width}.png`, item.title);
    }
    expect(provider.calls).toBe(0);
    expect(calls).toEqual([]);
  });

  test(`response handoffs and draft previews use final action rows at ${width}px`, async ({ page, provider }, testInfo) => {
    const capture = captureScreenshots(testInfo, width);
    await page.setViewportSize({ width, height: 900 });
    const env = loadTestEnv();
    const answers: SavedAnswer[] = [
      { title: "Open the review queue", metadata: { handoff: { kind: "review-database-changes", project: env.project } } },
      {
        title: "Review this Plan proposal",
        content: "Review this Plan proposal.\n\n```sql\nCREATE TABLE draft_layout_test (id INT PRIMARY KEY);\n```",
        metadata: { actions: [{ kind: "draft-plan", project: env.project, database: env.database }] },
      },
      {
        title: "Review this unsent Issue comment",
        metadata: { actions: [{ kind: "draft-issue-comment", issue: `${env.project}/issues/42`, comment: "Review the exact access scope before approving this unsent proposal." }] },
      },
      {
        title: "Review this access request",
        metadata: { accessRequestDraft: { project: env.project, sourceDatabase: env.database, targets: [env.database], statement: "SELECT 1" } },
      },
    ];
    const calls = effectfulCalls(page);
    const { agent } = await openFixture(page, `response-drafts-${width}`, answers);
    for (const [index, item] of answers.entries()) {
      const answer = agent.locator("[data-agent-answer]").filter({ hasText: item.title });
      const action = answer.locator("[data-agent-primary-action]");
      await action.scrollIntoViewIfNeeded();
      await expectFinalActionRow(action, answer);
      if (index === 2) {
        await answer.locator("summary").click();
        await expect(answer.getByText("Review the exact access scope before approving this unsent proposal.", { exact: true })).toBeVisible();
        await expectFinalActionRow(action, answer);
      }
      await capture(answer, `response-draft-${index}-${width}.png`, item.title);
    }
    expect(provider.calls).toBe(0);
    expect(calls).toEqual([]);
  });

  test(`a restored SQL-fix proposal stays unavailable with final review commands at ${width}px`, async ({ page, provider }, testInfo) => {
    const capture = captureScreenshots(testInfo, width);
    await page.setViewportSize({ width, height: 900 });
    const env = loadTestEnv();
    const calls = effectfulCalls(page);
    const { agent } = await openFixture(page, `sql-fix-${width}`, [{
      title: "Review this saved SQL fix",
      metadata: { actions: [{ kind: "review-sql-fix", draftId: "missing-source" }] },
    }], {
      capability: { kind: "diagnostic", resource: { kind: "plan", name: `${env.project}/plans/42`, project: env.project, label: "Plan 42" } },
    });
    await agent.locator("[data-agent-action-review-sql-fix]").click();
    const dialog = page.getByRole("dialog", { name: "Review SQL fix", exact: true });
    await expect(dialog).toBeVisible();
    const apply = dialog.getByRole("button", { name: "Apply to draft", exact: true });
    await expect(apply).toBeDisabled();
    await expectFinalActionRow(apply, dialog);
    await expectContained(dialog, page.locator("body"));
    await capture(dialog, `sql-fix-unavailable-${width}.png`, "Unavailable restored SQL-fix review");
    await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
    expect(provider.calls).toBe(0);
    expect(calls).toEqual([]);
  });

  test(`runtime-prepared masking drafts retain final review actions at ${width}px`, async ({ page, provider }, testInfo) => {
    const capture = captureScreenshots(testInfo, width);
    await page.setViewportSize({ width, height: 900 });
    const { env, agent } = await openFixture(page, `runtime-drafts-${width}`, [{ title: "No draft has been prepared yet.", metadata: {} }]);
    const calls = effectfulCalls(page);
    const scenarios = [
      {
        section: "semantic-type",
        tool: "prepare_semantic_type_draft",
        args: { title: "Layout test full masking", description: "Synthetic masking proposal for visual verification.", algorithm: { kind: "full-mask", substitution: "[hidden]" } },
        selector: "[data-agent-action-draft-semantic-type]",
      },
      {
        section: "rule",
        tool: "prepare_masking_rule_draft",
        args: { title: "Layout test masking rule", condition: 'resource.column_name == "email"', semanticType: "bb.default", scope: { kind: "workspace" } },
        selector: "[data-agent-action-draft-masking-rule]",
      },
      {
        section: "exemption",
        tool: "prepare_masking_exemption_draft",
        args: { project: env.project, scope: { kind: "project" }, members: ["self"], condition: 'resource.column_name == "email"', description: "Synthetic temporary investigation", expirationTimestamp: new Date(Date.now() + 3_600_000).toISOString() },
        selector: "[data-agent-action-draft-masking-exemption]",
      },
    ];
    for (const [index, scenario] of scenarios.entries()) {
      const title = `Prepared ${scenario.section} proposal`;
      for (const [suffix, name, args] of [
        ["skill", "get_skill", { name: "data-masking", section: scenario.section }],
        ["draft", scenario.tool, scenario.args],
      ] as const) {
        provider.respond({ toolCalls: [{ id: `draft-layout-${index}-${suffix}`, name, arguments: JSON.stringify(args) }], stopReason: "TURN_COMPLETED" });
      }
      provider.respond(completedAnswer(10, title));
      await agent.locator("textarea").last().fill(`Prepare an unsaved ${scenario.section} proposal using only the confirmed fixture scope.`);
      await agent.getByRole("button", { name: "Send", exact: true }).click();
      const answer = agent.locator("[data-agent-answer]").filter({ hasText: title });
      const action = answer.locator(scenario.selector);
      await expect(action).toBeEnabled();
      await expectFinalActionRow(action, answer);
      await capture(answer, `runtime-draft-${scenario.section}-${width}.png`, `Active ${scenario.section} draft`);
    }
    expect(provider.calls).toBe(9);
    expect(calls).toEqual([]);
  });

  test(`unknown API outcomes remain passive at ${width}px`, async ({ page, provider }, testInfo) => {
    const capture = captureScreenshots(testInfo, width);
    await page.setViewportSize({ width, height: 900 });
    const id = `unknown-api-${width}`;
    const operationId = "bytebase.v1.DatabaseService.UpdateDatabase";
    const calls = effectfulCalls(page);
    const { agent } = await openFixture(page, id, [{
      title: "A prior API attempt has no recorded result.",
      metadata: {},
    }], {
      apiAttempts: [{
        version: 1,
        id: "orphan-attempt",
        logicalActionId: "orphan-action",
        ordinal: 1,
        scopeKey: "*",
        userName: "users/demo@example.com",
        chatId: id,
        runId: "orphan-run",
        rootUserMessageId: `${id}-user`,
        toolCallId: "orphan-call",
        operationId,
        path: "/bytebase.v1.DatabaseService/UpdateDatabase",
        fingerprint: "a".repeat(64),
        startedAt: Date.now() - 60_000,
        phase: "dispatching",
      }],
    });
    const recovery = agent.getByRole("region", { name: "Request recovery", exact: true });
    await expect(recovery).toContainText("Outcome unknown");
    await expect(recovery.getByRole("button")).toHaveCount(0);
    await expect(recovery.getByRole("link")).toHaveCount(0);
    await expectContained(recovery, agent);
    await capture(recovery, `api-recovery-passive-${width}.png`, "Passive request recovery");
    expect(provider.calls).toBe(0);
    expect(calls).toEqual([]);
  });

  test(`an inline missing API result stays passive at ${width}px`, async ({ page, provider }, testInfo) => {
    const capture = captureScreenshots(testInfo, width);
    await page.setViewportSize({ width, height: 900 });
    const env = loadTestEnv();
    const calls = effectfulCalls(page);
    const { agent } = await openFixture(page, `inline-api-${width}`, [{
      title: "The earlier request has no recorded result.",
      metadata: {},
      toolCalls: [{
        id: "missing-write-result",
        name: "call_api",
        arguments: JSON.stringify({
          operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
          body: { database: { name: env.database, labels: { team: "example" } }, updateMask: "labels" },
        }),
      }],
    }], { interrupted: true, interruptionReason: "unexpected" });
    const card = agent.getByRole("button").filter({ hasText: "call_api" });
    await expect(card).toContainText("Outcome unknown");
    await card.click();
    const tool = card.locator("..");
    await expect(tool.getByRole("button")).toHaveCount(1);
    await expect(tool.getByRole("link")).toHaveCount(0);
    await expectContained(tool, agent);
    await capture(tool, `api-inline-passive-${width}.png`, "Passive inline missing-result outcome");
    expect(provider.calls).toBe(0);
    expect(calls).toEqual([]);
  });
}
