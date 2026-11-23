import { fromBinary, fromJsonString } from "@bufbuild/protobuf";
import { expect, type BrowserContext, type Page, test } from "@playwright/test";
import {
  AccessGrantSchema,
  AccessGrant_Status,
  SearchMyAccessGrantsRequestSchema,
} from "../../../src/types/proto-es/v1/access_grant_service_pb";
import {
  type AIChatRequest,
  AIChatMessageRole,
  AIChatRequestSchema,
} from "../../../src/types/proto-es/v1/ai_service_pb";
import { GetSettingRequestSchema } from "../../../src/types/proto-es/v1/setting_service_pb";
import { DeterministicProvider } from "../agent-token-budget/token-budget.fixture";
import { loadTestEnv } from "../framework/env";
import { loginBrowserContext } from "../framework/sign-in";

test.setTimeout(90_000);

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
          apiKey?: string;
          model?: string;
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
    aiMask
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
    aiMask
  );
});

const windowOf = (page: Page) => page.locator("[data-agent-window]:not(button)");
const button = (page: Page, name: string) =>
  windowOf(page).getByRole("button", { name, exact: true });
const tool = (id: string, name: string, args: object) => ({
  stopReason: "TURN_COMPLETED",
  toolCalls: [{ id, name, arguments: JSON.stringify(args) }],
});
const done = (text: string, success = false) =>
  tool("finish", "done", { text, success, actions: [] });

function observeChat(context: BrowserContext) {
  const requests: AIChatRequest[] = [];
  context.on("request", (request) => {
    if (
      !/\/bytebase\.v1\.AIService\/Chat$/.test(new URL(request.url()).pathname)
    )
      return;
    const body = request.postDataBuffer();
    if (!body) throw new Error("Missing Chat request body");
    requests.push(
      request.headers()["content-type"]?.startsWith("application/proto")
        ? fromBinary(AIChatRequestSchema, body)
        : fromJsonString(AIChatRequestSchema, body.toString())
    );
  });
  return requests;
}

function afterTool(requests: AIChatRequest[], id: string): AIChatRequest {
  const request = requests.find((request) =>
    request.messages.some((message) => message.toolCallId === id)
  );
  if (!request) throw new Error(`Missing provider-bound result ${id}`);
  return request;
}

function resultOf(
  requests: AIChatRequest[],
  id: string
): Record<string, unknown> {
  const content = afterTool(requests, id).messages.find(
    (message) => message.toolCallId === id
  )?.content;
  if (!content) throw new Error(`Missing tool content ${id}`);
  return JSON.parse(content);
}

const guidanceOf = (request: AIChatRequest) =>
  request.messages
    .filter(
      (message) =>
        message.role === AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_SYSTEM
    )
    .map((message) => message.content ?? "")
    .join("\n");

async function open(page: Page, baseURL: string) {
  await page.goto(`${baseURL}/landing`);
  await page.getByRole("button", { name: "Agent", exact: true }).click();
  await expect(windowOf(page)).toBeVisible();
}

async function send(page: Page, prompt: string) {
  await windowOf(page).locator("textarea").last().fill(prompt);
  await button(page, "Send").click();
}

async function sendReferencedQuery(
  page: Page,
  env: ReturnType<typeof loadTestEnv>,
  statement: string
) {
  await windowOf(page).locator("textarea").last().fill(`Run ${statement} on @`);
  const picker = page.locator("[data-agent-resource-picker]");
  await picker.getByRole("button", { name: "Databases", exact: true }).click();
  await picker
    .getByLabel("Search databases", { exact: true })
    .fill(env.databaseId);
  await picker
    .locator(`[data-agent-resource-database="${env.database}"]`)
    .click();
  await button(page, "Send").click();
}

