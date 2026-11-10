import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  AIChatMessageRole,
  AIChatMessageSchema,
  AIChatToolDefinitionSchema,
} from "@/types/proto-es/v1/ai_service_pb";
import { buildPlanReviewPrompt } from "./planIssueAssistance";
import { buildSystemPrompt } from "./prompt";
import {
  measureAgentRequest,
  serializedAgentRequestBytes,
} from "./requestDiagnostics";
import { getMaskingSkillGuidance } from "./skills";
import { maskingSections } from "./skills/dataMasking";
import { getToolDefinitions } from "./tools";

const initialRequestFixtures = Object.freeze([
  Object.freeze({
    name: "workspace",
    page: Object.freeze({
      path: "/",
      title: "Workspace",
      context: "workspaces/example",
    }),
  }),
  Object.freeze({
    name: "project",
    page: Object.freeze({
      path: "/projects/example",
      title: "Project",
      context: "projects/example",
    }),
  }),
  Object.freeze({
    name: "selected Plan",
    page: Object.freeze({
      path: "/projects/example/plans/1",
      title: "Plan",
      context: "projects/example",
      referencedProject: "projects/example",
      referencedPlan: "projects/example/plans/1",
      runMode: "read-only" as const,
    }),
  }),
]);

const recordBenchmarkMetrics = (metrics: Record<string, unknown>) => {
  if (process.env.AGENT_REQUEST_BENCHMARK === "1")
    process.stdout.write(`${JSON.stringify(metrics)}\n`);
};

describe("measureAgentRequest", () => {
  test("matches the request guard's UTF-8 serialization without exposing content", () => {
    const messages = [
      create(AIChatMessageSchema, {
        role: AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_SYSTEM,
        content: "Safety rules",
      }),
      create(AIChatMessageSchema, {
        role: AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_USER,
        content: "\u79d8\u5bc6",
      }),
    ];
    const toolDefinitions = [
      create(AIChatToolDefinitionSchema, {
        name: "lookup",
        description: "private tool description",
        parametersSchema: JSON.stringify({ type: "object" }),
      }),
    ];
    const bytes = (value: unknown) =>
      new TextEncoder().encode(JSON.stringify(value)).byteLength;

    expect(measureAgentRequest(messages, toolDefinitions)).toEqual({
      serializedBytes: bytes({ messages, toolDefinitions }),
      systemBytes: bytes([messages[0]]),
      conversationBytes: bytes([messages[1]]),
      toolBytes: bytes(toolDefinitions),
      messageCount: 2,
      toolCount: 1,
    });
    expect(serializedAgentRequestBytes(messages, toolDefinitions)).toBe(
      bytes({ messages, toolDefinitions })
    );
  });

  test("measures empty request lists exactly", () => {
    expect(measureAgentRequest([], [])).toEqual({
      serializedBytes: 36,
      systemBytes: 2,
      conversationBytes: 2,
      toolBytes: 2,
      messageCount: 0,
      toolCount: 0,
    });
  });

  test.each(initialRequestFixtures)(
    "keeps the actual $name initial protobuf request below 50,000 bytes",
    ({ name, page }) => {
      const systemPrompt = buildSystemPrompt(page);
      const tools = getToolDefinitions(
        "referencedProject" in page
          ? { referencedProject: page.referencedProject, runMode: page.runMode }
          : {}
      );
      const messages = [
        create(AIChatMessageSchema, {
          role: AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_SYSTEM,
          content: systemPrompt,
        }),
        create(AIChatMessageSchema, {
          role: AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_USER,
          content: buildPlanReviewPrompt(),
        }),
      ];
      const toolDefinitions = tools.map((tool) =>
        create(AIChatToolDefinitionSchema, {
          name: tool.name,
          description: tool.description,
          parametersSchema: JSON.stringify(tool.parametersSchema),
        })
      );
      const measured = measureAgentRequest(messages, toolDefinitions);

      expect(measured.serializedBytes).toBe(
        new TextEncoder().encode(JSON.stringify({ messages, toolDefinitions }))
          .byteLength
      );
      expect(measured.serializedBytes).toBeLessThanOrEqual(50_000);
      expect(measured.messageCount).toBe(2);
      expect(measured.toolCount).toBe(tools.length);
      recordBenchmarkMetrics({
        fixture: name,
        ...measured,
        systemStringUnits: systemPrompt.length,
        systemUtf8Bytes: new TextEncoder().encode(systemPrompt).byteLength,
        rawToolBytes: new TextEncoder().encode(JSON.stringify(tools))
          .byteLength,
      });
    }
  );

  test("measures current masking guide sections", () => {
    recordBenchmarkMetrics({
      fixture: "masking-guides",
      sectionStringUnits: Object.fromEntries(
        Object.entries(maskingSections).map(([section, content]) => [
          section,
          content.length,
        ])
      ),
      resolvedStringUnits: Object.fromEntries(
        (Object.keys(maskingSections) as (keyof typeof maskingSections)[]).map(
          (section) => [section, getMaskingSkillGuidance([section]).length]
        )
      ),
    });
    expect(maskingSections.overview.length).toBeLessThanOrEqual(8_000);
  });
});
