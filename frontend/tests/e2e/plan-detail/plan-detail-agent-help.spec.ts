import {
  create,
  fromBinary,
  fromJsonString,
  toBinary,
  toJsonString,
  type DescMessage,
  type MessageShape,
} from "@bufbuild/protobuf";
import {
  expect,
  test,
  type Locator,
  type Page,
  type Route,
} from "@playwright/test";
import { ActuatorInfoSchema } from "../../../src/types/proto-es/v1/actuator_service_pb";
import {
  AIChatMessageRole,
  AIChatRequestSchema,
  AIChatResponseSchema,
  type AIChatRequest,
} from "../../../src/types/proto-es/v1/ai_service_pb";
import {
  GetSettingRequestSchema,
  SettingSchema,
} from "../../../src/types/proto-es/v1/setting_service_pb";
import { BytebaseApiClient } from "../framework/api-client";
import { loadTestEnv } from "../framework/env";
import { execSql, getInstancePgPort, querySql } from "../framework/psql";
import { loginBrowserContext } from "../framework/sign-in";
import { waitForPlanChecksDone } from "./plan-helpers";
import { PlanDetailPage } from "./plan-detail.page";

test.setTimeout(180_000);

type Env = ReturnType<typeof loadTestEnv>;
type Fixture = {
  plan: string;
  schema: string;
  sql: string;
  port: string;
  cleanup: () => Promise<void>;
};

function decode<T extends DescMessage>(schema: T, body: Buffer, json: boolean) {
  return json
    ? fromJsonString(schema, body.toString())
    : fromBinary(schema, body);
}

async function fulfill<T extends DescMessage>(
  route: Route,
  schema: T,
  message: MessageShape<T>
) {
  const json = route.request().headers()["content-type"]?.includes("json");
  await route.fulfill({
    contentType: json ? "application/json" : "application/proto",
    body: json
      ? toJsonString(schema, message)
      : Buffer.from(toBinary(schema, message)),
  });
}

// The embedded release frontend uses binary Connect; dev uses protobuf JSON.
// Stub the provider/configuration boundary, not the diagnostic tool executor.
async function stubAI(page: Page, enabled: boolean, plan: string) {
  const requests: AIChatRequest[] = [];
  await page.route(
    "**/bytebase.v1.ActuatorService/GetActuatorInfo",
    async (route) => {
      const response = await route.fetch();
      const info = decode(
        ActuatorInfoSchema,
        await response.body(),
        response.headers()["content-type"]?.includes("json") ?? false
      );
      info.aiEnabled = enabled;
      await fulfill(route, ActuatorInfoSchema, info);
    }
  );
  await page.route(
    "**/bytebase.v1.SettingService/GetSetting",
    async (route) => {
      const request = decode(
        GetSettingRequestSchema,
        route.request().postDataBuffer()!,
        route.request().headers()["content-type"]?.includes("json") ?? false
      );
      if (request.name !== "settings/AI") return route.continue();
      await fulfill(
        route,
        SettingSchema,
        create(SettingSchema, {
          name: "settings/AI",
          value: { value: { case: "ai", value: { enabled } } },
        })
      );
    }
  );
  await page.route("**/bytebase.v1.AIService/Chat", async (route) => {
    const request = decode(
      AIChatRequestSchema,
      route.request().postDataBuffer()!,
      route.request().headers()["content-type"]?.includes("json") ?? false
    );
    requests.push(request);
    const denied = request.messages.some(
      (message) => message.toolCallId === "attempt-mutation"
    );
    await fulfill(
      route,
      AIChatResponseSchema,
      create(AIChatResponseSchema, {
        toolCalls: [
          {
            id: denied ? "finish-diagnosis" : "attempt-mutation",
            name: denied ? "done" : "call_api",
            arguments: JSON.stringify(
              denied
                ? {
                    text: "The nullable column violates the configured SQL review rule. Add NOT NULL after checking existing rows. No changes were executed.",
                    success: true,
                    actions: [],
                  }
                : {
                    operationId: "bytebase.v1.PlanService.UpdatePlan",
                    body: {
                      plan: { name: plan, title: "This must never be saved." },
                      updateMask: "title",
                    },
                  }
            ),
          },
        ],
      })
    );
  });
  return requests;
}