async function queryRecoveryJourney(
  page: Page,
  context: BrowserContext,
  confirmed: boolean,
  restricted = false,
  submitRequest = false
) {
  const env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const previous = await env.api.getProject(env.project);
  const { workspace } = await env.api.getActuatorInfo();
  const memberEmail = restricted
    ? `agent-query-viewer-${Date.now()}@example.com`
    : undefined;
  const originalWorkspacePolicy = restricted
    ? await env.api.getWorkspaceIamPolicy(workspace)
    : undefined;
  const originalProjectPolicy = restricted
    ? await env.api.getProjectIamPolicy(env.project)
    : undefined;
  const originalApproval = submitRequest
    ? (await env.api.getSetting("WORKSPACE_APPROVAL"))?.value
    : undefined;
  let submittedGrant: string | undefined;
  let submittedIssue: string | undefined;
  const provider = new DeterministicProvider();
  await provider.install(context);
  const requests = observeChat(context);
  const statement = "SELECT 1 AS value";
  const body = { name: env.database, statement, limit: 1 };
  const completed = confirmed
    ? "The access request is ready for review, not submitted."
    : "The denied query stopped without requesting access.";
  let queries = 0;
  const writes: string[] = [];
  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    if (
      request.method() === "POST" &&
      path === "/bytebase.v1.SQLService/Query"
    ) {
      ++queries;
      expect(request.postDataJSON()).toEqual(body);
    }
    if (
      request.method() === "POST" &&
      /\/(?:Create|Update|Delete|Approve|BatchRun)[A-Z]/.test(path)
    )
      writes.push(path);
  });
  if (!restricted)
    await context.route("**/bytebase.v1.SQLService/Query", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          results: [
            {
              error: "Query access denied",
              permissionDenied: {
                requiredPermissions: ["bb.sql.select"],
                resources: [env.database],
              },
            },
          ],
        }),
      });
    });
  provider.respond(tool("query-skill", "get_skill", { name: "query" }));
  provider.respond(
    tool("query-api", "search_api", { operationId: "SQLService/Query" })
  );
  provider.respond(
    tool("query-call", "call_api", { operationId: "SQLService/Query", body })
  );
  provider.respond(
    tool("eligibility", "get_just_in_time_overview", {
      project: env.project,
      database: env.database,
    })
  );
  provider.respond(
    tool("access-consent", "ask_user", {
      kind: "confirm",
      prompt:
        "The query was denied because bb.sql.select is missing. Request temporary access for this exact query?",
      confirmLabel: "Request temporary access",
      cancelLabel: "Not now",
    })
  );
  if (confirmed) {
    provider.respond(tool("jit-skill", "get_skill", { name: "just-in-time" }));
    provider.respond(
      tool("access-draft", "prepare_access_grant_draft", {
        project: env.project,
        database: env.database,
        statement,
        unmask: false,
      })
    );
  }
  provider.respond(done(completed, confirmed));
  try {
    if (memberEmail) {
      const password = "e2e-agent-query-viewer-1!";
      await env.api.createUser(memberEmail, password, "Agent Query Viewer");
      await env.api.addWorkspaceRoleMember(
        workspace,
        memberEmail,
        "roles/workspaceMember"
      );
      await env.api.appendProjectBinding(env.project, "roles/projectViewer", [
        `user:${memberEmail}`,
      ]);
      await loginBrowserContext(context, env.baseURL, memberEmail, password);
      const currentUser = await context.request.get(
        `${env.baseURL}/v1/users/me`
      );
      expect(currentUser.ok()).toBe(true);
      expect(await currentUser.json()).toMatchObject({ email: memberEmail });
    }
    await env.api.updateProjectSettings(env.project, {
      allowJustInTimeAccess: true,
    });
    if (submitRequest) {
      await env.api.upsertSetting("WORKSPACE_APPROVAL", {
        workspaceApproval: { rules: [{
          source: "REQUEST_ACCESS",
          condition: {expression: "true"},
          template: {
            flow: {roles: ["roles/workspaceAdmin"]},
            title: "Agent access request review",
          },
        }] },
      }, "value.workspace_approval");
    }
    await open(page, env.baseURL);
    await sendReferencedQuery(page, env, statement);
    await expect(button(page, "Allow once")).toBeEnabled();
    expect(queries).toBe(0);
    const queryResponse = page.waitForResponse(
      (response) =>
        new URL(response.url()).pathname === "/bytebase.v1.SQLService/Query"
    );
    await button(page, "Allow once").click();
    const response = await queryResponse;
    expect(response.status()).toBe(200);
    expect(await response.json()).toMatchObject({
      results: [
        {
          error: expect.any(String),
          permissionDenied: {
            requiredPermissions: ["bb.sql.select"],
            resources: [env.database],
          },
        },
      ],
    });
    await expect(button(page, "Request temporary access")).toBeVisible();
    expect(queries).toBe(1);
    expect(writes).toEqual([]);
    expect(resultOf(requests, "query-call")).toMatchObject({
      status: 200,
      apiRecovery: { kind: "query-access-denied" },
      agentExecution: { reason: "query-result-error", effects: "possible" },
    });
    expect(guidanceOf(afterTool(requests, "query-call"))).toContain(
      "Wait for response.confirmed"
    );
    expect(resultOf(requests, "eligibility")).toMatchObject({
      justInTime: { eligible: true, create: { status: "allowed" } },
    });
    await expect(
      windowOf(page).locator("[data-agent-action-request-access]")
    ).toHaveCount(0);
    await button(
      page,
      confirmed ? "Request temporary access" : "Not now"
    ).click();
    await expect(
      windowOf(page).getByText(completed, { exact: true })
    ).toBeVisible();
    expect(resultOf(requests, "access-consent")).toMatchObject({ confirmed });
    expect(queries).toBe(1);
    expect(writes).toEqual([]);
    const draft = windowOf(page).locator("[data-agent-action-request-access]");
    if (confirmed) {
      expect(resultOf(requests, "access-draft")).toMatchObject({
        status: "unsaved",
        draft: {
          project: env.project,
          sourceDatabase: env.database,
          statement,
          unmask: false,
        },
      });
      await expect(draft).toBeVisible();
      if (restricted) {
        await draft.click();
        const drawer = page.getByRole("dialog", {
          name: "Request access grant",
          exact: true,
        });
        await expect(drawer).toBeVisible();
        await expect(
          drawer.getByRole("code").getByText(statement, { exact: true })
        ).toBeVisible();
        await expect(
          drawer.getByRole("checkbox", { name: "See unmasked sensitive data" })
        ).not.toBeChecked();
        expect(queries).toBe(1);
        expect(writes).toEqual([]);
        if (submitRequest) {
          let statusReads = 0;
          page.on("request", (request) => {
            if (new URL(request.url()).pathname !== "/bytebase.v1.AccessGrantService/SearchMyAccessGrants") return;
            const payload = request.postDataBuffer()!;
            const body = request.headers()["content-type"]?.startsWith("application/proto")
              ? fromBinary(SearchMyAccessGrantsRequestSchema, payload)
              : fromJsonString(SearchMyAccessGrantsRequestSchema, payload.toString());
            if (!body.filter?.includes("/accessGrants/")) return;
            ++statusReads;
            expect(body).toMatchObject({
              parent: env.project,
              filter: expect.stringContaining(submittedGrant ?? "/accessGrants/"),
            });
          });
          const editedStatement = "SELECT 2 AS value";
          await drawer.getByRole("code").click();
          await page.keyboard.press("Control+a");
          await page.keyboard.insertText(editedStatement);
          await expect(drawer.getByRole("code").getByText(editedStatement, {exact: true})).toBeVisible();
          await drawer.getByRole("checkbox", {name: "Export the query result"}).check();
          const createdResponse = page.waitForResponse((response) =>
            new URL(response.url()).pathname === "/bytebase.v1.AccessGrantService/CreateAccessGrant"
          );
          const nativeIssuePage = context.waitForEvent("page");
          await drawer.getByRole("button", {name: "Submit", exact: true}).click();
          const created = await createdResponse;
          expect(created.ok()).toBe(true);
          const payload = await created.body();
          const grant = created.headers()["content-type"]?.startsWith("application/proto")
            ? fromBinary(AccessGrantSchema, payload)
            : fromJsonString(AccessGrantSchema, payload.toString());
          submittedGrant = grant.name;
          submittedIssue = grant.issue;
          expect(grant).toMatchObject({
            creator: `users/${memberEmail}`,
            query: editedStatement,
            targets: [env.database],
            export: true,
            status: AccessGrant_Status.PENDING,
            issue: expect.stringMatching(/\/issues\/\d+$/),
          });
          await (await nativeIssuePage).close();
          await expect(drawer).toBeHidden();
          await page.locator("#app").getByRole("button", {name: "Agent", exact: true}).click();
          const receipt = windowOf(page).locator(`[data-agent-submitted-access-grant="${grant.name}"]`);
          await expect(receipt.getByRole("button", {name: "Check status", exact: true})).toBeVisible();
          await expect(draft).toHaveCount(0);
          expect(writes).toEqual(["/bytebase.v1.AccessGrantService/CreateAccessGrant"]);
          const sourceChatId = await windowOf(page)
            .locator('[data-agent-chat-row]:has(button[aria-current="true"])')
            .getAttribute("data-agent-chat-row");
          expect(sourceChatId).toBeTruthy();
          const restoreSourceChat = async () => {
            if (!(await windowOf(page).isVisible())) {
              await page.locator("#app").getByRole("button", {name: "Agent", exact: true}).click();
            }
            const sourceChat = windowOf(page).locator(`[data-agent-chat-row="${sourceChatId}"]`);
            if (!(await sourceChat.isVisible())) {
              await windowOf(page).getByRole("button", {name: "Chats", exact: true}).click();
            }
            await sourceChat.getByRole("button").first().click();
            await expect(receipt).toBeVisible();
          };

          await page.reload();
          await restoreSourceChat();
          expect(statusReads).toBe(0);
          const expectSmallActions = async () => {
            for (const name of ["Open request", "Check status"]) {
              const action = receipt.getByRole("button", {name, exact: true});
              await expect(action).toBeVisible();
              await expect.poll(() => action.evaluate((element) => {
                const control = getComputedStyle(element);
                const icon = getComputedStyle(element.querySelector("svg")!);
                return {
                  height: element.getBoundingClientRect().height,
                  paddingInline: control.paddingInline,
                  fontSize: control.fontSize,
                  lineHeight: control.lineHeight,
                  gap: control.columnGap,
                  iconWidth: icon.width,
                  iconHeight: icon.height,
                  iconShrink: icon.flexShrink,
                };
              })).toEqual({
                height: 28,
                paddingInline: "8px",
                fontSize: "12px",
                lineHeight: "16px",
                gap: "6px",
                iconWidth: "16px",
                iconHeight: "16px",
                iconShrink: "0",
              });
            }
          };
          for (const viewport of [{width: 1440, height: 1000}, {width: 390, height: 844}]) {
            await page.setViewportSize(viewport);
            await expectSmallActions();
            expect(await receipt.evaluate((element) => element.scrollWidth - element.clientWidth)).toBeLessThanOrEqual(1);
            await page.screenshot({path: test.info().outputPath(`submitted-grant-${viewport.width}.png`)});
            await receipt.screenshot({path: test.info().outputPath(`submitted-grant-actions-${viewport.width}.png`)});
            let release: (() => void) | undefined;
            const held = new Promise<void>((resolve) => { release = resolve; });
            const searchPath = "**/bytebase.v1.AccessGrantService/SearchMyAccessGrants";
            await page.route(searchPath, async (route) => {
              await held;
              await route.continue();
            });
            const check = receipt.getByRole("button", {name: "Check status", exact: true});
            try {
              await check.click();
              await expect(check).toBeDisabled();
              await expect(check).toHaveAttribute("aria-busy", "true");
              await expectSmallActions();
              await receipt.screenshot({path: test.info().outputPath(`submitted-grant-loading-${viewport.width}.png`)});
            } finally {
              release?.();
            }
            await expect(check).toBeEnabled();
            await page.unroute(searchPath);
            await expect(receipt.locator("[data-agent-access-grant-status]")).toHaveText("Pending");
          }
          await page.setViewportSize({width: 1440, height: 1000});
          expect(statusReads).toBe(2);

          await receipt.getByRole("button", {name: "Open request", exact: true}).click();
          await expect(page).toHaveURL(new RegExp(`/${grant.issue}(?:[?#]|$)`));
          await expect(windowOf(page)).toBeHidden();
          // The native issue page refreshes its own grant when approval changes.
          await page.goto(`${env.baseURL}/landing`);
          await restoreSourceChat();
          const readsBeforeCheck = statusReads;
          await env.api.approveIssue(grant.issue);
          await expect.poll(async () => (await env.api.getIssue(grant.issue)).status).toBe("DONE");
          expect(statusReads).toBe(readsBeforeCheck);
          await receipt.getByRole("button", {name: "Check status", exact: true}).click();
          await expect(receipt.locator("[data-agent-access-grant-status]")).toHaveText("Active");
          expect(statusReads).toBe(readsBeforeCheck + 1);
          expect(queries).toBe(1);
          expect(writes).toEqual(["/bytebase.v1.AccessGrantService/CreateAccessGrant"]);
          expect(provider.calls).toBe(8);
          provider.respond(done("Your submitted Access Grant was saved and last checked as Active.", true));
          await send(page, "Was my access request saved?");
          await expect(windowOf(page).getByText("Your submitted Access Grant was saved and last checked as Active.", {exact: true})).toBeVisible();
          const followUp = requests.at(-1)!;
          const outcomes = followUp.messages.map((message) => message.content).join("\n");
          expect(outcomes).toContain("Observed native-save outcomes");
          expect(outcomes).toContain(grant.name);
          expect(outcomes).toContain('"status":"ACTIVE"');
          expect(provider.calls).toBe(9);
          expect(queries).toBe(1);
          expect(writes).toEqual(["/bytebase.v1.AccessGrantService/CreateAccessGrant"]);
        }
      }
    } else {
      await expect(draft).toHaveCount(0);
      expect(
        requests
          .flatMap((request) => request.messages)
          .flatMap((message) => message.toolCalls)
          .some((call) => call.name === "prepare_access_grant_draft")
      ).toBe(false);
    }
    await expect(button(page, "Allow once")).toBeHidden();
    expect(provider.unexpectedCalls).toBe(0);
    expect(provider.calls).toBe(confirmed ? submitRequest ? 9 : 8 : 6);
  } finally {
    try {
      if (submittedGrant && submittedIssue) {
        const issue = await env.api.getIssue(submittedIssue);
        if (issue.status === "DONE") await env.api.revokeAccessGrant(submittedGrant);
        else await env.api.rejectIssue(submittedIssue, "Close the disposable test request");
      }
    } finally {
      if (submitRequest) {
        await env.api.upsertSetting("WORKSPACE_APPROVAL", originalApproval ?? {
          workspaceApproval: {rules: []},
        }, "value.workspace_approval");
      }
      await env.api.updateProjectSettings(env.project, {
        allowJustInTimeAccess: !!previous.allowJustInTimeAccess,
      });
      if (originalWorkspacePolicy) {
        const latest = await env.api.getWorkspaceIamPolicy(workspace);
        await env.api.setWorkspaceIamPolicy(workspace, {
          ...originalWorkspacePolicy,
          etag: latest.etag,
        });
      }
      if (originalProjectPolicy) {
        const latest = await env.api.getProjectIamPolicy(env.project);
        await env.api.setProjectIamPolicy(env.project, {
          ...originalProjectPolicy,
          etag: latest.etag,
        });
      }
      if (memberEmail) await env.api.deleteUser(memberEmail);
    }
  }
}

