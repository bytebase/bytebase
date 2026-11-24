import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { basename } from "node:path";
import { create, fromBinary, toBinary } from "@bufbuild/protobuf";
import type { Locator, Page, TestInfo } from "@playwright/test";
import type {
  AgentChat,
  AgentMessage,
  AgentPendingAsk,
} from "../../../src/modules/agent/logic/types";
import { ActuatorInfoSchema } from "../../../src/types/proto-es/v1/actuator_service_pb";
import {
  GetSettingRequestSchema,
  SettingSchema,
  SettingValueSchema,
} from "../../../src/types/proto-es/v1/setting_service_pb";
import {
  answer,
  expect,
  test,
} from "../agent-token-budget/token-budget.fixture";
import { loadTestEnv } from "../framework/env";
import { loginBrowserContext } from "../framework/sign-in";

const chatId = "agent-chat-layout-e2e";
const chatTitle = "Chat layout regression";
const runId = "layout-run";
const widths = [320, 390, 1440];
const userContent = JSON.stringify(
  {
    database: `instances/test-instance/databases/${"orders_".repeat(100)}`,
    fingerprint: "x".repeat(1200),
  },
  null,
  2
);
const assistantContent = `Review result: ${"longtoken".repeat(200)}.`;
const sql = `SELECT '${"value_".repeat(400)}' AS value;`;
const confirmLabel =
  "Prepare the unsaved database access request for review before submitting";
const cancelLabel =
  "KeepTheCurrentDatabaseAccessUnchangedAndDoNotPrepareAnyRequest";
const confirmationPrompt = "Prepare this unsaved draft?";
const connectionError = "Connection failed";
const storageErrors = {
  "storage-full": "QuotaExceededError: Browser storage is full",
  "storage-unavailable": "SecurityError: Browser storage is unavailable",
};

type Scenario =
  | "messages"
  | "storage-full"
  | "storage-unavailable"
  | "error-retry"
  | "error-no-retry"
  | "interrupted-unexpected"
  | "interrupted-user"
  | "interrupted-user-recoverable"
  | "confirmation-default"
  | "confirmation-long"
  | "confirmation-two-actions"
  | "permission-expired"
  | "choose"
  | "choose-custom"
  | "input"
  | "ai-configuration"
  | "ai-contextual-start"
  | "token-warning"
  | "token-reached"
  | "token-exhausted"
  | "token-resumable"
  | "token-multi-action"
  | "follow-up-queue";

