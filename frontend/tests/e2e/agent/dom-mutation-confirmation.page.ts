import {
  fromBinary,
  fromJsonString,
  toBinary,
  toJson,
} from "@bufbuild/protobuf";
import {
  expect,
  type BrowserContext,
  type Page,
  type Request,
  type Route,
} from "@playwright/test";
import {
  type AIChatRequest,
  AIChatMessageRole,
  AIChatRequestSchema,
  AIChatResponseSchema,
} from "../../../src/types/proto-es/v1/ai_service_pb";
import { UpdateDatabaseRequestSchema } from "../../../src/types/proto-es/v1/database_service_pb";

type ModelResponse = {
  stopReason: "TURN_COMPLETED";
  usage: { totalTokens: number; inputTokens: number; outputTokens: number };
  toolCalls: { id: string; name: string; arguments: string }[];
};

export type DomStep = {
  name: string;
  match: RegExp;
  type: "click" | "input" | "select";
  value?: string;
  index?: number;
  before?: () => Promise<void>;
  after?: () => Promise<void>;
  reuse?: string;
};

const tool = (
  id: string,
  name: string,
  args: Record<string, unknown>
): ModelResponse => ({
  stopReason: "TURN_COMPLETED",
  usage: { totalTokens: 20, inputTokens: 15, outputTokens: 5 },
  toolCalls: [{ id, name, arguments: JSON.stringify(args) }],
});

function decodeChat(request: Request) {
  const body = request.postDataBuffer();
  if (!body) throw new Error("Missing Chat request body");
  return request.headers()["content-type"]?.startsWith("application/proto")
    ? fromBinary(AIChatRequestSchema, body)
    : fromJsonString(AIChatRequestSchema, body.toString("utf8"));
}

async function fulfill(route: Route, response: ModelResponse) {
  const binary = route
    .request()
    .headers()
    ["content-type"]?.startsWith("application/proto");
  await route.fulfill({
    status: 200,
    contentType: binary ? "application/proto" : "application/json",
    body: binary
      ? Buffer.from(
          toBinary(
            AIChatResponseSchema,
            fromJsonString(AIChatResponseSchema, JSON.stringify(response))
          )
        )
      : JSON.stringify(response),
  });
}

let nextSequence = 0;

/** Only Chat is scripted; every resource read, permission check and write is real. */
export class DomMutationAgent {
  readonly window;
  readonly allow;
  readonly deny;
  readonly dismiss;
  readonly results = new Map<string, unknown>();
  readonly snapshots = new Map<string, string>();
  readonly refs = new Map<string, string>();
  readonly errors: string[] = [];
  unexpectedCalls = 0;
  calls = 0;
  private responses: ((request: AIChatRequest) => Promise<ModelResponse>)[] =
    [];

  constructor(readonly page: Page) {
    this.window = page.locator("div[data-agent-window]");
    this.allow = this.window.getByRole("button", {
      name: "Allow once",
      exact: true,
    });
    this.deny = this.window.getByRole("button", { name: "Deny", exact: true });
    this.dismiss = this.window.getByRole("button", {
      name: "Dismiss",
      exact: true,
    });
  }

  async install(context: BrowserContext) {
    await context.route(
      /\/(?:bytebase\.v1\.AIService\/Chat|v1\/ai\/chat)(?:\?|$)/,
      async (route) => {
        this.calls++;
        try {
          const request = decodeChat(route.request());
          for (const message of request.messages) {
            if (
              message.role !== AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_TOOL ||
              !message.toolCallId ||
              !message.content
            )
              continue;
            const result = JSON.parse(message.content) as unknown;
            this.results.set(message.toolCallId, result);
            if (
              result &&
              typeof result === "object" &&
              "domTree" in result &&
              typeof result.domTree === "string"
            )
              this.snapshots.set(message.toolCallId, result.domTree);
          }
          const next = this.responses.shift();
          if (!next) {
            this.unexpectedCalls++;
            throw new Error("Unexpected deterministic Chat request");
          }
          await fulfill(route, await next(request));
        } catch (error) {
          this.errors.push(String(error));
          await route.fulfill({
            status: 400,
            contentType: "application/json",
            body: JSON.stringify({
              code: "failed_precondition",
              message: "DOM fixture script could not match the observed page",
            }),
          });
        }
      }
    );
  }

  async open() {
    if (await this.window.isVisible()) return;
    const launcher = this.page.locator("button[data-agent-window]");
    if (await launcher.isVisible()) await launcher.click();
    else
      await this.page
        .getByRole("button", { name: "Agent", exact: true })
        .click();
    await expect(this.window).toBeVisible();
  }

  async minimize() {
    await this.window
      .getByRole("button", { name: "Minimize", exact: true })
      .click();
    await expect(this.window).toBeHidden();
  }