for (const confirmed of [false, true]) {
  test(`all-denied Query recovery ${confirmed ? "prepares only a member-reviewed draft after consent" : "stops after declined consent without retry or draft"}`, async ({
    page,
    context,
  }) => {
    await queryRecoveryJourney(page, context, confirmed);
  });
}

test.describe("restricted-member query recovery", () => {
  test.use({ storageState: { cookies: [], origins: [] } });
  for (const confirmed of [false, true]) {
    test(`real permission denial ${confirmed ? "opens an unsent Access Grant draft after consent" : "stops after declined consent"}`, async ({
      page,
      context,
    }) => {
      await queryRecoveryJourney(page, context, confirmed, true);
    });
  }
  test("links a submitted grant across reload and checks approval only on member clicks without retrying the query", async ({page, context}) => {
    await queryRecoveryJourney(page, context, true, true, true);
  });
});

test("typed skill errors retain paid-feature guidance without leaking diagnostics or inventing HTTP", async ({
  page,
  context,
}) => {
  const env = loadTestEnv();
  const provider = new DeterministicProvider();
  await provider.install(context);
  const requests = observeChat(context);
  let rejectedReads = 0;
  await context.route(
    "**/bytebase.v1.SettingService/GetSetting",
    async (route) => {
      const body = route.request().postDataBuffer()!;
      const request = route
        .request()
        .headers()
        ["content-type"]?.startsWith("application/proto")
        ? fromBinary(GetSettingRequestSchema, body)
        : fromJsonString(GetSettingRequestSchema, body.toString());
      if (request.name !== "settings/SEMANTIC_TYPES") return route.continue();
      ++rejectedReads;
      await route.fulfill({
        status: 403,
        contentType: "application/json",
        body: JSON.stringify({
          code: "permission_denied",
          message:
            "feature FEATURE_DATA_CLASSIFICATION is a ENTERPRISE feature, please upgrade to access it",
          details: [
            {
              type: "example.PrivateDiagnostic",
              value: Buffer.from("private diagnostic").toString("base64"),
            },
          ],
        }),
      });
    }
  );
  provider.respond(
    tool("masking-skill", "get_skill", {
      name: "data-masking",
      section: "semantic-type",
    })
  );
  provider.respond(tool("semantic-types", "ask_masking_semantic_type", {}));
  provider.respond(
    done(
      "The semantic-type feature requires Enterprise. No draft was prepared."
    )
  );
  await open(page, env.baseURL);
  await send(page, "Which semantic type can I use?");
  await expect(
    windowOf(page).getByText(
      "The semantic-type feature requires Enterprise. No draft was prepared.",
      { exact: true }
    )
  ).toBeVisible();
  const result = resultOf(requests, "semantic-types");
  expect(result).toMatchObject({
    apiError: { source: "connect", code: "permission_denied" },
    apiRecovery: {
      kind: "feature-required",
      requiredPlan: "ENTERPRISE",
      source: "api-client",
    },
  });
  expect(result).not.toHaveProperty("status");
  expect(result).not.toHaveProperty("response");
  expect(JSON.stringify(result)).not.toContain("PrivateDiagnostic");
  expect(JSON.stringify(result)).not.toContain(
    Buffer.from("private diagnostic").toString("base64")
  );
  expect(guidanceOf(afterTool(requests, "semantic-types"))).toContain(
    "The API client reported an explicit paid-feature requirement"
  );
  expect(guidanceOf(afterTool(requests, "semantic-types"))).toContain(
    "does not establish an HTTP response"
  );
  await expect(button(page, "Allow once")).toBeHidden();
  await expect(
    windowOf(page).locator("[data-agent-action-request-access]")
  ).toHaveCount(0);
  expect(rejectedReads).toBeGreaterThan(0);
  expect(provider.calls).toBe(3);
  expect(provider.unexpectedCalls).toBe(0);
});

