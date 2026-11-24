import { writeFile } from "node:fs/promises";
import {
  expect,
  type Locator,
  type Page,
  type Request,
  type TestInfo,
  test,
} from "@playwright/test";
import {
  type BareServer,
  bareServerStartTimeoutMs,
  startBareServer,
} from "../framework/bare-server";
import { loginBrowserContext } from "../framework/sign-in";
import {
  databasePatch,
  DomMutationAgent,
  type DomStep,
  observeFocus,
  watchDatabaseWrites,
} from "./dom-mutation-confirmation.page";

test.use({ storageState: { cookies: [], origins: [] } });
test.setTimeout(90_000);

const baseline = { seed: "baseline" };
const projectId = "agent-dom-fixture";
const dev = "environments/dom-dev";
const prod = "environments/dom-prod";
const sql = "SELECT 41 AS agent_dom_fixture;";
let server: BareServer;
let token: string;
let workspace: string;
let project: string;
let instance: string;
let database: string;
const entitlementRequests = new WeakMap<Page, string[]>();

type DatabaseState = {
  name: string;
  labels?: Record<string, string>;
  environment?: string;
  effectiveEnvironment?: string;
};

async function api<T>(
  method: string,
  path: string,
  body?: unknown
): Promise<T> {
  const response = await fetch(`${server.baseURL}/v1/${path}`, {
    method,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  });
  if (!response.ok)
    throw new Error(
      `Fixture API ${method} ${path}: ${response.status} ${await response.text()}`
    );
  return response.json() as Promise<T>;
}

const readDatabase = () => api<DatabaseState>("GET", database);
const databaseURL = () =>
  `${server.baseURL}/${database}?parent=${encodeURIComponent(instance)}#setting`;

test.beforeAll(async () => {
  test.setTimeout(bareServerStartTimeoutMs());
  server = await startBareServer();
  token = await server.api.login(server.adminEmail, server.adminPassword);
  ({ workspace } = await server.api.getActuatorInfo());
  ({ name: project } = await server.api.createProject(
    projectId,
    "Agent DOM Fixture"
  ));
  ({ name: instance } = await server.api.prepareSampleProjectInstance(project));
  database = `${instance}/databases/hr_test`;
  await expect.poll(async () => (await readDatabase()).name).toBe(database);
  await server.api.setWorkspaceExternalUrl(server.baseURL);
  await server.api.upsertSetting(
    "AI",
    {
      ai: {
        enabled: true,
        provider: "OPEN_AI",
        endpoint: "http://127.0.0.1:1",
        model: "deterministic-fixture",
        apiKey: "fixture-not-a-credential",
      },
    },
    "value.ai.enabled,value.ai.provider,value.ai.endpoint,value.ai.model,value.ai.api_key"
  );
  await server.api.upsertSetting(
    "ENVIRONMENT",
    {
      environment: {
        environments: [
          { id: "dom-dev", title: "DOM Development" },
          { id: "dom-prod", title: "DOM Production" },
        ],
      },
    },
    "value.environment"
  );
});

test.afterAll(async () => {
  await server?.stop();
});

test.beforeEach(async ({ page }) => {
  const requests: string[] = [];
  entitlementRequests.set(page, requests);
  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    if (path.includes("EntitlementService") || path.includes("entitlements"))
      requests.push(path);
  });
  await api("PATCH", `${instance}?updateMask=environment`, {
    name: instance,
    environment: dev,
  });
  await api("PATCH", `${database}?updateMask=labels,environment`, {
    name: database,
    labels: baseline,
    environment: "",
  });
  expect((await server.api.getSubscription()).plan).toBe("FREE");
});

test.afterEach(async ({ page }) => {
  expect(
    entitlementRequests.get(page),
    "DOM safety needs no entitlement service"
  ).toEqual([]);
});