function layoutState(scenario: Scenario, workspace: string, project: string) {
  if (scenario === "ai-contextual-start") {
    return {
      currentChatId: null,
      chats: [],
      messagesByChatId: {},
      pendingAskByChatId: {},
      draftsByChatId: {},
      queuedUserMessagesByChatId: {},
    };
  }
  const now = Date.now();
  const chat: AgentChat = {
    id: chatId,
    title: chatTitle,
    context: project,
    workspaceName: workspace,
    createdTs: now,
    updatedTs: now,
    status: "idle",
    totalTokensUsed: 0,
    archived: false,
  };
  let messages: AgentMessage[] = [
    { role: "user", content: "Count databases" },
    { role: "assistant", content: "Checking databases." },
  ].map((message, index) => ({
    ...message,
    role: message.role as AgentMessage["role"],
    id: `${chatId}-${index}`,
    chatId,
    createdTs: now + index,
  }));
  let pendingAsk: AgentPendingAsk | undefined;
  if (scenario === "messages") {
    messages = [
      { role: "user", content: userContent },
      { role: "assistant", content: assistantContent },
      {
        role: "assistant",
        content: `SQL statement:\n\n\`\`\`sql\n${sql}\n\`\`\``,
      },
    ].map((message, index) => ({
      ...message,
      role: message.role as AgentMessage["role"],
      id: `${chatId}-${index}`,
      chatId,
      createdTs: now + index,
    }));
  } else if (scenario.startsWith("storage-")) {
    chat.status = "error";
    chat.lastError = storageErrors[scenario as keyof typeof storageErrors];
  } else if (scenario.startsWith("error-")) {
    chat.status = "error";
    chat.lastError = connectionError;
    if (scenario === "error-no-retry") messages = messages.slice(1);
  } else if (scenario.startsWith("interrupted-")) {
    chat.interrupted = true;
    chat.interruptionReason = scenario.startsWith("interrupted-user")
      ? "user"
      : "unexpected";
    messages[0].metadata = { interruptionReason: chat.interruptionReason };
    if (scenario === "interrupted-user-recoverable") {
      chat.runId = runId;
      chat.totalTokensUsed = 100;
      chat.runBudget = {
        version: 1,
        runId,
        limit: 1000,
        totalTokensUsed: 100,
        hasUnreportedUsage: false,
        providerAttemptCount: 1,
        revision: 0,
      };
      chat.modelAttempts = [
        {
          runId,
          attemptId: `${runId}:1`,
          ordinal: 1,
          outcome: "response",
          usage: { purpose: "agent", toolNames: [], totalTokens: 100 },
        },
      ];
      messages = messages.map((message) => ({
        ...message,
        metadata: { ...message.metadata, runId },
      }));
      chat.runCheckpoint = {
        version: 1,
        runId,
        suspension: "interrupted",
        rootUserMessageId: messages[0].id,
        throughMessageId: messages[1].id,
        bindings: {
          page: { path: "/landing", title: "Home" },
          context: project,
          references: [],
          runMode: "default",
        },
        maskingSections: [],
        phase: "working",
        startupComplete: true,
      };
    }
  } else if (scenario === "ai-configuration") {
    chat.status = "error";
    chat.requiresAIConfiguration = true;
  } else if (
    scenario.startsWith("confirmation-") ||
    scenario === "permission-expired" ||
    ["choose", "choose-custom", "input"].includes(scenario)
  ) {
    chat.status = "awaiting_user";
    pendingAsk = {
      kind: "confirm",
      toolCallId: "layout-confirmation",
      prompt: confirmationPrompt,
      ...(scenario === "confirmation-long"
        ? { confirmLabel, cancelLabel }
        : scenario === "confirmation-two-actions"
          ? {
              confirmLabel: "Prepare draft for review",
              cancelLabel: "Keep current access unchanged",
            }
          : {}),
    };
    if (scenario === "permission-expired") {
      pendingAsk.apiPermission = {
        actionId: "expired-layout-permission",
        operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
        expiresAt: now - 1000,
      };
    } else if (scenario === "choose" || scenario === "choose-custom") {
      pendingAsk = {
        kind: "choose",
        toolCallId: "layout-choice",
        prompt: "Choose a database access scope for review.",
        options: [
          {
            label: "Development databases",
            value: "development",
            description: "Use the databases in the development environment.",
          },
          {
            label: "Choose a different database scope",
            value: "custom",
            description: "Keep the existing database access unchanged.",
            kind: "custom-input",
            placeholder: "Describe the database scope",
          },
        ],
      };
    } else if (scenario === "input") {
      pendingAsk = {
        kind: "input",
        toolCallId: "layout-input",
        prompt: "Which database should I review?",
        defaultValue: "development_orders",
      };
    }
  } else if (scenario.startsWith("token-")) {
    const active = scenario === "token-warning" || scenario === "token-reached";
    const partial = scenario === "token-multi-action";
    const used =
      scenario === "token-warning"
        ? 800
        : scenario === "token-resumable"
          ? 500
          : 1100;
    chat.status = active ? "awaiting_user" : "paused";
    chat.runId = runId;
    chat.totalTokensUsed = used;
    chat.runBudget = {
      version: 1,
      runId,
      limit: 1000,
      totalTokensUsed: used,
      hasUnreportedUsage: partial,
      providerAttemptCount: partial ? 2 : 1,
      revision: 0,
    };
    chat.modelAttempts = [
      {
        runId,
        attemptId: `${runId}:1`,
        ordinal: 1,
        outcome: "response",
        usage: { purpose: "agent", toolNames: [], totalTokens: used },
      },
      ...(partial
        ? [
            {
              runId,
              attemptId: `${runId}:2`,
              ordinal: 2,
              outcome: "error" as const,
            },
          ]
        : []),
    ];
    messages = messages.map((message) => ({ ...message, metadata: { runId } }));
    chat.runCheckpoint = {
      version: 1,
      runId,
      suspension: active ? "awaiting-user" : "token-limit",
      rootUserMessageId: messages[0].id,
      throughMessageId: messages[1].id,
      bindings: {
        page: { path: "/landing", title: "Home" },
        context: project,
        references: [],
        runMode: "default",
      },
      maskingSections: [],
      phase: "working",
      startupComplete: true,
    };
    if (active) {
      // Running work recovers as interrupted; a pending question keeps the
      // active threshold notice stable without starting a model call.
      pendingAsk = {
        kind: "confirm",
        toolCallId: "layout-budget-confirmation",
        prompt: "Continue preparing the unsaved draft?",
      };
    }
    if (partial)
      chat.lastError = "A provider attempt ended before usage was reported.";
  }
  return {
    currentChatId: chatId,
    chats: [chat],
    messagesByChatId: { [chatId]: messages },
    pendingAskByChatId: pendingAsk ? { [chatId]: pendingAsk } : {},
    draftsByChatId: {},
    queuedUserMessagesByChatId: {},
    ...(scenario === "follow-up-queue"
      ? {
          followUpQueuesByChatId: {
            [chatId]: {
              paused: true,
              messages: [
                "Review the development database access policy",
                "Keep the current production database access unchanged",
              ].map((text, index) => ({
                id: `layout-queued-${index}`,
                chatId,
                text,
                page: { path: "/landing", title: "Home" },
              })),
            },
          },
        }
      : {}),
  };
}