  async send(prompt: string) {
    await this.window.locator("textarea").last().fill(prompt);
    await this.window
      .getByRole("button", { name: "Send", exact: true })
      .click();
  }

  navigate(path: string) {
    this.responses.push(async () =>
      tool("dom-fixture-navigation", "navigate", { path })
    );
    return this.queue([]);
  }

  queue(steps: DomStep[], paused = false) {
    const sequence = ++nextSequence;
    const text = `DOM fixture ${sequence} completed.`;
    const ids: Record<string, string> = {};
    let release!: () => void;
    let started!: () => void;
    const ready = new Promise<void>((resolve) => {
      started = resolve;
    });
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    for (const [index, step] of steps.entries()) {
      const snapshotId = `dom-${sequence}-${index}-snapshot`;
      const actionId = `dom-${sequence}-${index}-action`;
      ids[step.name] = actionId;
      if (step.reuse) {
        const originalId = ids[step.reuse];
        this.responses.push(async () => {
          await steps[index - 1]?.after?.();
          const ref = this.refs.get(originalId);
          if (!ref)
            throw new Error(`Missing previous actual ref ${step.reuse}`);
          await step.before?.();
          return tool(actionId, "dom_action", { type: step.type, ref });
        });
        continue;
      }
      this.responses.push(async () => {
        if (index > 0) await steps[index - 1].after?.();
        if (index === 0) {
          started();
          if (paused) await gate;
        }
        return tool(snapshotId, "get_page_state", { mode: "dom" });
      });
      this.responses.push(async () => {
        const tree = this.snapshots.get(snapshotId);
        if (tree === undefined)
          throw new Error(`Missing actual DOM snapshot ${snapshotId}`);
        const candidates = tree
          .split("\n")
          .filter((line) => step.match.test(line));
        const line = candidates.at(step.index ?? 0);
        if (!line || (step.index === undefined && candidates.length !== 1))
          throw new Error(
            `${step.name}: expected one ${step.match}, found ${candidates.length}\n${tree}`
          );
        const ref = /^\s*\[(e\d+)\]/.exec(line)?.[1];
        if (!ref) throw new Error(`Missing ref on ${line}`);
        this.refs.set(actionId, ref);
        await step.before?.();
        return tool(actionId, "dom_action", {
          type: step.type,
          ref,
          ...(step.value === undefined ? {} : { value: step.value }),
        });
      });
    }
    this.responses.push(async () => {
      await steps.at(-1)?.after?.();
      return tool(`dom-${sequence}-done`, "done", {
        text,
        success: true,
        actions: [],
      });
    });
    return {
      ids,
      text,
      ready,
      release,
      done: async () => {
        await expect(
          this.window.locator("[data-agent-answer]").last()
        ).toContainText(text);
        expect(
          this.errors,
          JSON.stringify(
            [...this.results].filter(([id]) => id.endsWith("-action")),
            null,
            2
          )
        ).toEqual([]);
        expect(this.unexpectedCalls).toBe(0);
      },
    };
  }
}

export function watchDatabaseWrites(page: Page, name: string) {
  const requests: Request[] = [];
  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    if (
      (request.method() === "PATCH" && path === `/v1/${name}`) ||
      (request.method() === "POST" &&
        path === "/bytebase.v1.DatabaseService/UpdateDatabase")
    )
      requests.push(request);
  });
  return requests;
}

export function databasePatch(request: Request) {
  if (request.method() === "PATCH") {
    return {
      database: request.postDataJSON(),
      paths: (
        new URL(request.url()).searchParams.get("updateMask") ?? ""
      ).split(","),
    };
  }
  const body = request.postDataBuffer();
  if (!body) throw new Error("Missing UpdateDatabase body");
  const decoded = request
    .headers()
    ["content-type"]?.startsWith("application/proto")
    ? fromBinary(UpdateDatabaseRequestSchema, body)
    : fromJsonString(UpdateDatabaseRequestSchema, body.toString("utf8"));
  const json = toJson(UpdateDatabaseRequestSchema, decoded) as {
    database?: Record<string, unknown>;
  };
  return { database: json.database, paths: decoded.updateMask?.paths };
}

export async function observeFocus(page: Page) {
  const active = await page.evaluateHandle(() => document.activeElement);
  await page.evaluate(() => {
    const events: string[] = [];
    Object.assign(window, { domFixtureFocusEvents: events });
    document.addEventListener("focusin", () => events.push("focusin"));
    document.addEventListener("focusout", () => events.push("focusout"));
  });
  return async () => {
    expect(
      await active.evaluate((element) => element === document.activeElement)
    ).toBe(true);
    expect(
      await page.evaluate(
        () =>
          (window as unknown as { domFixtureFocusEvents: string[] })
            .domFixtureFocusEvents
      )
    ).toEqual([]);
    await active.dispose();
  };
}