async function open(
  page: Page,
  path = databaseURL(),
  email = server.adminEmail
) {
  const agent = new DomMutationAgent(page);
  await agent.install(page.context());
  await loginBrowserContext(
    page.context(),
    server.baseURL,
    email,
    server.adminPassword
  );
  await page.addInitScript(() => {
    localStorage.setItem(
      "bb.release",
      JSON.stringify({
        ignoreRemindModalTillNextRelease: true,
        nextCheckTs: Date.now() + 86_400_000,
      })
    );
  });
  await page.goto(`${server.baseURL}/landing`);
  await agent.open();
  const destination = new URL(path);
  const initial = agent.navigate(
    `${destination.pathname}${destination.search}${destination.hash}`
  );
  await agent.send(
    "Open the fixture page in this ordinary workspace chat without a resource reference."
  );
  await initial.done();
  await expect(page).toHaveURL(path);
  if (path === databaseURL())
    await expect(
      page.getByPlaceholder("Label value...", { exact: true })
    ).toHaveValue("baseline");
  await agent.open();
  await expect(agent.window.locator("[data-agent-answer]")).toContainText(
    initial.text
  );
  return agent;
}

const inputValue = (value: string, next: string, name = "input"): DomStep => ({
  name,
  match: new RegExp(`<input value="${value}">.*input:local-draft$`),
  type: "input",
  value: next,
});
const button = (
  label: string,
  name = label,
  effect = "local-draft"
): DomStep => ({
  name,
  match: new RegExp(`<button>${label}</button> click:${effect}$`),
  type: "click",
});
const selectEnvironment = (value: string): DomStep => ({
  name: "environment",
  match: /select:persist$/,
  type: "select",
  value,
});

async function pendingLabels(
  agent: DomMutationAgent,
  value = "agent-approved",
  replay = false
) {
  const run = agent.queue([
    inputValue("baseline", value),
    button("Save", "save", "persist"),
    ...(replay
      ? [
          {
            name: "replay",
            match: /unused/,
            type: "click" as const,
            reuse: "save",
          },
        ]
      : []),
  ]);
  await agent.send("Prepare the exact database label update for my review.");
  await expect(agent.allow).toBeVisible();
  await expect(agent.allow).toBeEnabled();
  return run;
}

function expectLabelsPatch(request: Request, labels: Record<string, string>) {
  expect(databasePatch(request)).toMatchObject({
    database: { name: database, labels },
    paths: ["labels"],
  });
  const patch = databasePatch(request).database;
  expect(Object.keys(patch ?? {}).sort()).toEqual(["labels", "name"]);
}

async function expectPreviewContained(element: Locator, container: Locator) {
  await expect(container).toBeVisible();
  await expect(element).toBeVisible();
  await expect.poll(async () => {
    const bounds = await element.boundingBox();
    const parent = await container.boundingBox();
    return bounds !== null && parent !== null &&
      bounds.x >= parent.x - 1 &&
      bounds.x + bounds.width <= parent.x + parent.width + 1 &&
      bounds.y >= parent.y - 1 &&
      bounds.y + bounds.height <= parent.y + parent.height + 1;
  }).toBe(true);
  const bounds = await element.boundingBox();
  const parent = await container.boundingBox();
  expect(bounds).not.toBeNull();
  expect(parent).not.toBeNull();
  expect(bounds!.x).toBeGreaterThanOrEqual(parent!.x - 1);
  expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(parent!.x + parent!.width + 1);
  expect(bounds!.y).toBeGreaterThanOrEqual(parent!.y - 1);
  expect(bounds!.y + bounds!.height).toBeLessThanOrEqual(parent!.y + parent!.height + 1);
  expect(await element.evaluate((node) => node.scrollWidth - node.clientWidth)).toBeLessThanOrEqual(1);
}