async function disableAIReadiness(page: Page) {
  await page.route(
    /\/bytebase\.v1\.(?:ActuatorService\/GetActuatorInfo|SettingService\/GetSetting)(?:\?|$)/,
    async (route) => {
      const actuator = route.request().url().includes("ActuatorService");
      const binary = route
        .request()
        .headers()
        ["content-type"]?.startsWith("application/proto");
      if (!actuator) {
        const request = binary
          ? fromBinary(
              GetSettingRequestSchema,
              route.request().postDataBuffer()!
            )
          : route.request().postDataJSON();
        if (request.name !== "settings/AI") return route.continue();
      }
      const response = await route.fetch();
      if (!response.ok()) return route.fulfill({ response });
      const responseBinary = response
        .headers()
        ["content-type"]?.startsWith("application/proto");
      if (responseBinary) {
        if (actuator) {
          const info = fromBinary(ActuatorInfoSchema, await response.body());
          info.aiEnabled = false;
          return route.fulfill({
            response,
            body: Buffer.from(toBinary(ActuatorInfoSchema, info)),
          });
        }
        const setting = fromBinary(SettingSchema, await response.body());
        setting.value ??= create(SettingValueSchema);
        if (setting.value.value.case === "ai")
          setting.value.value.value.enabled = false;
        return route.fulfill({
          response,
          body: Buffer.from(toBinary(SettingSchema, setting)),
        });
      }
      const value = await response.json();
      if (actuator) value.aiEnabled = false;
      else {
        value.value ??= {};
        value.value.ai ??= {};
        value.value.ai.enabled = false;
      }
      return route.fulfill({ response, body: JSON.stringify(value) });
    }
  );
}

async function openLayoutFixture(
  page: Page,
  scenario: Scenario = "messages",
  actorEmail?: string
) {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const { workspace } = await env.api.getActuatorInfo();
  const workspaceId = workspace.split("/").at(-1)!;
  if (scenario === "ai-configuration" || scenario === "ai-contextual-start")
    await disableAIReadiness(page);
  await page.addInitScript(
    ({ key, state, storageFailure }) => {
      localStorage.setItem(key, JSON.stringify(state));
      localStorage.setItem(
        "bb.release",
        JSON.stringify({
          ignoreRemindModalTillNextRelease: true,
          nextCheckTs: Date.now() + 86_400_000,
        })
      );
      if (storageFailure) {
        const setItem = Storage.prototype.setItem;
        Storage.prototype.setItem = function (key, value) {
          if (key.startsWith("bb-agent-state-v")) {
            throw new DOMException(
              storageFailure === "storage-full"
                ? "Browser storage is full"
                : "Browser storage is unavailable",
              storageFailure === "storage-full"
                ? "QuotaExceededError"
                : "SecurityError"
            );
          }
          setItem.call(this, key, value);
        };
      }
    },
    {
      key: `bb-agent-state-v3.${workspaceId.length}:${workspaceId}.${actorEmail ?? env.adminEmail}`,
      state: layoutState(scenario, workspace, env.project),
      storageFailure: scenario.startsWith("storage-") ? scenario : undefined,
    }
  );
  await page.goto(
    scenario === "ai-contextual-start"
      ? `${env.baseURL}/${env.project}`
      : `${env.baseURL}/landing`
  );
  await page.getByRole("button", { name: "Agent", exact: true }).click();
  const agent = page.locator("[data-agent-window]");
  await expect(agent).toBeVisible();
  if (scenario === "ai-contextual-start") {
    await expect(
      agent.locator('[data-agent-contextual-start="project"]')
    ).toBeVisible();
    return agent;
  }
  const title = agent.getByRole("button", { name: chatTitle, exact: true });
  if ((await title.count()) === 0)
    await agent.getByRole("button", { name: "Chats", exact: true }).click();
  await title.click();
  return agent;
}