test("an unclassified API rejection reaches the common fallback and completes without retry", async ({
  page,
  context,
}) => {
  const env = loadTestEnv();
  const provider = new DeterministicProvider();
  await provider.install(context);
  const requests = observeChat(context);
  let calls = 0;
  await context.route(
    "**/bytebase.v1.DatabaseService/GetDatabase",
    async (route) => {
      if (
        !route
          .request()
          .headers()
          ["content-type"]?.startsWith("application/json")
      )
        return route.continue();
      ++calls;
      await route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({
          code: "internal",
          message: "Controlled internal error",
        }),
      });
    }
  );
  provider.respond(
    tool("database-read", "call_api", {
      operationId: "DatabaseService/GetDatabase",
      body: { name: env.database },
    })
  );
  provider.respond(
    done("The resource read failed; its cause is not established.")
  );
  await open(page, env.baseURL);
  await send(page, "Read the selected database state");
  await expect(
    windowOf(page).getByText(
      "The resource read failed; its cause is not established.",
      { exact: true }
    )
  ).toBeVisible();
  const guidance = guidanceOf(afterTool(requests, "database-read"));
  expect(guidance).toContain(
    "Explain the observed failure or uncertainty using available evidence"
  );
  expect(guidance).not.toContain("paid-feature requirement");
  expect(guidance).not.toContain(
    "get_just_in_time_overview for the known database"
  );
  expect(calls).toBe(1);
  await expect(button(page, "Allow once")).toBeHidden();
  expect(provider.calls).toBe(2);
  expect(provider.unexpectedCalls).toBe(0);
});