async function captureMobilePermission(
  page: Page,
  agent: DomMutationAgent,
  testInfo: TestInfo,
  kind: "labels" | "environment",
  requestValue: string
) {
  const originalViewport = page.viewportSize();
  const scenarios: {
    title: string;
    width: number;
    screenshots: { file: string; label: string }[];
  }[] = [];
  const exact = agent.window.locator("details").filter({ hasText: "Exact API request" });
  for (const width of [320, 390]) {
    await page.setViewportSize({ width, height: 900 });
    await expect(agent.window).toBeVisible();
    await expect(agent.allow).toBeVisible();
    await expect(agent.deny).toBeVisible();
    await expect(agent.allow).toBeEnabled();
    await expectPreviewContained(agent.allow, agent.window);
    await expectPreviewContained(agent.deny, agent.window);
    const actions = agent.allow.locator("xpath=ancestor::*[@data-agent-action-group][1]");
    await expect(actions).toHaveCSS("flex-wrap", "wrap");
    await expect.poll(async () => {
      const group = await actions.boundingBox();
      const last = await actions.getByRole("button").last().boundingBox();
      return group !== null && last !== null && Math.abs(last.x + last.width - group.x - group.width) <= 1;
    }).toBe(true);
    const screenshots = [];
    const diffFile = `${kind}-diff-${width}.png`;
    await agent.window.screenshot({ path: testInfo.outputPath(diffFile) });
    screenshots.push({ file: diffFile, label: `${kind} before-and-after permission preview` });
    await exact.locator("summary").click();
    await expect(exact.locator("pre")).toContainText(requestValue);
    await expect(exact.locator("pre")).toContainText(database);
    await exact.locator("pre").scrollIntoViewIfNeeded();
    await expectPreviewContained(exact.locator("pre"), agent.window);
    await expectPreviewContained(agent.allow, agent.window);
    await expectPreviewContained(agent.deny, agent.window);
    const exactFile = `${kind}-exact-request-${width}.png`;
    await agent.window.screenshot({ path: testInfo.outputPath(exactFile) });
    screenshots.push({ file: exactFile, label: `${kind} exact API request` });
    await exact.locator("summary").click();
    scenarios.push({ title: `${kind} permission preview`, width, screenshots });
    await writeFile(testInfo.outputPath("screenshots.json"), JSON.stringify({ scenarios }, null, 2));
  }
  if (originalViewport) await page.setViewportSize(originalViewport);
}

test("labels stay local and focus-preserving until one exact approved write", async ({
  page,
}, testInfo) => {
  const agent = await open(page);
  const writes = watchDatabaseWrites(page, database);
  await page.clock.install();
  // Each action needs a fresh snapshot; two user turns respect the real
  // per-run provider-attempt cap without relaxing the consent boundary.
  const localTurns: DomStep[][] = [
    [
      inputValue("baseline", "local-only"),
      button("Add label"),
      {
        name: "new-key",
        match: /<input>.*input:local-draft$/,
        index: 0,
        type: "input",
        value: "temporary",
      },
    ],
    [
      {
        name: "new-value",
        match: /<input>.*input:local-draft$/,
        index: 0,
        type: "input",
        value: "temporary-value",
      },
      {
        name: "remove",
        match: /<button>Delete<\/button> click:local-draft$/,
        index: 1,
        type: "click",
      },
      button("Revert"),
    ],
  ];
  const localNetwork: Request[] = [];
  const snapshotReads: Request[] = [];
  let localActionActive = false;
  const observeLocalNetwork = (request: Request) => {
    const path = new URL(request.url()).pathname;
    if (
      (path.startsWith("/v1/") || path.startsWith("/bytebase.v1.")) &&
      !/\/(?:Chat|ai\/chat)$/.test(path)
    )
      (localActionActive ? localNetwork : snapshotReads).push(request);
  };
  for (const steps of localTurns) {
    const local = agent.queue(
      steps.map((step) => ({
        ...step,
        before: async () => {
          localActionActive = true;
        },
        after: async () => {
          await page.clock.fastForward(3_000);
          localActionActive = false;
        },
      })),
      true
    );
    await agent.send("Continue only the local label draft; do not save it.");
    await local.ready;
    page.on("request", observeLocalNetwork);
    const expectFocusUnchanged = await observeFocus(page);
    local.release();
    await local.done();
    page.off("request", observeLocalNetwork);
    await expectFocusUnchanged();
    for (const id of Object.values(local.ids))
      expect(agent.results.get(id)).toMatchObject({ success: true });
  }
  expect(writes).toHaveLength(0);
  expect(
    localNetwork.map((request) => new URL(request.url()).pathname)
  ).toEqual([]);
  expect(
    snapshotReads.map((request) => new URL(request.url()).pathname)
  ).toEqual(
    Array.from(
      { length: localTurns.flat().length },
      () => "/bytebase.v1.DatabaseService/GetDatabase"
    )
  );
  expect((await readDatabase()).labels).toEqual(baseline);
  await expect(
    page.getByPlaceholder("Label value...", { exact: true })
  ).toHaveValue("baseline");

  const run = await pendingLabels(agent, "agent-approved", true);
  await expect(agent.window).toContainText(database);
  await expect(
    agent.window.getByRole("list", { name: "Update database labels" })
  ).toContainText("seed");
  await expect(agent.window).toContainText("baseline");
  await expect(agent.window).toContainText("agent-approved");
  expect(writes).toHaveLength(0);
  expect((await readDatabase()).labels).toEqual(baseline);
  await page.screenshot({
    path: testInfo.outputPath("labels-desktop.png"),
    fullPage: true,
  });
  await captureMobilePermission(page, agent, testInfo, "labels", "agent-approved");
  expect(writes).toHaveLength(0);
  expect((await readDatabase()).labels).toEqual(baseline);
  await agent.allow.dblclick();
  await run.done();
  expect(writes).toHaveLength(1);
  expectLabelsPatch(writes[0], { seed: "agent-approved" });
  expect((await readDatabase()).labels).toEqual({ seed: "agent-approved" });
  await expect(agent.allow).toBeHidden();
  expect(agent.results.get(run.ids.save)).toMatchObject({ status: 200 });
  expect(agent.results.get(run.ids.replay)).toMatchObject({ success: false });
});