async function expectContained(element: Locator, container: Locator) {
  await expect(element).toBeVisible();
  const bounds = (await element.boundingBox())!;
  const containerBounds = (await container.boundingBox())!;
  expect(bounds).not.toBeNull();
  expect(containerBounds).not.toBeNull();
  expect(bounds.x).toBeGreaterThanOrEqual(containerBounds.x - 1);
  expect(bounds.x + bounds.width).toBeLessThanOrEqual(
    containerBounds.x + containerBounds.width + 1
  );
  expect(
    await element.evaluate((node) => node.scrollWidth - node.clientWidth)
  ).toBeLessThanOrEqual(1);
}

async function expectAIConfigurationSpacing(
  alert: Locator,
  width: number,
  actions?: Locator
) {
  const mobile = width < 640;
  for (const side of ["top", "bottom"]) {
    await expect(alert).toHaveCSS(`padding-${side}`, mobile ? "8px" : "12px");
  }
  for (const side of ["left", "right"]) {
    await expect(alert).toHaveCSS(`padding-${side}`, mobile ? "8px" : "16px");
  }
  await expect(alert).toHaveCSS("column-gap", mobile ? "8px" : "12px");
  if (actions) {
    await expect(actions).toHaveCSS("margin-top", mobile ? "8px" : "12px");
  }
}

async function expectBottomActions(
  group: Locator,
  container: Locator,
  preceding: Locator[]
) {
  await expectContained(group, container);
  await expect(group).toHaveCSS("display", "flex");
  await expect(group).toHaveCSS("flex-wrap", "wrap");
  const groupBounds = (await group.boundingBox())!;
  for (const content of preceding) {
    await expectContained(content, container);
    const bounds = (await content.boundingBox())!;
    expect(groupBounds.y).toBeGreaterThanOrEqual(bounds.y + bounds.height - 1);
  }
  const controls = group.getByRole("button").or(group.getByRole("link"));
  expect(await controls.count()).toBeGreaterThan(0);
  for (const control of await controls.all()) {
    await expectContained(control, group);
    await expect(control).toBeInViewport();
    if (await control.isEnabled()) {
      await control.focus();
      await expect(control).toBeFocused();
    }
  }
  const lastBounds = (await controls.last().boundingBox())!;
  expect(
    Math.abs(
      lastBounds.x + lastBounds.width - groupBounds.x - groupBounds.width
    )
  ).toBeLessThanOrEqual(1);
}

async function captureScenario(
  page: Page,
  focus: Locator,
  scenario: string,
  width: number,
  testInfo: TestInfo
) {
  const windowPath = testInfo.outputPath(
    `agent-${scenario}-${width}-window.png`
  );
  const detailPath = testInfo.outputPath(
    `agent-${scenario}-${width}-detail.png`
  );
  await page
    .locator("[data-agent-window]")
    .screenshot({ path: windowPath, animations: "disabled" });
  await focus.screenshot({ path: detailPath, animations: "disabled" });
  const manifestPath = testInfo.outputPath("screenshots.json");
  const manifest: {
    title: string;
    width: number;
    screenshots: { file: string; label: string }[];
  } = existsSync(manifestPath)
    ? JSON.parse(readFileSync(manifestPath, "utf8"))
    : { title: testInfo.title, width, screenshots: [] };
  manifest.screenshots.push(
    { file: basename(windowPath), label: `${scenario}: Agent window` },
    { file: basename(detailPath), label: `${scenario}: surface` }
  );
  writeFileSync(manifestPath, JSON.stringify(manifest, null, 2));
  await testInfo.attach(`gallery-${scenario}-${width}`, {
    body: JSON.stringify({ scenario, width, windowPath, detailPath }),
    contentType: "application/json",
  });
}