async function createFixture(
  env: Env,
  level: "WARNING" | "ERROR"
): Promise<Fixture> {
  await env.api.login(env.adminEmail, env.adminPassword);
  const id = `agent-check-${Date.now()}`;
  const schema = id.replaceAll("-", "_");
  const port = await getInstancePgPort(env);
  const originalTag = await env.api.getPolicy(`${env.project}/policies/tag`);
  const project = await env.api.getProject(env.project);
  const settings = {
    requireIssueApproval: !!project.requireIssueApproval,
    requirePlanCheckNoError: !!project.requirePlanCheckNoError,
  };
  let plan = "";
  let config = "";
  const cleanup = async () => {
    const errors: unknown[] = [];
    for (const restore of [
      async () => {
        if (plan) await env.api.closePlan(plan);
      },
      async () => {
        if (originalTag)
          await env.api.upsertPolicy(env.project, "tag", originalTag);
        else await env.api.deletePolicy(env.project, "tag");
      },
      async () => {
        if (config) await env.api.deleteReviewConfig(config);
      },
      () => env.api.updateProjectSettings(env.project, settings),
      async () => {
        execSql(
          env.databaseId,
          port,
          `DROP SCHEMA IF EXISTS ${schema} CASCADE;`
        );
      },
      () => env.api.syncDatabase(env.database),
    ]) {
      try {
        await restore();
      } catch (error) {
        errors.push(error);
      }
    }
    if (errors.length)
      throw new AggregateError(errors, "Diagnostic fixture cleanup failed");
  };
  try {
    execSql(
      env.databaseId,
      port,
      `CREATE SCHEMA ${schema}; CREATE TABLE ${schema}.records (id INTEGER PRIMARY KEY);`
    );
    await env.api.syncDatabase(env.database);
    config = (
      await env.api.upsertReviewConfig(
        id,
        "Agent diagnostic check",
        [{ type: "COLUMN_NO_NULL", level, engine: "POSTGRES" }],
        true
      )
    ).name;
    await env.api.upsertReviewConfigTag(env.project, config);
    await env.api.updateProjectSettings(env.project, {
      requireIssueApproval: false,
      requirePlanCheckNoError: true,
    });
    const sql = `ALTER TABLE ${schema}.records ADD COLUMN diagnostic_detail TEXT;`;
    const sheet = await env.api.createSheet(env.project, sql);
    plan = (
      await env.api.createPlan(env.project, `Agent ${level} diagnosis`, [
        { id: "diagnostic-spec", targets: [env.database], sheet },
      ])
    ).name;
    await env.api.runPlanChecks(plan);
    await waitForPlanChecksDone(env.api, plan);
    const check = await env.api.getPlanCheckRun(plan);
    expect(
      check.results.some(
        (result) => result.status === level && result.title === "COLUMN_NO_NULL"
      )
    ).toBe(true);
    return { plan, schema, sql, port, cleanup };
  } catch (error) {
    await cleanup();
    throw error;
  }
}

async function openChecks(
  page: Page,
  env: Env,
  fixture: Fixture,
  checksLabel = "Checks"
) {
  const planPage = new PlanDetailPage(page, env.baseURL);
  await planPage.goto(
    env.project.split("/").at(-1)!,
    fixture.plan.split("/").at(-1)!
  );
  await planPage.dismissModals();
  await planPage.expandSection("Changes");
  await page
    .locator("#plan-phase-changes")
    .getByRole("button", { name: checksLabel, exact: true })
    .click();
  const drawer = page.getByRole("dialog", { name: checksLabel, exact: true });
  await expect(drawer).toBeVisible();
  return drawer;
}

async function assertFits(page: Page, target: Locator) {
  await target.scrollIntoViewIfNeeded();
  const box = await target.boundingBox();
  expect(box).not.toBeNull();
  const viewport = page.viewportSize()!;
  expect(box!.x).toBeGreaterThanOrEqual(0);
  expect(box!.x + box!.width).toBeLessThanOrEqual(viewport.width + 1);
  expect(box!.y).toBeGreaterThanOrEqual(0);
  expect(box!.y + box!.height).toBeLessThanOrEqual(viewport.height + 1);
}

