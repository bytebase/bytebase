import type { AgentMaskingPageResource } from "./chatContext";
import { getMaskingResumeArgs } from "./maskingResume";
import {
  type GetSkillArgs,
  type MaskingSkillSection,
  resolveSkill,
} from "./skills";
import type { Message, ToolCall } from "./types";

const TOOL_SECTIONS: Record<string, readonly MaskingSkillSection[]> = {
  get_masking_overview: ["overview"],
  get_masking_rule_draft: ["rule"],
  prepare_masking_rule_draft: ["rule"],
  prepare_masking_rule_drafts: ["rule"],
  ask_masking_rule_scope: ["rule"],
  get_masking_exemption_draft: ["exemption"],
  prepare_masking_exemption_draft: ["exemption"],
  ask_masking_exemption_scope: ["exemption"],
  ask_unmask_method: ["exemption"],
  prepare_semantic_type_draft: ["semantic-type"],
  ask_masking_semantic_type: ["rule", "exemption", "semantic-type"],
};

export function getMaskingSectionsForPage(
  page?: AgentMaskingPageResource
): MaskingSkillSection[] {
  if (!page) return [];
  switch (page.kind) {
    case "global-masking":
      return ["overview", "rule"];
    case "masking-exemption":
      return ["overview", "exemption"];
    case "semantic-types":
      return ["overview", "semantic-type"];
    default:
      return ["overview"];
  }
}

export function isMaskingToolEnabled(
  name: string,
  sections: readonly MaskingSkillSection[]
): boolean {
  const required = TOOL_SECTIONS[name];
  if (!required) return true;
  if (name === "get_masking_overview") return sections.length > 0;
  return required.some((section) => sections.includes(section));
}

const parseObject = (
  content: string | undefined
): Record<string, unknown> | undefined => {
  try {
    const value: unknown = JSON.parse(content ?? "");
    return value && typeof value === "object" && !Array.isArray(value)
      ? (value as Record<string, unknown>)
      : undefined;
  } catch {
    return undefined;
  }
};

export function getMaskingSectionsForTurn(
  messages: Message[]
): MaskingSkillSection[] {
  const userIndex = messages.findLastIndex(
    (message) => message.role === "user"
  );
  if (userIndex < 0) return [];
  const turn = messages.slice(userIndex + 1);
  const calls = new Map<string, ToolCall | null>();
  const sections = new Set<MaskingSkillSection>();
  const add = (section: MaskingSkillSection) => {
    sections.add("overview");
    sections.add(section);
  };
  for (const message of turn) {
    if (message.role === "assistant") {
      for (const call of message.toolCalls ?? []) {
        calls.set(call.id, calls.has(call.id) ? null : call);
      }
    }
  }
  for (const message of turn) {
    if (message.role !== "tool" || !message.toolCallId) continue;
    const call = calls.get(message.toolCallId);
    if (call?.name !== "get_skill") continue;
    const args = parseObject(call.arguments);
    const record = parseObject(message.content);
    if (args?.name !== "data-masking" || record?.kind !== "loaded") continue;
    const load = resolveSkill(args as GetSkillArgs);
    if (
      load.kind === "loaded" &&
      load.section &&
      record.name === load.name &&
      record.section === load.section
    ) {
      add(load.section);
    }
  }

  // Legacy clarification records predate typed skill acknowledgements.
  if (getMaskingResumeArgs(messages)) {
    for (const call of calls.values()) {
      if (!call || call.name === "ask_masking_semantic_type") continue;
      for (const section of TOOL_SECTIONS[call.name] ?? []) add(section);
    }
    const answer = parseObject(turn.at(-1)?.content);
    if (
      answer?.value === "action:create-semantic-type" ||
      (!sections.has("rule") && !sections.has("exemption"))
    ) {
      add("semantic-type");
    }
  }
  return ["overview", "rule", "exemption", "semantic-type"].filter(
    (section): section is MaskingSkillSection =>
      sections.has(section as MaskingSkillSection)
  );
}