test("local skill validation stays a clarification rather than an API failure", async ({
  page,
  context,
}) => {
  const env = loadTestEnv();
  const provider = new DeterministicProvider();
  await provider.install(context);
  const requests = observeChat(context);
  provider.respond(
    tool("masking-skill", "get_skill", {
      name: "data-masking",
      section: "semantic-type",
    })
  );
  provider.respond(tool("invalid-draft", "prepare_semantic_type_draft", {}));
  provider.respond(
    done("Please provide the semantic type title and masking algorithm.")
  );
  await open(page, env.baseURL);
  await send(page, "Prepare a semantic type without enough details");
  await expect(
    windowOf(page).getByText(
      "Please provide the semantic type title and masking algorithm.",
      { exact: true }
    )
  ).toBeVisible();
  const result = resultOf(requests, "invalid-draft");
  expect(result.error).toBeTruthy();
  expect(result).not.toHaveProperty("apiError");
  expect(result).not.toHaveProperty("apiRecovery");
  expect(guidanceOf(afterTool(requests, "invalid-draft"))).not.toContain(
    "Explain the observed failure or uncertainty using available evidence"
  );
  await expect(button(page, "Allow once")).toBeHidden();
  expect(provider.calls).toBe(3);
  expect(provider.unexpectedCalls).toBe(0);
});