test("environment select prepares without native change, inherited selection is a no-op", async ({
  page,
}, testInfo) => {
  const agent = await open(page);
  const writes = watchDatabaseWrites(page, database);
  const unchanged = agent.queue([selectEnvironment(dev)]);
  await agent.send(
    "Keep the database's effective inherited development environment."
  );
  await unchanged.done();
  expect(agent.results.get(unchanged.ids.environment)).toMatchObject({
    success: true,
  });
  await expect(agent.allow).toBeHidden();
  expect(writes).toHaveLength(0);
  expect((await readDatabase()).environment ?? "").toBe("");

  const run = agent.queue([selectEnvironment(prod)]);
  await agent.send("Prepare the production environment update for review.");
  await expect(agent.allow).toBeVisible();
  await expect(
    page.getByText("DOM Development", { exact: true }).first()
  ).toBeVisible();
  expect(writes).toHaveLength(0);
  expect((await readDatabase()).environment ?? "").toBe("");
  await captureMobilePermission(page, agent, testInfo, "environment", prod);
  expect(writes).toHaveLength(0);
  expect((await readDatabase()).environment ?? "").toBe("");
  await agent.allow.click();
  await run.done();
  expect(writes).toHaveLength(1);
  expect(databasePatch(writes[0])).toEqual({
    database: { name: database, environment: prod },
    paths: ["environment"],
  });
  expect((await readDatabase()).environment).toBe(prod);
});

test("an unknown portaled environment option cannot invoke native autosave", async ({
  page,
}) => {
  const agent = await open(page);
  const writes = watchDatabaseWrites(page, database);
  const run = agent.queue(
    [
      {
        name: "option",
        match: /<button>DOM Production.*<\/button> manual-required$/,
        type: "click",
      },
    ],
    true
  );
  await agent.send("Try the visible production option through its page ref.");
  await run.ready;
  await page.getByText("DOM Development", { exact: true }).last().click();
  await expect(
    page.getByRole("option", { name: /DOM Production/ })
  ).toBeVisible();
  const expectFocusUnchanged = await observeFocus(page);
  run.release();
  await run.done();
  await expectFocusUnchanged();
  expect(agent.results.get(run.ids.option)).toMatchObject({
    success: false,
    code: "manual-required",
  });
  await expect(
    page.getByRole("option", { name: /DOM Production/ })
  ).toBeVisible();
  expect(writes).toHaveLength(0);
  expect((await readDatabase()).environment ?? "").toBe("");

  const trigger = agent.queue([
    { name: "trigger", match: /select:persist$/, type: "click" },
    selectEnvironment(""),
  ]);
  await agent.send(
    "Do not open the picker or clear an inherited environment through an unregistered action."
  );
  await trigger.done();
  expect(agent.results.get(trigger.ids.trigger)).toMatchObject({
    success: false,
    code: "manual-required",
  });
  expect(agent.results.get(trigger.ids.environment)).toMatchObject({
    success: false,
    code: "manual-required",
  });
  expect(writes).toHaveLength(0);
  expect((await readDatabase()).environment ?? "").toBe("");
});