for (const width of widths) {
  for (const scenario of [
    "storage-full",
    "storage-unavailable",
    "error-retry",
    "error-no-retry",
  ] as const) {
    test(`Agent ${scenario} actions stay below diagnostics at ${width}px`, async ({
      page,
      provider,
    }, testInfo) => {
      await page.setViewportSize({ width, height: 900 });
      const agent = await openLayoutFixture(page, scenario);
      const pane = agent.locator("[data-agent-chat-scroll-area]");
      const alert = scenario.startsWith("storage-")
        ? pane.locator("[data-agent-storage-status]")
        : pane.getByRole("alert");
      const summary = alert.locator(
        scenario.startsWith("storage-")
          ? "[data-agent-storage-header]"
          : "[data-agent-error-summary]"
      );
      const details = alert.locator("[data-agent-error-details]");
      const actions = alert.locator("[data-agent-error-actions]");
      const actionLabel =
        scenario === "storage-full"
          ? "Free space and retry"
          : scenario === "storage-unavailable"
            ? "Retry saving"
            : "Retry";
      const recovery = alert.getByRole("button", {
        name: actionLabel,
        exact: true,
      });
      await expectContained(alert, pane);
      await expect(details).toBeVisible();
      if (scenario === "error-no-retry") {
        await expect(actions).toHaveCount(0);
        await expect(recovery).toHaveCount(0);
      } else {
        await expect(recovery).toBeVisible();
        await expectBottomActions(actions, alert, [summary, details]);
      }
      await captureScenario(
        page,
        alert,
        `${scenario}-collapsed`,
        width,
        testInfo
      );
      await details.click();
      await expect(details).toHaveAttribute("aria-expanded", "true");
      const diagnostics = alert.locator("pre");
      await expect(diagnostics).toHaveText(
        scenario.startsWith("storage-")
          ? storageErrors[scenario as keyof typeof storageErrors]
          : connectionError
      );
      await expectContained(diagnostics, alert);
      if (scenario !== "error-no-retry")
        await expectBottomActions(actions, alert, [
          summary,
          details,
          diagnostics,
        ]);
      await captureScenario(
        page,
        alert,
        `${scenario}-expanded`,
        width,
        testInfo
      );
      await details.click();
      await expect(diagnostics).toBeHidden();
      if (scenario === "storage-full") {
        await recovery.click();
        const confirmation = page.locator("[data-agent-storage-confirmation]");
        await expect(confirmation).toBeVisible();
        await expectContained(confirmation, page.locator("body"));
        const controls = confirmation.getByRole("button");
        await expect(controls).toHaveCount(2);
        await expectBottomActions(
          controls.first().locator(".."),
          confirmation,
          [confirmation.getByRole("heading")]
        );
        await captureScenario(
          page,
          confirmation,
          "storage-cleanup-confirmation",
          width,
          testInfo
        );
        await confirmation
          .getByRole("button", { name: "Cancel", exact: true })
          .click();
        await expect(confirmation).toBeHidden();
        await expect(recovery).toBeFocused();
      }
      expect(provider.calls).toBe(0);
    });
  }

  for (const scenario of [
    "interrupted-unexpected",
    "interrupted-user",
    "interrupted-user-recoverable",
  ] as const) {
    test(`Agent ${scenario} actions follow the warning at ${width}px`, async ({
      page,
      provider,
    }, testInfo) => {
      await page.setViewportSize({ width, height: 900 });
      const agent = await openLayoutFixture(page, scenario);
      const status = agent.locator("[data-agent-interruption-status]");
      const title = status.getByText(
        scenario.startsWith("interrupted-user")
          ? "Request stopped by user"
          : "Interrupted",
        { exact: true }
      );
      const hint = status.getByText(
        scenario.startsWith("interrupted-user")
          ? "An action already sent may still complete. Check its status before retrying."
          : "The conversation was interrupted. You can retry or send a new message.",
        { exact: true }
      );
      const actions = status.locator("[data-agent-interruption-actions]");
      await expect(title).toBeVisible();
      await expect(hint).toBeVisible();
      if (scenario === "interrupted-user") {
        await expect(actions).toHaveCount(0);
        await expect(status.getByRole("button")).toHaveCount(0);
      } else {
        await expectBottomActions(actions, status, [title, hint]);
        await expect(
          actions.getByRole("button", { name: "Retry", exact: true })
        ).toBeEnabled();
      }
      await captureScenario(page, status, scenario, width, testInfo);
      expect(provider.calls).toBe(0);
    });
  }

  for (const scenario of [
    "confirmation-default",
    "confirmation-long",
    "confirmation-two-actions",
    "permission-expired",
  ] as const) {
    test(`Agent ${scenario} actions wrap at the bottom right at ${width}px`, async ({
      page,
      provider,
    }, testInfo) => {
      await page.setViewportSize({ width, height: 900 });
      const agent = await openLayoutFixture(page, scenario);
      const actions = agent.locator("[data-agent-confirmation-actions]");
      const container = actions.locator("..");
      const prompt = container.getByText(
        scenario === "permission-expired"
          ? "Permission request expired"
          : confirmationPrompt,
        { exact: true }
      );
      await expectBottomActions(actions, container, [prompt]);
      await expect(actions.getByRole("button")).toHaveCount(2);
      if (scenario === "confirmation-long") {
        for (const label of [confirmLabel, cancelLabel]) {
          const button = actions.getByRole("button", {
            name: label,
            exact: true,
          });
          await expect(button).toHaveText(label);
          await expect(button).toHaveCSS("white-space", "normal");
        }
      } else if (scenario === "permission-expired") {
        await expect(
          actions.getByRole("button", { name: "Allow once", exact: true })
        ).toBeDisabled();
        await expect(
          actions.getByRole("button", { name: "Dismiss", exact: true })
        ).toBeEnabled();
      }
      await captureScenario(page, container, scenario, width, testInfo);
      expect(provider.calls).toBe(0);
    });
  }

  for (const scenario of ["choose", "choose-custom", "input"] as const) {
    test(`Agent ${scenario} reply controls fit at ${width}px`, async ({
      page,
      provider,
    }, testInfo) => {
      await page.setViewportSize({ width, height: 900 });
      const agent = await openLayoutFixture(page, scenario);
      if (scenario !== "input")
        await agent
          .getByRole("radio", {
            name:
              scenario === "choose"
                ? /Development databases/
                : /Choose a different database scope/,
          })
          .click();
      const reply = agent.getByRole("button", { name: "Reply", exact: true });
      const composer =
        scenario === "choose"
          ? agent.getByRole("radiogroup").locator("../..")
          : agent.locator("[data-agent-input-row]").locator("../..");
      if (scenario === "choose-custom")
        await agent
          .getByRole("textbox", { name: "Details", exact: true })
          .fill("development_orders");
      await expect(reply).toBeEnabled();
      await expectContained(reply, composer);
      await expectContained(composer, agent);
      if (scenario !== "input")
        await expectContained(agent.getByRole("radiogroup"), composer);
      await captureScenario(page, composer, scenario, width, testInfo);
      expect(provider.calls).toBe(0);
    });
  }

  test(`Agent AI configuration actions fit at ${width}px`, async ({
    page,
    provider,
  }, testInfo) => {
    await page.setViewportSize({ width, height: 900 });
    const agent = await openLayoutFixture(page, "ai-configuration");
    const alert = agent.locator("[data-agent-ai-configuration]");
    const actions = alert.locator("[data-agent-ai-configuration-actions]");
    await expectAIConfigurationSpacing(alert, width, actions);
    await expectBottomActions(actions, alert, [
      alert.getByText("AI Not Configured", { exact: true }),
      alert.getByText("AI is not configured for this workspace.", {
        exact: true,
      }),
    ]);
    await expect(
      actions.getByRole("link", { name: "Configure", exact: true })
    ).toBeVisible();
    await expect(alert.getByRole("link")).toHaveCount(1);
    await expect(alert.getByRole("button")).toHaveCount(0);
    await captureScenario(page, alert, "ai-configuration", width, testInfo);
    expect(provider.calls).toBe(0);
  });

  test(`Agent contextual AI configuration action fits at ${width}px`, async ({
    page,
    provider,
  }, testInfo) => {
    await page.setViewportSize({ width, height: 900 });
    const agent = await openLayoutFixture(page, "ai-contextual-start");
    const alert = agent.locator("[data-agent-ai-configuration]");
    const actions = alert.locator("[data-agent-ai-configuration-actions]");
    await expectAIConfigurationSpacing(alert, width, actions);
    await expectBottomActions(actions, alert, [
      alert.getByText("AI Not Configured", { exact: true }),
      alert.getByText("AI is not configured for this workspace.", {
        exact: true,
      }),
    ]);
    await expect(
      actions.getByRole("link", { name: "Configure", exact: true })
    ).toBeVisible();
    await expect(actions.getByRole("button")).toHaveCount(0);
    await captureScenario(
      page,
      alert,
      "ai-contextual-configure-only",
      width,
      testInfo
    );
    expect(provider.calls).toBe(0);
  });

  test(`Agent restricted AI configuration recovery fits at ${width}px`, async ({
    page,
    context,
    provider,
  }, testInfo) => {
    await page.setViewportSize({ width, height: 900 });
    const env = loadTestEnv();
    await env.api.login(env.adminEmail, env.adminPassword);
    const { workspace } = await env.api.getActuatorInfo();
    const email = `agent-layout-member-${width}-${Date.now()}@example.com`;
    const password = "e2e-layout-member-1!";
    await env.api.createUser(email, password, "Layout restricted member");
    try {
      await env.api.addWorkspaceRoleMember(
        workspace,
        email,
        "roles/workspaceMember"
      );
      await env.api.appendProjectBinding(env.project, "roles/projectViewer", [
        `user:${email}`,
      ]);
      await loginBrowserContext(context, env.baseURL, email, password);
      const agent = await openLayoutFixture(page, "ai-configuration", email);
      const alert = agent.locator("[data-agent-ai-configuration]");
      const actions = alert.locator("[data-agent-ai-configuration-actions]");
      const contact = alert.getByText(
        "Contact your workspace admin to configure AI.",
        { exact: true }
      );
      await expectAIConfigurationSpacing(alert, width);
      await expectContained(alert, agent);
      await expectContained(contact, alert);
      await expect(actions).toHaveCount(0);
      await expect(alert.getByRole("link")).toHaveCount(0);
      await expect(alert.getByRole("button")).toHaveCount(0);
      await captureScenario(
        page,
        alert,
        "ai-configuration-restricted",
        width,
        testInfo
      );
      expect(provider.calls).toBe(0);
    } finally {
      const policy = await env.api.getProjectIamPolicy(env.project);
      await env.api.setProjectIamPolicy(env.project, {
        ...policy,
        bindings: policy.bindings.map((binding) => ({
          ...binding,
          members: binding.members.filter(
            (member) => member !== `user:${email}`
          ),
        })),
      });
      await env.api.removeWorkspaceRoleMember(
        workspace,
        email,
        "roles/workspaceMember"
      );
      await env.api.deleteUser(email);
    }
  });

  test(`Agent paused follow-up queue controls fit at ${width}px`, async ({
    page,
    provider,
  }, testInfo) => {
    await page.setViewportSize({ width, height: 900 });
    const agent = await openLayoutFixture(page, "follow-up-queue");
    const queue = agent
      .getByRole("list", { name: "2 queued", exact: true })
      .locator("..");
    await expect(queue.getByRole("listitem")).toHaveCount(2);
    for (const name of [
      "Resume queue",
      "Remove queued message 1",
      "Remove queued message 2",
    ]) {
      const control = queue.getByRole("button", { name, exact: true });
      await expect(control).toBeEnabled();
      await expectContained(control, queue);
    }
    await expectContained(queue, agent);
    await captureScenario(
      page,
      queue,
      "follow-up-queue-paused",
      width,
      testInfo
    );
    expect(provider.calls).toBe(0);
  });

  test(`Agent Send Stop and Queue controls fit at ${width}px`, async ({
    page,
    provider,
  }, testInfo) => {
    await page.setViewportSize({ width, height: 900 });
    const agent = await openLayoutFixture(page, "interrupted-unexpected");
    const composer = agent.locator("[data-agent-composer]");
    const input = agent.locator("textarea").last();
    await input.fill("Review the development database access policy.");
    const send = composer.getByRole("button", { name: "Send", exact: true });
    await expect(send).toBeEnabled();
    await expectContained(send, composer);
    await captureScenario(page, composer, "composer-send", width, testInfo);
    const release = provider.hold(
      answer(20, "The scripted layout review completed.")
    );
    try {
      const requested = page.waitForRequest(
        /\/bytebase\.v1\.AIService\/Chat(?:\?|$)/
      );
      await send.click();
      await requested;
      const stop = composer.getByRole("button", { name: "Stop", exact: true });
      const queue = composer.getByRole("button", {
        name: "Queue",
        exact: true,
      });
      await input.fill("Check the production database access policy next.");
      for (const control of [stop, queue]) {
        await expect(control).toBeEnabled();
        await expectContained(control, composer);
      }
      await captureScenario(
        page,
        composer,
        "composer-stop-and-queue",
        width,
        testInfo
      );
      await stop.click();
      await expect(stop).toBeHidden();
      expect(provider.calls).toBe(1);
    } finally {
      release();
    }
  });

  for (const scenario of [
    "token-warning",
    "token-reached",
    "token-exhausted",
    "token-resumable",
    "token-multi-action",
  ] as const) {
    test(`Agent ${scenario} actions follow all content at ${width}px`, async ({
      page,
      provider,
    }, testInfo) => {
      await page.setViewportSize({ width, height: 900 });
      const agent = await openLayoutFixture(page, scenario);
      const status = agent.locator("[data-agent-token-budget]");
      const actions = status.locator("[data-agent-token-budget-actions]");
      const content = status.locator("[data-agent-token-budget-content]");
      await expectBottomActions(
        actions,
        status,
        await content
          .locator(":scope > :not([data-agent-token-budget-actions])")
          .all()
      );
      await expect(actions.getByRole("button")).toHaveCount(
        scenario === "token-warning" || scenario === "token-reached" ? 1 : 2
      );
      if (scenario === "token-resumable") {
        await expect(
          actions.getByRole("button", { name: "Retry", exact: true })
        ).toBeEnabled();
        await expect(
          actions.getByRole("button", { name: "Increase limit", exact: true })
        ).toHaveCount(0);
      } else
        await expect(
          actions.getByRole("button", { name: "Increase limit", exact: true })
        ).toBeEnabled();
      if (scenario === "token-multi-action") {
        await expect(content).toContainText(
          "Some token usage was not reported."
        );
        await status.locator("[data-agent-error-details]").click();
        await expect(status.locator("pre")).toBeVisible();
        await expectBottomActions(actions, status, [
          ...(await content
            .locator(":scope > :not([data-agent-token-budget-actions])")
            .all()),
          status.locator("pre"),
        ]);
      }
      await captureScenario(page, status, scenario, width, testInfo);
      if (scenario === "token-exhausted") {
        await actions
          .getByRole("button", { name: "Increase limit", exact: true })
          .click();
        const dialog = page.getByRole("dialog", {
          name: "Increase limit for this run",
          exact: true,
        });
        await expectContained(dialog, page.locator("body"));
        const input = dialog.getByRole("spinbutton", {
          name: "This run's token limit",
          exact: true,
        });
        await expect(input).toHaveValue("2");
        const confirm = dialog.getByRole("button", {
          name: "Increase and resume",
          exact: true,
        });
        await expect(confirm).toBeEnabled();
        await expectBottomActions(confirm.locator(".."), dialog, [
          dialog.getByRole("heading"),
          input,
        ]);
        await captureScenario(
          page,
          dialog,
          "token-increase-valid",
          width,
          testInfo
        );
        await input.fill("1");
        await expect(confirm).toBeDisabled();
        await expect(dialog.getByRole("alert")).toBeVisible();
        await captureScenario(
          page,
          dialog,
          "token-increase-invalid",
          width,
          testInfo
        );
        await dialog
          .getByRole("button", { name: "Cancel", exact: true })
          .click();
        await expect(dialog).toBeHidden();
      }
      expect(provider.calls).toBe(0);
    });
  }

  test(`Agent API permission preview actions fit at ${width}px without sending a write`, async ({
    page,
    provider,
  }, testInfo) => {
    await page.setViewportSize({ width, height: 900 });
    const env = loadTestEnv();
    let writes = 0;
    await page.route(
      /\/bytebase\.v1\.DatabaseService\/UpdateDatabase(?:\?|$)/,
      async (route) => {
        writes++;
        await route.abort();
      }
    );
    provider.respond({
      toolCalls: [
        {
          id: "layout-protected-api",
          name: "call_api",
          arguments: JSON.stringify({
            operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
            body: {
              database: {
                name: env.database,
                labels: { "layout-review": "prepared-not-sent" },
              },
              updateMask: "labels",
            },
          }),
        },
      ],
      usage: { totalTokens: 20, inputTokens: 15, outputTokens: 5 },
      stopReason: "TURN_COMPLETED",
    });
    const agent = await openLayoutFixture(page, "error-retry");
    await agent
      .locator("textarea")
      .last()
      .fill(
        "Prepare the fixture database label update for my review, without sending it."
      );
    await agent.getByRole("button", { name: "Send", exact: true }).click();
    const actions = agent.locator("[data-agent-confirmation-actions]");
    const container = actions.locator("..");
    const preview = container.locator("pre");
    await expect(preview).toContainText('"layout-review": "prepared-not-sent"');
    await expectBottomActions(actions, container, [
      container.getByText("Request permission", { exact: true }),
      preview,
    ]);
    await expect(
      actions.getByRole("button", { name: "Allow once", exact: true })
    ).toBeEnabled();
    await expect(
      actions.getByRole("button", { name: "Deny", exact: true })
    ).toBeEnabled();
    await captureScenario(
      page,
      container,
      "permission-live-preview",
      width,
      testInfo
    );
    expect(provider.calls).toBe(1);
    expect(writes).toBe(0);
  });

  test(`long Agent messages stay inside the conversation at ${width}px`, async ({
    page,
    provider,
  }, testInfo) => {
    await page.setViewportSize({ width, height: 900 });
    const agent = await openLayoutFixture(page);
    const pane = agent.locator("[data-agent-chat-scroll-area]");
    const userBubble = agent.locator("[data-agent-user-message]");
    const answers = agent.locator("[data-agent-answer]");
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
    await expect
      .poll(() =>
        pre.evaluate((element) => element.scrollWidth - element.clientWidth)
      )
      .toBeGreaterThan(0);
    await expect
      .poll(() =>
        pane.evaluate((element) => element.scrollWidth - element.clientWidth)
      )
      .toBeLessThanOrEqual(1);
    for (const bubble of [userBubble, answers.first(), answers.nth(1)])
      await expectContained(bubble, pane);
    await captureScenario(page, pane, "long-messages", width, testInfo);
    expect(provider.calls).toBe(0);
  });
}