async function persistedAgent(page: Page) {
  return page.evaluate(() => {
    const key = Object.keys(localStorage).find((key) =>
      key.startsWith("bb-agent-state-v2.")
    );
    if (!key) return null;
    return JSON.parse(localStorage.getItem(key)!) as {
      currentChatId: string;
      chats: {
        id: string;
        capability?: {
          kind: string;
          resource: { kind: string; name: string; project: string };
        };
        requiresAIConfiguration?: boolean;
      }[];
      messagesByChatId: Record<
        string,
        {
          role: string;
          content: string;
          metadata?: {
            resourceReference?: { name: string };
            resourceReferences?: { name: string }[];
          };
        }[]
      >;
      draftsByChatId: Record<
        string,
        { text: string; mentions?: { resource: { name: string } }[] }
      >;
    };
  });
}

for (const scenario of [
  {
    level: "WARNING",
    viewport: { width: 1440, height: 1000 },
    label: "desktop",
    locale: "en-US",
    checks: "Checks",
    help: "Ask Agent",
    agent: "Agent",
    continueLastChat: "Continue last chat",
  },
  {
    level: "ERROR",
    viewport: { width: 390, height: 844 },
    label: "mobile",
    locale: "es-ES",
    checks: "Verificaciones",
    help: "Preguntar al Agente",
    agent: "Asistente",
    continueLastChat: "Continuar el último chat",
  },
] as const) {
  test(`${scenario.label} ${scenario.level} check opens a resource-bound diagnosis and blocks a model mutation`, async ({
    page,
  }, testInfo) => {
    const env = loadTestEnv();
    const fixture = await createFixture(env, scenario.level);
    try {
      await page.setViewportSize(scenario.viewport);
      await page.addInitScript(
        (locale) => localStorage.setItem("bb.language", locale),
        scenario.locale
      );
      const requests = await stubAI(page, true, fixture.plan);
      const drawer = await openChecks(page, env, fixture, scenario.checks);
      const help = drawer.getByRole("button", {
        name: scenario.help,
        exact: true,
      });
      await expect(help).toHaveCount(1);
      await assertFits(page, help);
      await help.focus();
      await expect(help).toBeFocused();
      await page.screenshot({
        path: testInfo.outputPath(`${scenario.label}-sql-check.png`),
      });
      const mutations: string[] = [];
      page.on("request", (request) => {
        if (
          /\/(?:Create|Update|Delete|BatchRun|BatchCancel|Approve|Reject|Query)[^/]*$/.test(
            new URL(request.url()).pathname
          )
        )
          mutations.push(request.url());
      });
      await page.keyboard.press("Enter");
      const agent = page.locator("[data-agent-window]");
      await expect(agent).toBeVisible();
      await expect(
        agent.getByText("No changes were executed.", { exact: false })
      ).toBeVisible({ timeout: 30_000 });
      expect(requests).toHaveLength(2);
      const prompt = requests[0].messages.find(
        (message) =>
          message.role === AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_USER
      )!.content!;
      const snapshot = JSON.parse(
        prompt.split("\n").find((line) => line.startsWith('{"resource":'))!
      );
      expect(snapshot).toMatchObject({
        resource: { kind: "plan", name: fixture.plan, project: env.project },
        kind: "sql-check",
        sourceId: `${fixture.plan}/planCheckRun`,
        summary: "COLUMN_NO_NULL",
      });
      expect(snapshot.facts).toEqual(
        expect.arrayContaining([
          { label: "Severity", value: scenario.level },
          { label: "Target", value: env.database },
        ])
      );
      expect(snapshot.detail).toContain("diagnostic_detail");
      const denied = requests[1].messages.find(
        (message) => message.toolCallId === "attempt-mutation"
      );
      expect(denied?.content).toContain("not available in a diagnostic chat");
      expect(mutations).toEqual([]);
      expect((await env.api.getPlan(fixture.plan)).title).toBe(
        `Agent ${scenario.level} diagnosis`
      );
      const state = await persistedAgent(page);
      const chat = state!.chats.find(
        (chat) => chat.id === state!.currentChatId
      )!;
      expect(chat.capability).toMatchObject({
        kind: "diagnostic",
        resource: { kind: "plan", name: fixture.plan, project: env.project },
      });
      const user = state!.messagesByChatId[chat.id].find(
        (message) => message.role === "user"
      )!;
      expect(user.content).toContain(fixture.plan);
      expect(
        user.metadata?.resourceReference?.name ??
          user.metadata?.resourceReferences?.[0]?.name
      ).toBe(fixture.plan);
      expect(
        querySql(
          env.databaseId,
          fixture.port,
          `SELECT count(*) FROM information_schema.columns WHERE table_schema = '${fixture.schema}' AND column_name = 'diagnostic_detail';`
        )
      ).toBe("0");
      await page.screenshot({
        path: testInfo.outputPath(`${scenario.label}-diagnosis.png`),
      });
      await page.reload();
      await page
        .getByRole("button", { name: scenario.agent, exact: true })
        .click();
      await expect(page.locator("[data-agent-window]")).toBeVisible();
      await page
        .getByRole("button", { name: scenario.continueLastChat, exact: true })
        .click();
      await expect(
        page
          .locator("[data-agent-window]")
          .getByText("No changes were executed.", { exact: false })
      ).toBeVisible();
      const hydrated = await persistedAgent(page);
      expect(
        hydrated!.chats.find((item) => item.id === chat.id)?.capability
      ).toEqual(chat.capability);
      expect(requests).toHaveLength(2);
      await page.screenshot({
        path: testInfo.outputPath(`${scenario.label}-hydrated-diagnosis.png`),
      });
    } finally {
      await fixture.cleanup();
    }
  });
}

