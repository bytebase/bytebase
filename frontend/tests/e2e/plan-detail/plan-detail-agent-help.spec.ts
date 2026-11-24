import { join } from "node:path";
import {
  create,
  fromBinary,
  fromJsonString,
  toBinary,
  toJsonString,
  type DescMessage,
  type MessageShape,
} from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  expect,
  test,
  type Locator,
  type Page,
  type Route,
  type TestInfo,
} from "@playwright/test";
import { TIMESTAMP_COLUMN_WIDTH } from "../../../src/components/timestampColumn";
import { ActuatorInfoSchema } from "../../../src/types/proto-es/v1/actuator_service_pb";
import {
  AIChatMessageRole,
  AIChatRequestSchema,
  AIChatResponseSchema,
  type AIChatRequest,
} from "../../../src/types/proto-es/v1/ai_service_pb";
import {
  GetTaskRunLogRequestSchema,
  TaskRunLogEntry_Type,
  TaskRunLogSchema,
  type TaskRunLog,
} from "../../../src/types/proto-es/v1/rollout_service_pb";
import {
  GetSettingRequestSchema,
  SettingSchema,
} from "../../../src/types/proto-es/v1/setting_service_pb";
import { BytebaseApiClient } from "../framework/api-client";
import { loadTestEnv } from "../framework/env";
import { execSql, getInstancePgPort, querySql, spawnPsql } from "../framework/psql";
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
async function stubAI(
  page: Page,
  enabled: boolean,
  plan: string,
  execution?: { stage: string; task: string }
) {
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
                    text: execution
                      ? "The Task Run recorded an execution error. Here is a proposed replacement:\n\n```sql replacement\nSELECT 1;\n```\n\nNo changes were executed."
                      : "The nullable column violates the configured SQL review rule. Add NOT NULL after checking existing rows. No changes were executed.",
                    success: true,
                    actions: [],
                  }
                : execution
                ? {
                    operationId: "bytebase.v1.RolloutService.BatchRunTasks",
                    body: { parent: execution.stage, tasks: [execution.task] },
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
      key.startsWith("bb-agent-state-v3.")
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

type ExecutionSurface =
  | "latest-failed-plain"
  | "latest-failed-release"
  | "historical-failed-run"
  | "latest-failed-replica"
  | "done-recorded-sync-error"
  | "create-database-failed";

const executionSurfaces: readonly ExecutionSurface[] = [
  "latest-failed-plain",
  "latest-failed-release",
  "historical-failed-run",
  "latest-failed-replica",
  "done-recorded-sync-error",
  "create-database-failed",
];

const executionViewports = [
  {
    label: "desktop-en-US",
    locale: "en-US",
    viewport: { width: 1440, height: 1000 },
    help: "Ask Agent",
    history: "History",
    firstRun: "Run #1",
  },
  {
    label: "mobile-es-ES",
    locale: "es-ES",
    viewport: { width: 390, height: 844 },
    help: "Preguntar al Agente",
    history: "Historial",
    firstRun: "Ejecución n.º 1",
  },
] as const;

async function createExecutionFixture(env: Env, surface: ExecutionSurface) {
  await env.api.login(env.adminEmail, env.adminPassword);
  const project = await env.api.getProject(env.project);
  const settings = {
    requireIssueApproval: !!project.requireIssueApproval,
    requirePlanCheckNoError: !!project.requirePlanCheckNoError,
    enforceSqlReview: !!project.enforceSqlReview,
    forceIssueLabels: !!project.forceIssueLabels,
    executionRetryPolicy: (project.executionRetryPolicy as
      | { maximumRetries?: number }
      | undefined) ?? { maximumRetries: 0 },
  };
  const version = String(Date.now());
  const schema = `agent_run_${version}`;
  const port = await getInstancePgPort(env);
  const createDatabase = surface === "create-database-failed";
  const databaseName = `agent_database_${version}`;
  const history = surface === "historical-failed-run";
  const releaseFile = `migrations/${version}02_break_records.sql`;
  const title = `Agent diagnosis: ${surface}`;
  let plan = "";
  let issue = "";
  let release = "";
  let lock: ReturnType<typeof spawnPsql> | undefined;
  const sql = (statement: string) => execSql(env.databaseId, port, statement);
  const query = (statement: string) => querySql(env.databaseId, port, statement);
  const cleanup = async () => {
    lock?.kill();
    const errors: unknown[] = [];
    for (const restore of [
      () => env.api.updateProjectSettings(env.project, settings),
      async () => { if (plan) await env.api.closePlan(plan); },
      async () => {
        if (!release) return;
        const revision = await env.api.findRevision(env.database, `${version}01`);
        if (revision) await env.api.deleteRevision(revision.name);
      },
      async () => { if (release) await env.api.deleteRelease(release); },
      async () => {
        if (createDatabase) {
          execSql("postgres", port, `DROP DATABASE IF EXISTS "${databaseName}";`);
          await env.api.syncInstance(env.instance);
        } else {
          sql(`DROP SCHEMA IF EXISTS ${schema} CASCADE;`);
        }
      },
      () => env.api.syncDatabase(env.database),
    ]) {
      try {
        await restore();
      } catch (error) {
        errors.push(error);
      }
    }
    if (errors.length) throw new AggregateError(errors, "Execution fixture cleanup failed");
  };
  try {
    await env.api.updateProjectSettings(env.project, {
      requireIssueApproval: false,
      requirePlanCheckNoError: false,
      enforceSqlReview: false,
      forceIssueLabels: false,
      executionRetryPolicy: { maximumRetries: history ? 6 : 0 },
    });
    if (!createDatabase) {
      sql(`CREATE SCHEMA ${schema}; CREATE SEQUENCE ${schema}.attempt;`);
      await env.api.syncDatabase(env.database);
    }
    if (surface === "latest-failed-release") {
      release = (await env.api.createRelease(env.project, {
        files: [
          { version: `${version}01`, path: `migrations/${version}01_add_records.sql`, statement: `CREATE TABLE ${schema}.records (id INT PRIMARY KEY); INSERT INTO ${schema}.records VALUES (7);` },
          { version: `${version}02`, path: releaseFile, statement: `ALTER TABLE ${schema}.missing ADD COLUMN diagnostic INT;` },
          { version: `${version}03`, path: `migrations/${version}03_add_later.sql`, statement: `CREATE TABLE ${schema}.later (id INT);` },
        ],
        vcsSource: { vcsType: "GITHUB", url: "https://github.com/example/app/commit/0a1b2c3d4e5f60718293a4b5c6d7e8f901234567" },
      })).name;
      plan = (await env.api.createReleasePlan(env.project, title, {
        id: "diagnostic-release", targets: [env.database], release,
      })).name;
    } else if (createDatabase) {
      plan = (await env.api.createCreateDatabasePlan(env.project, title, {
        id: "diagnostic-create",
        target: env.instance,
        database: databaseName,
        environment: "environments/prod",
        characterSet: "UTF8",
        collation: "e2e_missing_diagnostic_collation",
        owner: "bbsample",
      })).name;
      issue = (await env.api.createIssue(env.project, title, plan)).name;
    } else {
      const statement = surface === "done-recorded-sync-error"
        ? `CREATE TABLE ${schema}.records (id INT);`
        : [
            ...(history ? [`SELECT set_config('lock_timeout', CASE WHEN nextval('${schema}.attempt') = 1 THEN '2s' ELSE '0' END, false);`] : []),
            `ALTER TABLE ${schema}.records ADD COLUMN diagnostic INT;`,
          ].join("\n");
      const sheet = await env.api.createSheet(env.project, statement);
      plan = (await env.api.createPlan(env.project, title, [
        { id: "diagnostic-execution", targets: [env.database], sheet },
      ])).name;
    }
    if (!(await env.api.getPlan(plan)).hasRollout) {
      await env.api.createRollout(plan);
    }
    const stages = (await env.api.getRollout(plan)).stages ?? [];
    expect(stages).toHaveLength(1);
    expect(stages[0].tasks).toHaveLength(1);
    const stage = stages[0].name;
    const task = stages[0].tasks![0];
    const runs = () => env.api.listTaskRuns(task.name);
    const runAndWait = async (status: "DONE" | "FAILED") => {
      const before = new Set((await runs()).map((run) => run.name));
      await env.api.batchRunTasks(stage, [task.name]);
      await expect.poll(async () => (await runs()).filter((run) => !before.has(run.name)).map((run) => run.status), {
        timeout: 90_000,
      }).toEqual([status]);
      return (await runs()).find((run) => !before.has(run.name))!;
    };
    const selected = await runAndWait(surface === "done-recorded-sync-error" ? "DONE" : "FAILED");
    let latest = selected;
    if (history) {
      await runAndWait("FAILED");
      await runAndWait("FAILED");
      sql(`CREATE TABLE ${schema}.records (id INT); ALTER SEQUENCE ${schema}.attempt RESTART WITH 1;`);
      await env.api.syncDatabase(env.database);
      lock = spawnPsql(env.databaseId, port, `BEGIN; LOCK TABLE ${schema}.records IN ACCESS EXCLUSIVE MODE;`);
      await expect.poll(() => query(`SELECT count(*) FROM pg_locks WHERE relation = '${schema}.records'::regclass AND mode = 'AccessExclusiveLock' AND granted;`)).toBe("1");
      const previousNames = new Set((await runs()).map((run) => run.name));
      await env.api.batchRunTasks(stage, [task.name]);
      const newRunName = async () => (await runs()).find((run) => !previousNames.has(run.name))?.name;
      await expect.poll(async () => {
        const name = await newRunName();
        if (!name) return false;
        const log = await env.api.getTaskRunLog(name);
        return log.entries?.some((entry) => entry.type === "RETRY_INFO");
      }, { timeout: 60_000 }).toBe(true);
      lock.kill();
      lock = undefined;
      await expect.poll(async () => (await runs()).find((run) => !previousNames.has(run.name))?.status, { timeout: 90_000 }).toBe("DONE");
      latest = (await runs()).find((run) => !previousNames.has(run.name))!;
      expect((await env.api.getTaskRunLog(latest.name)).entries?.some(
        (entry) => entry.commandExecute?.response?.error?.includes("lock timeout")
      )).toBe(true);
    }
    if (release) {
      expect(await env.api.findRevision(env.database, `${version}01`)).toBeDefined();
    }
    const assertEffects = async () => {
      if (createDatabase) {
        expect(querySql("postgres", port, `SELECT count(*) FROM pg_database WHERE datname = '${databaseName}';`)).toBe("0");
      } else if (release) {
        expect(query(`SELECT id FROM ${schema}.records WHERE id = 7;`)).toBe("7");
        expect(query(`SELECT to_regclass('${schema}.later') IS NULL;`)).toBe("t");
        expect(await env.api.findRevision(env.database, `${version}02`)).toBeUndefined();
        expect(await env.api.findRevision(env.database, `${version}03`)).toBeUndefined();
      } else if (history) {
        expect(query(`SELECT count(*) FROM information_schema.columns WHERE table_schema = '${schema}' AND table_name = 'records' AND column_name = 'diagnostic';`)).toBe("1");
      } else {
        expect(query(`SELECT to_regclass('${schema}.records') IS NOT NULL;`))
          .toBe(surface === "done-recorded-sync-error" ? "t" : "f");
      }
    };
    await assertEffects();
    const expectedRuns = await runs();
    expect(expectedRuns).toHaveLength(history ? 4 : 1);
    return { plan, issue, title, stage, task, latest, selected, databaseName, releaseFile, version, cleanup, runs, expectedRuns, assertEffects };
  } catch (error) {
    await cleanup();
    throw error;
  }
}

const logMethod = "**/bytebase.v1.RolloutService/GetTaskRunLog";

async function observeExecutionLogs(
  page: Page,
  ownedRun: string,
  surface: ExecutionSurface
) {
  const names: string[] = [];
  const completedAt = timestampFromDate(new Date());
  let entries: TaskRunLog["entries"] | undefined;
  // Synthetic service-boundary fixtures: the harness has one real Postgres
  // instance, not replica failover or a database-sync fault injector.
  if (surface === "latest-failed-replica") {
    entries = create(TaskRunLogSchema, {
      entries: [
        {
          type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
          logTime: completedAt,
          replicaId: "fixtureA",
          commandExecute: {
            logTime: completedAt,
            statement: "SELECT 1;",
            response: { logTime: completedAt, error: "Synthetic replica command error" },
          },
        },
        {
          type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
          logTime: completedAt,
          replicaId: "fixtureB",
          commandExecute: {
            logTime: completedAt,
            statement: "SELECT 1;",
            response: { logTime: completedAt },
          },
        },
      ],
    }).entries;
  } else if (surface === "done-recorded-sync-error") {
    entries = create(TaskRunLogSchema, {
      entries: [{
        type: TaskRunLogEntry_Type.DATABASE_SYNC,
        logTime: completedAt,
        databaseSync: {
          startTime: completedAt,
          endTime: completedAt,
          error: "Synthetic recorded database synchronization error",
        },
      }],
    }).entries;
  }
  await page.route(logMethod, async (route) => {
    const request = route.request();
    const { parent } = decode(
      GetTaskRunLogRequestSchema,
      request.postDataBuffer()!,
      request.headers()["content-type"]?.includes("json") ?? false
    );
    names.push(parent);
    if (parent !== ownedRun || !entries) return route.continue();
    await fulfill(route, TaskRunLogSchema, create(TaskRunLogSchema, {
      name: `${parent}/log`, entries,
    }));
  });
  return names;
}

async function captureExecution(
  page: Page,
  testInfo: TestInfo,
  name: string,
  target?: Locator
) {
  const file = `agent-task-run-${name}.png`;
  const directory = process.env.BYTEBASE_E2E_SCREENSHOT_DIR;
  await (target ?? page).screenshot({
    animations: "disabled",
    path: directory ? join(directory, file) : testInfo.outputPath(file),
  });
}

for (const viewport of executionViewports) {
  for (const surface of executionSurfaces) {
    test(`${viewport.label} ${surface} opens a read-only Task Run diagnosis`, async ({
      page,
    }, testInfo) => {
      const env = loadTestEnv();
      const fixture = await createExecutionFixture(env, surface);
      try {
        await page.setViewportSize(viewport.viewport);
        await page.addInitScript(
          (locale) => localStorage.setItem("bb.language", locale),
          viewport.locale
        );
        const logRequests = await observeExecutionLogs(page, fixture.selected.name, surface);
        const requests = await stubAI(page, true, fixture.plan, {
          stage: fixture.stage, task: fixture.task.name,
        });
        const wrappers = page.getByTestId("task-run-diagnostic-help");
        const history = surface === "historical-failed-run";
        const viewer = page.getByTestId("task-run-log-viewer");
        if (fixture.issue) {
          await page.goto(`${env.baseURL}/${fixture.issue}`);
          await expect(page.getByText(fixture.databaseName, { exact: true })).toBeVisible();
        } else {
          await page.goto(`${env.baseURL}/${fixture.task.name}`);
          await new PlanDetailPage(page, env.baseURL).dismissModals();
          if (history) {
            await expect(viewer.getByTestId("task-run-log-previous-attempts")).toBeVisible();
            await expect(wrappers).toHaveCount(0);
            await viewer.scrollIntoViewIfNeeded();
            await captureExecution(page, testInfo, `${viewport.label}-recovered-no-action`);
            await page.getByRole("button", {
              name: `${viewport.history} (4)`,
              exact: true,
            }).click();
            const sheet = page.getByRole("dialog", { name: viewport.history });
            await expect(sheet).toBeVisible();
            await expect(sheet.getByRole("button", { expanded: false }).filter({
              hasText: viewport.firstRun,
            })).toBeVisible();
            expect([...new Set(logRequests)]).toEqual([fixture.latest.name]);
            await expect(wrappers).toHaveCount(0);
            await sheet.getByRole("button", { name: new RegExp(viewport.firstRun) }).click();
            await expect(sheet.getByTestId("task-run-log-viewer")).toHaveCount(2);
            await expect.poll(() => new Set(logRequests)).toEqual(
              new Set([fixture.latest.name, fixture.selected.name])
            );
          } else if (surface === "latest-failed-release") {
            const spine = page.getByTestId("release-spine");
            await expect(spine.getByTestId(`spine-file-${fixture.version}01`)).toBeVisible();
            await expect(spine.getByTestId(`spine-file-${fixture.version}02`)).toHaveAttribute("aria-expanded", "true");
            await expect(spine.getByTestId(`spine-file-${fixture.version}03`)).toHaveCount(0);
          } else {
            await expect(viewer).toBeVisible();
            if (surface === "latest-failed-replica") {
              await expect(viewer.getByRole("button", { name: /fixtureA/ })).toBeVisible();
              await expect(viewer.getByRole("button", { name: /fixtureB/ })).toBeVisible();
            }
          }
        }
        await expect(wrappers).toHaveCount(1);
        await expect(wrappers).toHaveAttribute("data-task-run-name", fixture.selected.name);
        if (fixture.issue) {
          const row = page.getByRole("row").filter({ has: wrappers });
          await expect(row.getByRole("cell")).toHaveCount(5);
          await expect(row.getByRole("cell").nth(1).getByTestId("task-run-diagnostic-help")).toHaveCount(1);
          expect(logRequests).toEqual([]);
        }
        const help = wrappers.getByRole("button", { name: viewport.help, exact: true });
        await expect(help).toHaveCount(1);
        await assertFits(page, help);
        expect(await help.evaluate((button) => {
          const bounds = button.getBoundingClientRect();
          return [...button.children].every((child) => {
            const content = child.getBoundingClientRect();
            return content.left >= bounds.left - 1 && content.right <= bounds.right + 1 && child.scrollWidth <= child.clientWidth + 1;
          });
        })).toBe(true);
        if (fixture.issue) {
          const row = page.getByRole("row").filter({ has: wrappers });
          const cells = row.getByRole("cell");
          const detailCell = cells.nth(1);
          const comment = detailCell.getByText(fixture.selected.detail!, { exact: true });
          await expect(comment).toBeVisible();
          const commentBounds = (await comment.boundingBox())!;
          const helpBounds = (await help.boundingBox())!;
          const detailBounds = (await detailCell.boundingBox())!;
          const rowBounds = (await row.boundingBox())!;
          expect(Math.abs(commentBounds.y + commentBounds.height / 2 - helpBounds.y - helpBounds.height / 2)).toBeLessThanOrEqual(1);
          expect(commentBounds.x + commentBounds.width).toBeLessThanOrEqual(helpBounds.x);
          const paddingRight = await detailCell.evaluate((cell) => parseFloat(getComputedStyle(cell).paddingRight));
          expect(Math.abs(detailBounds.x + detailBounds.width - paddingRight - helpBounds.x - helpBounds.width)).toBeLessThanOrEqual(1);
          const baselineRowHeight = await wrappers.evaluate((wrapper) => {
            const display = wrapper.style.display;
            wrapper.style.display = "none";
            const height = wrapper.closest("tr")!.getBoundingClientRect().height;
            wrapper.style.display = display;
            return height;
          });
          expect(rowBounds.height).toBeLessThanOrEqual(baselineRowHeight + 1);
          for (const timestamp of [cells.nth(2), cells.nth(3)]) {
            expect((await timestamp.boundingBox())!.width).toBe(TIMESTAMP_COLUMN_WIDTH.compact);
          }
          expect(await comment.evaluate((node) => node.scrollWidth > node.clientWidth)).toBe(true);
          const iconOnly = viewport.viewport.width < 768;
          await expect(help).toHaveText(iconOnly ? "" : viewport.help);
          await expect(help).toHaveAttribute("title", viewport.help);
          if (iconOnly) {
            const hitArea = await help.evaluate((button) => {
              const target = getComputedStyle(button, "::after");
              return { width: parseFloat(target.width), height: parseFloat(target.height) };
            });
            expect(hitArea.width).toBeGreaterThanOrEqual(44);
            expect(hitArea.height).toBeGreaterThanOrEqual(44);
          }
          await captureExecution(page, testInfo, `${viewport.label}-create-database-failed-inline-table`, row.locator("xpath=ancestor::table/.."));
        }
        await help.focus();
        await expect(help).toBeFocused();
        const synthetic = surface === "latest-failed-replica" || surface === "done-recorded-sync-error";
        await captureExecution(page, testInfo, `${viewport.label}-${synthetic ? "synthetic-" : ""}${surface}`);

        const mutationRequests: string[] = [];
        page.on("request", (request) => {
          const pathname = new URL(request.url()).pathname;
          if (/\/(?:Create|Update|Delete|BatchRun|BatchSkip|BatchCancel|Approve|Reject|Query)[^/]*$/.test(pathname)) {
            mutationRequests.push(pathname);
          }
        });
        const pathname = new URL(page.url()).pathname;
        await page.keyboard.press("Enter");
        const agent = page.locator("[data-agent-window]");
        await expect(agent.getByText("No changes were executed.", { exact: false })).toBeVisible({ timeout: 30_000 });
        await expect(agent.getByText(/Allow the agent to run this API request|Permitir que el agente ejecute/i)).toHaveCount(0);
        expect(new URL(page.url()).pathname).toBe(pathname);
        if (fixture.issue) expect(pathname).toBe(`/${fixture.issue}`);
        await expect(agent.getByRole("button", {
          name: /request permission|solicitar permiso|review sql|revisar(?: corrección)? sql|apply to draft|aplicar al borrador|request access|solicitar (?:acceso|concesión)|retry|reintentar/i,
        })).toHaveCount(0);
        expect(requests).toHaveLength(2);
        expect(requests[1].messages.find((message) => message.toolCallId === "attempt-mutation")?.content)
          .toContain("not available in a diagnostic chat");
        const prompt = requests[0].messages.find(
          (message) => message.role === AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_USER
        )!.content!;
        const snapshot = JSON.parse(
          prompt.split("\n").find((line) => line.startsWith('{"resource":'))!
        );
        const resource = {
          kind: fixture.issue ? "issue" : "plan",
          name: fixture.issue || fixture.plan,
          project: env.project,
        };
        expect(snapshot).toMatchObject({
          resource, kind: "task-run", sourceId: fixture.selected.name,
        });
        expect(snapshot).not.toHaveProperty("sqlFixSource");
        expect(snapshot.facts).toEqual(expect.arrayContaining([
          { label: "Status", value: fixture.selected.status },
          { label: "Provenance", value: history ? "explicit-history" : "latest" },
          { label: "Recorded target", value: fixture.task.target },
        ]));
        if (surface === "latest-failed-release") {
          expect(snapshot.facts).toEqual(expect.arrayContaining([
            { label: "Recorded file", value: fixture.releaseFile },
            { label: "Recorded version", value: `${fixture.version}02` },
          ]));
        }
        if (surface === "latest-failed-replica") {
          expect(snapshot.facts).toContainEqual({
            label: "Logs", value: "Final snapshot available; replica attribution ambiguous",
          });
          expect(snapshot.facts).not.toContainEqual(
            expect.objectContaining({ label: "Recorded replica" })
          );
        }
        if (surface === "done-recorded-sync-error") {
          expect(snapshot.facts).toContainEqual({ label: "Recorded error type", value: "DATABASE_SYNC" });
          expect(snapshot.detail).toContain("Synthetic recorded database synchronization error");
        }
        await expect.poll(async () => {
          const state = await persistedAgent(page);
          const chat = state?.chats.find((item) => item.id === state.currentChatId);
          const user = chat && state?.messagesByChatId[chat.id]?.find((message) => message.role === "user");
          return {
            capability: chat?.capability,
            prompt: user?.content,
            selected: user?.metadata?.resourceReference?.name ?? user?.metadata?.resourceReferences?.[0]?.name,
          };
        }).toMatchObject({
          capability: { kind: "diagnostic", resource },
          prompt: expect.stringContaining(fixture.selected.name),
          selected: resource.name,
        });
        expect(mutationRequests).toEqual([]);
        expect(await fixture.runs()).toEqual(fixture.expectedRuns);
        expect((await env.api.getPlan(fixture.plan)).title).toBe(fixture.title);
        await fixture.assertEffects();
        if (history) {
          await captureExecution(page, testInfo, `${viewport.label}-historical-run-diagnostic-chat`);
        }
      } finally {
        await fixture.cleanup();
      }
    });
  }
}
