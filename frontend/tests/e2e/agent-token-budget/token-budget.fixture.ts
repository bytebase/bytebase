import {
  expect,
  test as base,
  type BrowserContext,
  type Page,
  type Route,
} from "@playwright/test";
import {
  create,
  fromBinary,
  fromJsonString,
  toBinary,
} from "@bufbuild/protobuf";
import {
  AIChatRequestSchema,
  AIChatResponseSchema,
} from "../../../src/types/proto-es/v1/ai_service_pb";
import { ActuatorInfoSchema } from "../../../src/types/proto-es/v1/actuator_service_pb";
import {
  AISettingSchema,
  GetSettingRequestSchema,
  SettingSchema,
  SettingValueSchema,
} from "../../../src/types/proto-es/v1/setting_service_pb";
import { loadTestEnv } from "../framework/env";
import { TokenBudgetPage } from "./token-budget.page";

type MockResponse = {
  content?: string;
  toolCalls?: { id: string; name: string; arguments: string }[];
  usage?: {
    totalTokens: number;
    inputTokens?: number;
    outputTokens?: number;
    cachedInputTokens?: number;
  };
  stopReason?: string;
};

const isBinary = (route: Route) =>
  route.request().headers()["content-type"]?.startsWith("application/proto");
const fulfillResponse = (route: Route, response: MockResponse) =>
  route.fulfill({
    status: 200,
    contentType: isBinary(route) ? "application/proto" : "application/json",
    body: isBinary(route)
      ? Buffer.from(
          toBinary(
            AIChatResponseSchema,
            fromJsonString(AIChatResponseSchema, JSON.stringify(response))
          )
        )
      : JSON.stringify(response),
  });

export const answer = (
  totalTokens?: number,
  content = "Deterministic budget fixture completed."
): MockResponse => ({
  toolCalls: [
    {
      id: `budget-qa-done-${totalTokens ?? "unknown"}`,
      name: "done",
      arguments: JSON.stringify({ text: content, success: true, actions: [] }),
    },
  ],
  ...(totalTokens === undefined
    ? {}
    : { usage: { totalTokens, inputTokens: totalTokens, outputTokens: 0 } }),
  stopReason: "TURN_COMPLETED",
});

async function mockAIReadiness(context: BrowserContext) {
  // The disposable harness has no provider configuration. Change only UI
  // readiness on reads; provider traffic is independently trapped below.
  await context.route(
    /\/bytebase\.v1\.(?:ActuatorService\/GetActuatorInfo|SettingService\/GetSetting)(?:\?|$)/,
    async (route) => {
      const actuator = route.request().url().includes("ActuatorService");
      if (!actuator) {
        const request = isBinary(route)
          ? fromBinary(
              GetSettingRequestSchema,
              route.request().postDataBuffer()!
            )
          : route.request().postDataJSON();
        if (request.name !== "settings/AI") return route.continue();
      }
      const response = await route.fetch().catch(() => undefined);
      if (!response) return;
      if (!response.ok()) return route.fulfill({ response });
      if (isBinary(route)) {
        if (actuator) {
          const info = fromBinary(ActuatorInfoSchema, await response.body());
          info.aiEnabled = true;
          return route.fulfill({
            response,
            body: Buffer.from(toBinary(ActuatorInfoSchema, info)),
          });
        }
        const setting = fromBinary(SettingSchema, await response.body());
        setting.value ??= create(SettingValueSchema);
        if (setting.value.value.case === "ai")
          setting.value.value.value.enabled = true;
        else
          setting.value.value = {
            case: "ai",
            value: create(AISettingSchema, { enabled: true }),
          };
        return route.fulfill({
          response,
          body: Buffer.from(toBinary(SettingSchema, setting)),
        });
      }
      const value = await response.json();
      if (actuator) value.aiEnabled = true;
      else {
        value.value ??= {};
        value.value.ai ??= {};
        value.value.ai.enabled = true;
      }
      return route.fulfill({ response, body: JSON.stringify(value) });
    }
  );
}

export const readPage = (totalTokens: number, count = 1): MockResponse => ({
  toolCalls: Array.from({ length: count }, (_, index) => ({
    id: `budget-qa-read-${totalTokens}-${index}`,
    name: "get_page_state",
    arguments: JSON.stringify({ mode: "semantic" }),
  })),
  usage: { totalTokens, inputTokens: totalTokens - 100, outputTokens: 100 },
  stopReason: "TURN_COMPLETED",
});

export class DeterministicProvider {
  calls = 0;
  unexpectedCalls = 0;
  toolResults = 0;
  readToolResultIds: string[] = [];
  ownerPage?: Page;
  phase = "running";
  private steps: ((route: Route) => Promise<void>)[] = [];

  async install(context: BrowserContext) {
    // Every Chat request is intercepted, including unexpected calls, so a test
    // cannot fall through to the configured provider or spend real credits.
    await context.route(
      /\/(?:bytebase\.v1\.AIService\/Chat|v1\/ai\/chat)(?:\?|$)/,
      async (route) => {
        this.calls++;
        this.ownerPage = route.request().frame().page();
        const request = isBinary(route)
          ? fromBinary(AIChatRequestSchema, route.request().postDataBuffer()!)
          : route.request().postDataJSON();
        this.toolResults =
          request.messages?.filter(
            (message: { role: string | number }) =>
              message.role === "AI_CHAT_MESSAGE_ROLE_TOOL" || message.role === 4
          ).length ?? 0;
        this.readToolResultIds =
          request.messages
            ?.filter((message: { toolCallId?: string }) =>
              message.toolCallId?.startsWith("budget-qa-read-")
            )
            .map((message: { toolCallId: string }) => message.toolCallId) ?? [];
        const step = this.steps.shift();
        if (step) await step(route);
        else {
          this.unexpectedCalls++;
          await route.fulfill({
            status: 400,
            contentType: "application/json",
            body: JSON.stringify({
              code: "failed_precondition",
              message: "Unexpected deterministic fixture request",
            }),
          });
        }
      }
    );
  }