test("semantic environment clear needs one permission when no instance environment is inherited", async ({
  page,
}) => {
  await api("PATCH", `${instance}?updateMask=environment`, {
    name: instance,
    environment: "",
  });
  await api("PATCH", `${database}?updateMask=environment`, {
    name: database,
    environment: prod,
  });
  const agent = await open(page);
  const writes = watchDatabaseWrites(page, database);
  const run = agent.queue([selectEnvironment("")]);
  await agent.send("Prepare clearing the database's explicit environment.");
  await expect(agent.allow).toBeEnabled();
  expect(writes).toHaveLength(0);
  expect((await readDatabase()).environment).toBe(prod);
  await agent.allow.click();
  await run.done();
  expect(writes).toHaveLength(1);
  const patch = databasePatch(writes[0]);
  expect(patch.paths).toEqual(["environment"]);
  expect(patch.database?.name).toBe(database);
  expect(patch.database?.environment ?? "").toBe("");
  expect(patch.database).toEqual({ name: database, environment: "" });
  expect((await readDatabase()).environment ?? "").toBe("");
});

test("denying a label permission preserves the draft and sends no write", async ({
  page,
}) => {
  const agent = await open(page);
  const writes = watchDatabaseWrites(page, database);
  const run = await pendingLabels(agent, "denied-draft");
  await agent.deny.click();
  await run.done();
  expect(agent.results.get(run.ids.save)).toMatchObject({
    error: expect.stringMatching(/denied/i),
  });
  expect(writes).toHaveLength(0);
  expect((await readDatabase()).labels).toEqual(baseline);
  await expect(
    page.getByPlaceholder("Label value...", { exact: true })
  ).toHaveValue("denied-draft");
});

test("archiving a fallback chat cannot revive its pending label permission after restore", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1600, height: 1100 });
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "locks", {
      configurable: true,
      value: undefined,
    });
  });
  const agent = await open(page);
  expect(await page.evaluate(() => typeof navigator.locks?.request)).toBe(
    "undefined"
  );
  const writes = watchDatabaseWrites(page, database);
  const value = "archived-draft";
  await pendingLabels(agent, value);
  expect(writes).toHaveLength(0);
  expect((await readDatabase()).labels).toEqual(baseline);

  type SavedState = {
    currentChatId?: string;
    chats?: {
      id: string;
      runId?: string | null;
      runBudget?: unknown;
      archived?: boolean;
    }[];
    pendingAskByChatId?: Record<
      string,
      { apiPermission?: { actionId?: string } }
    >;
  };
  const workspaceId = workspace.split("/").at(-1)!;
  const stateKey = `bb-agent-state-v2.${workspaceId.length}:${workspaceId}.${server.adminEmail}`;
  const saved = (): Promise<SavedState> =>
    page.evaluate(
      (key) => JSON.parse(localStorage.getItem(key) || "{}"),
      stateKey
    );
  await expect
    .poll(async () => {
      const state = await saved();
      return state.currentChatId
        ? state.pendingAskByChatId?.[state.currentChatId]?.apiPermission
            ?.actionId
        : undefined;
    })
    .toBeTruthy();
  const before = await saved();
  const chatId = before.currentChatId!;
  const chat = before.chats?.find((chat) => chat.id === chatId);
  expect(chat?.runId).toBeTruthy();
  expect(chat?.runBudget).toBeFalsy();

  const row = agent.window.locator(`[data-agent-chat-row="${chatId}"]`);
  if (!(await row.isVisible()))
    await agent.window
      .getByRole("button", { name: "Chats", exact: true })
      .click();
  await expect(row).toBeVisible();
  await row.hover();
  await row.locator("[data-agent-chat-menu]").click();
  await page.locator("[data-agent-archive-chat]").click();
  await page
    .locator("[data-agent-archive-chat-confirmation]")
    .getByRole("button", { name: "Archive", exact: true })
    .click();
  await expect
    .poll(
      async () =>
        (await saved()).chats?.find((chat) => chat.id === chatId)?.archived
    )
    .toBe(true);
  expect(writes).toHaveLength(0);

  await agent.window
    .getByRole("button", { name: "View archived chats", exact: true })
    .click();
  await expect(row).toBeVisible();
  await row.locator("button").first().click();
  const restore = agent.window.getByRole("button", {
    name: "Restore chat",
    exact: true,
  });
  await expect(restore).toBeVisible();
  await restore.click();
  await expect(restore).toBeHidden();
  await expect
    .poll(async () => {
      const state = await saved();
      return {
        currentChatId: state.currentChatId,
        archived: state.chats?.find((chat) => chat.id === chatId)?.archived,
      };
    })
    .toEqual({ currentChatId: chatId, archived: false });
  await expect(agent.allow).toBeHidden();
  await expect
    .poll(async () => (await saved()).pendingAskByChatId?.[chatId])
    .toBeUndefined();
  await expect
    .poll(
      async () =>
        (await saved()).chats?.find((chat) => chat.id === chatId)?.runId
    )
    .toBeNull();
  await expect(page).toHaveURL(databaseURL());
  await expect(
    page.getByPlaceholder("Label value...", { exact: true })
  ).toHaveValue(value);
  expect(writes).toHaveLength(0);
  expect((await readDatabase()).labels).toEqual(baseline);

  // The cancelled turn's remaining provider response must not script the new turn.
  const fresh = new DomMutationAgent(page);
  await fresh.install(page.context());
  const run = fresh.queue([button("Save", "save", "persist")]);
  await fresh.send("Prepare saving the same preserved database label draft.");
  await expect(fresh.allow).toBeVisible();
  await expect(fresh.allow).toBeEnabled();
  expect(writes).toHaveLength(0);
  expect((await readDatabase()).labels).toEqual(baseline);
  await fresh.allow.click();
  await run.done();
  expect(writes).toHaveLength(1);
  expectLabelsPatch(writes[0], { seed: value });
  expect((await readDatabase()).labels).toEqual({ seed: value });
  expect(agent.errors).toEqual([]);
  expect(agent.unexpectedCalls).toBe(0);
});