test("an unconfigured workspace retains the diagnosis as an unsent draft and offers configuration recovery", async ({
  page,
}, testInfo) => {
  const env = loadTestEnv();
  const fixture = await createFixture(env, "ERROR");
  try {
    const requests = await stubAI(page, false, fixture.plan);
    const drawer = await openChecks(page, env, fixture);
    await drawer
      .getByRole("button", { name: "Ask Agent", exact: true })
      .click();
    const agent = page.locator("[data-agent-window]");
    await expect(
      agent.getByText("AI Not Configured", { exact: true })
    ).toBeVisible();
    await expect(
      agent.getByRole("link", { name: "Configure", exact: true })
    ).toBeVisible();
    await expect
      .poll(async () => {
        const state = await persistedAgent(page);
        return state?.draftsByChatId[state.currentChatId]?.text;
      })
      .toContain("COLUMN_NO_NULL");
    const state = await persistedAgent(page);
    expect(
      state!.draftsByChatId[state!.currentChatId].mentions?.[0].resource.name
    ).toBe(fixture.plan);
    expect(
      state!.chats.find((chat) => chat.id === state!.currentChatId)
    ).toMatchObject({
      requiresAIConfiguration: true,
      capability: { kind: "diagnostic", resource: { name: fixture.plan } },
    });
    expect(requests).toEqual([]);
    await page.screenshot({
      path: testInfo.outputPath("ai-configuration-recovery.png"),
    });
    await agent.getByRole("link", { name: "Configure", exact: true }).click();
    await expect(page).toHaveURL(/\/setting\/general\?.*#ai-assistant$/);
  } finally {
    await fixture.cleanup();
  }
});

test("a member without project access cannot obtain a diagnostic from the Plan", async ({
  page,
}) => {
  const env = loadTestEnv();
  const fixture = await createFixture(env, "ERROR");
  const email = `agent-restricted-${Date.now()}@example.com`;
  const password = "diagnostic-e2e-only-123";
  try {
    await env.api.createUser(email, password, "Restricted diagnostic member");
    const restricted = await BytebaseApiClient.asUser(
      env.baseURL,
      email,
      password
    );
    await expect(restricted.getPlan(fixture.plan)).rejects.toThrow(
      /403|permission/i
    );
    await page.context().clearCookies();
    await loginBrowserContext(page.context(), env.baseURL, email, password);
    const requests = await stubAI(page, false, fixture.plan);
    await page.goto(`${env.baseURL}/${fixture.plan}`);
    await expect(
      page
        .getByText(/missing.*permission|permission.*denied|not found/i)
        .first()
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Ask Agent", exact: true })
    ).toHaveCount(0);
    expect(
      (await persistedAgent(page))?.chats.filter((chat) => chat.capability) ??
        []
    ).toEqual([]);
    expect(requests).toEqual([]);
  } finally {
    await env.api.deleteUser(email);
    await fixture.cleanup();
  }
});