  respond(response: MockResponse) {
    this.steps.push((route) => fulfillResponse(route, response));
  }

  hold(response: MockResponse) {
    let release!: () => void;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    this.steps.push(async (route) => {
      await gate;
      await fulfillResponse(route, response).catch(() => {});
    });
    return release;
  }

  fail() {
    this.steps.push((route) =>
      route.fulfill({
        status: 400,
        contentType: "application/json",
        body: JSON.stringify({
          code: "invalid_argument",
          message: "Deterministic provider failure",
        }),
      })
    );
  }
}

type Fixtures = { budget: TokenBudgetPage; provider: DeterministicProvider };

export const test = base.extend<Fixtures>({
  provider: async ({ context }, use, testInfo) => {
    const provider = new DeterministicProvider();
    const diagnostics = {
      consoleWarnings: 0,
      consoleErrors: 0,
      pageErrors: 0,
      errorClasses: [] as string[],
      securityErrors: [] as {
        message: string;
        stack: string;
        owner: boolean;
        phase: string;
      }[],
      consoleErrorClasses: {} as Record<string, number>,
    };
    const seen = new WeakSet<Page>();
    const observe = (page: Page) => {
      if (seen.has(page)) return;
      seen.add(page);
      page.on("console", (message) => {
        if (message.type() === "warning") diagnostics.consoleWarnings++;
        if (message.type() === "error") {
          diagnostics.consoleErrors++;
          const text = message.text();
          const category = /websocket/i.test(text)
            ? "dev-websocket"
            : /cors|cross.origin|access.control.allow.origin/i.test(text)
              ? "origin-policy"
              : /Failed to load resource/i.test(text)
                ? `failed-resource-${text.match(/\b[45]\d\d\b/)?.[0] ?? "network"}`
                : /aria|accessib|react|component/i.test(text)
                  ? "react-accessibility"
                  : "other";
          diagnostics.consoleErrorClasses[category] =
            (diagnostics.consoleErrorClasses[category] ?? 0) + 1;
        }
      });
      page.on("pageerror", (error) => {
        diagnostics.pageErrors++;
        const classification = /abort/i.test(error.message)
          ? "abort-related"
          : /lock/i.test(error.message)
            ? "lock-related"
            : /fetch/i.test(error.message)
              ? "fetch-related"
              : error.name;
        diagnostics.errorClasses.push(classification);
        if (error.name === "SecurityError")
          diagnostics.securityErrors.push({
            message: error.message.replace(/https?:\/\/\S+/g, "[url]"),
            stack: (error.stack ?? "")
              .replace(/https?:\/\/\S+/g, "[url]")
              .split("\n")
              .slice(0, 5)
              .join("\n"),
            owner: page === provider.ownerPage,
            phase: provider.phase,
          });
      });
    };
    context.pages().forEach(observe);
    context.on("page", observe);
    await provider.install(context);
    if (
      !process.env.BYTEBASE_TOKEN_BUDGET_BASE_URL ||
      process.env.BYTEBASE_TOKEN_BUDGET_MOCK_READINESS === "1"
    )
      await mockAIReadiness(context);
    await use(provider);
    await Promise.all(context.pages().map((page) => page.close()));
    await context.unrouteAll({ behavior: "ignoreErrors" });
    await testInfo.attach("browser-diagnostics", {
      body: JSON.stringify({
        ...diagnostics,
        chatCalls: provider.calls,
        unexpectedChatCalls: provider.unexpectedCalls,
      }),
      contentType: "application/json",
    });
    expect(provider.unexpectedCalls, "No unplanned Chat requests").toBe(0);
    expect(diagnostics.pageErrors, "No uncaught browser errors").toBe(0);
  },
  budget: async ({ page, context, provider: _provider }, use, testInfo) => {
    const existingURL = process.env.BYTEBASE_TOKEN_BUDGET_BASE_URL;
    const env = existingURL ? undefined : loadTestEnv();
    const baseURL = existingURL ?? env!.baseURL;
    if (existingURL) {
      const email = process.env.BYTEBASE_E2E_EMAIL;
      const password = process.env.BYTEBASE_E2E_PASSWORD;
      if (!email || !password)
        throw new Error(
          "Existing-server mode requires runtime authentication environment variables"
        );
      const response = await context.request.post(`${baseURL}/v1/auth/login`, {
        data: { email, password, web: true },
      });
      expect(response.status(), "Existing-server fixture login succeeds").toBe(
        200
      );
    }
    await context.addInitScript(() => {
      if (location.origin === "null") return;
      localStorage.setItem(
        "bb.release",
        JSON.stringify({
          ignoreRemindModalTillNextRelease: true,
          nextCheckTs: Date.now() + 86_400_000,
        })
      );
    });
    const budget = new TokenBudgetPage(page, baseURL, testInfo);
    await budget.open();
    await budget.setDefaultLimit("1");
    await use(budget);
  },
});

export { expect };