for (const invalidation of [
  "expiry",
  "newer-draft",
  "route-unmount",
] as const) {
  test(`${invalidation} invalidates a pending label permission without writing`, async ({
    page,
  }) => {
    if (invalidation === "expiry") await page.clock.install();
    const agent = await open(page);
    const writes = watchDatabaseWrites(page, database);
    await pendingLabels(agent);
    if (invalidation === "expiry") await page.clock.fastForward(300_001);
    else if (invalidation === "newer-draft")
      await page
        .getByPlaceholder("Label value...", { exact: true })
        .fill("newer-human-draft");
    else await page.getByRole("tab", { name: "Overview", exact: true }).click();
    await expect(agent.allow).toBeDisabled();
    await expect(agent.dismiss).toBeVisible();
    await agent.dismiss.click();
    await expect(agent.dismiss).toBeHidden();
    await expect(agent.window).toContainText(
      "This request is no longer available."
    );
    expect(writes).toHaveLength(0);
    expect((await readDatabase()).labels).toEqual(baseline);
    if (invalidation === "newer-draft")
      await expect(
        page.getByPlaceholder("Label value...", { exact: true })
      ).toHaveValue("newer-human-draft");
  });
}

test("native Save during a pending permission persists once without spending the Agent permission", async ({
  page,
}) => {
  const agent = await open(page);
  const writes = watchDatabaseWrites(page, database);
  await pendingLabels(agent, "human-saved");
  await agent.minimize();
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect
    .poll(async () => (await readDatabase()).labels)
    .toEqual({ seed: "human-saved" });
  expect(writes).toHaveLength(1);
  expect(writes[0].headers()["content-type"]).toMatch(/^application\/proto/);
  expectLabelsPatch(writes[0], { seed: "human-saved" });
  await agent.open();
  await expect(agent.allow).toBeDisabled();
  await agent.dismiss.click();
  await expect(agent.dismiss).toBeHidden();
  expect(writes).toHaveLength(1);
  expect((await readDatabase()).labels).toEqual({ seed: "human-saved" });
});

test("fresh backend IAM revocation blocks an approved pending mutation", async ({
  page,
}) => {
  const email = "dom-owner@example.com";
  await server.api.createUser(email, server.adminPassword, "DOM Owner Fixture");
  await server.api.appendProjectBinding(project, "roles/projectOwner", [
    `user:${email}`,
  ]);
  const agent = await open(page, databaseURL(), email);
  const writes = watchDatabaseWrites(page, database);
  const run = await pendingLabels(agent, "revoked-draft");
  const policy = await server.api.getProjectIamPolicy(project);
  policy.bindings = policy.bindings.map((binding) => ({
    ...binding,
    members: binding.members.filter((member) => member !== `user:${email}`),
  }));
  await server.api.setProjectIamPolicy(project, policy);
  expect(
    (await server.api.getProjectIamPolicy(project)).bindings.flatMap(
      (binding) => binding.members
    )
  ).not.toContain(`user:${email}`);
  await agent.allow.click();
  await run.done();
  expect(agent.results.get(run.ids.save)).toMatchObject({
    called: false,
    error: expect.stringMatching(/^Not called:/),
  });
  expect(writes).toHaveLength(0);
  expect((await readDatabase()).labels).toEqual(baseline);
});

test("Plan local Edit, input and Cancel preserve the warm reader and emit no SQL protocol or persistence traffic", async ({
  page,
}, testInfo) => {
  const sheet = await server.api.createSheet(project, sql);
  expect(sheet).toMatch(new RegExp(`^${project}/sheets/[0-9a-f]{64}$`));
  const { name: plan } = await server.api.createPlan(
    project,
    "Agent DOM Plan Fixture",
    [{ id: crypto.randomUUID(), targets: [database], sheet }]
  );
  expect(plan).toMatch(new RegExp(`^${project}/plans/[0-9]+$`));
  const protocol: { method: string; params: unknown }[] = [];
  page.on("websocket", (socket) => {
    if (!new URL(socket.url()).pathname.endsWith("/lsp")) return;
    socket.on("framesent", ({ payload }) => {
      try {
        const message = JSON.parse(payload.toString());
        if (
          /^textDocument\/did(?:Open|Change|Close)$/.test(message.method) ||
          (message.method === "workspace/executeCommand" &&
            message.params?.command === "setMetadata")
        )
          protocol.push({ method: message.method, params: message.params });
      } catch {
        // Non-JSON heartbeat frames do not describe editor activity.
      }
    });
  });
  const agent = await open(page, `${server.baseURL}/${plan}`);
  const section = page.locator("#plan-phase-changes");
  await expect(section.locator(".monaco-editor")).toHaveCount(1);
  await agent.minimize();
  await section
    .getByRole("button", { name: "Edit", exact: true })
    .last()
    .click();
  await expect
    .poll(
      () =>
        protocol.filter(({ method }) => method === "textDocument/didOpen")
          .length
    )
    .toBeGreaterThan(0);
  await expect
    .poll(
      () =>
        protocol.filter(({ method }) => method === "workspace/executeCommand")
          .length
    )
    .toBeGreaterThan(0);
  const nativeEditor = section.locator(".monaco-editor textarea");
  await nativeEditor.press("Control+End");
  await nativeEditor.press("Enter");
  await nativeEditor.pressSequentially("-- native warmup");
  await expect
    .poll(
      () =>
        protocol.filter(({ method }) => method === "textDocument/didChange")
          .length
    )
    .toBeGreaterThan(0);
  await section.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(section.locator(".monaco-editor")).toHaveCount(1);
  const nativeProtocolPath = testInfo.outputPath("native-lsp-protocol.json");
  await writeFile(nativeProtocolPath, JSON.stringify(protocol, null, 2));
  await testInfo.attach("native-lsp-protocol.json", {
    path: nativeProtocolPath,
    contentType: "application/json",
  });
  const reader = await section.locator(".monaco-editor").elementHandle();
  if (!reader) throw new Error("Missing warm read-only Monaco");
  const readerText = await reader.textContent();
  const originalPlan = await server.api.getPlan(plan);
  await agent.open();
  const relevant: Request[] = [];
  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    if (
      /\/(?:bytebase\.v1\.(?:SheetService\/(?:CreateSheet|UpdateSheet)|PlanService\/UpdatePlan|ReleaseService\/CheckRelease|SQLService\/(?:Query|AdminExecute)|IssueService\/(?:ApproveIssue|UpdateIssue|CreateIssue))|v1\/.*(?:sheets|plans|:query|:approve|:check))$/.test(
        path
      ) &&
      request.method() !== "GET"
    )
      relevant.push(request);
  });
  const protocolStart = protocol.length;
  await page.clock.install();
  const run = agent.queue(
    [
      button("Edit"),
      {
        name: "sql",
        match: /<textarea.*>SQL<\/textarea> input:local-draft$/,
        type: "input",
        value: "SELECT 42 AS agent_dom_fixture;",
        after: async () => {
          await expect(
            section.getByRole("textbox", { name: "SQL", exact: true })
          ).toHaveValue("SELECT 42 AS agent_dom_fixture;");
          expect(
            await reader.evaluate(
              (element) =>
                element.isConnected && element.closest("[inert]") !== null
            )
          ).toBe(true);
          expect(await reader.isVisible()).toBe(false);
          await expect(section.locator(".monaco-editor")).toHaveCount(1);
          await page.screenshot({
            path: testInfo.outputPath("plan-local-desktop.png"),
            fullPage: true,
          });
        },
      },
      {
        name: "save-blocked",
        match: /<button>Save<\/button> manual-required$/,
        type: "click",
      },
      {
        name: "edit-blocked",
        match: /<button>Edit<\/button> manual-required$/,
        index: -1,
        type: "click",
      },
      button("Cancel"),
    ],
    true
  );
  await agent.send(
    "Edit the local Plan SQL, refuse manual Save and Edit, then discard the draft."
  );
  await run.ready;
  const title = page.locator(`input[data-plan-title-input="${plan}"]`);
  await title.fill("Unsaved focused DOM Plan title");
  const expectFocusUnchanged = await observeFocus(page);
  run.release();
  await run.done();
  await page.clock.fastForward(3_000);
  await expectFocusUnchanged();
  expect(await reader.evaluate((element) => element.isConnected)).toBe(true);
  await expect.poll(() => reader.textContent()).toBe(readerText);
  expect(protocol.slice(protocolStart)).toEqual([]);
  expect(relevant).toHaveLength(0);
  const agentProtocolPath = testInfo.outputPath("agent-local-protocol.json");
  await writeFile(
    agentProtocolPath,
    JSON.stringify(
      {
        protocol: protocol.slice(protocolStart),
        requests: relevant.map((request) => ({
          method: request.method(),
          path: new URL(request.url()).pathname,
        })),
      },
      null,
      2
    )
  );
  await testInfo.attach("agent-local-protocol.json", {
    path: agentProtocolPath,
    contentType: "application/json",
  });
  expect(await server.api.getPlan(plan)).toEqual(originalPlan);
  const unchangedSheet = await api<{ content: string }>(
    "GET",
    `${sheet}?raw=true`
  );
  expect(Buffer.from(unchangedSheet.content, "base64").toString("utf8")).toBe(
    sql
  );
  expect(agent.results.get(run.ids.sql)).toMatchObject({ success: true });
  expect(agent.results.get(run.ids["save-blocked"])).toMatchObject({
    success: false,
    code: "manual-required",
  });
  expect(agent.results.get(run.ids["edit-blocked"])).toMatchObject({
    success: false,
    code: "manual-required",
  });
  await expect(title).toBeFocused();
  await expect(title).toHaveValue("Unsaved focused DOM Plan title");
  await page.screenshot({
    path: testInfo.outputPath("plan-read-desktop.png"),
    fullPage: true,
  });
  await title.fill(originalPlan.title);
  await agent.minimize();
  await section
    .getByRole("button", { name: "Edit", exact: true })
    .last()
    .click();
  const nativeSaveEditor = section.locator(".monaco-editor textarea");
  await nativeSaveEditor.press("Control+a");
  await nativeSaveEditor.pressSequentially("SELECT 43 AS agent_dom_fixture;");
  await section.getByRole("button", { name: "Save", exact: true }).click();
  await expect
    .poll(
      async () =>
        (await server.api.getPlan(plan)).specs?.[0].changeDatabaseConfig?.sheet
    )
    .not.toBe(sheet);
  const savedSheet = (await server.api.getPlan(plan)).specs?.[0]
    .changeDatabaseConfig?.sheet;
  const persistedSheet = await api<{ content: string }>(
    "GET",
    `${savedSheet}?raw=true`
  );
  expect(Buffer.from(persistedSheet.content, "base64").toString("utf8")).toBe(
    "SELECT 43 AS agent_dom_fixture;"
  );
  expect(
    relevant.some((request) => request.url().includes("CreateSheet"))
  ).toBe(true);
  expect(relevant.some((request) => request.url().includes("UpdatePlan"))).toBe(
    true
  );
  await reader.dispose();
});
