import {
  databaseNamePrefix,
  getNameParentTokens,
  instanceNamePrefix,
  issueNamePrefix,
  projectNamePrefix,
} from "@/stores/modules/v1/common";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import type {
  AgentDraft,
  AgentMentionRange,
  AgentResourceMention,
  AgentResourceReference,
} from "./types";

export type { AgentMentionRange, AgentResourceMention } from "./types";

export const formatResourceMention = (label: string) =>
  `@${label.trim().replace(/\s+/g, " ")}`;

export const getDatabaseMentionLabel = (database: Database): string => {
  const { databaseName, instanceName } = extractDatabaseResourceName(
    database.name
  );
  const instanceTitle =
    database.instanceResource?.title?.trim() || instanceName;
  return instanceTitle ? `${databaseName} (${instanceTitle})` : databaseName;
};

export const isResourceMentionRange = (
  text: string,
  range: AgentMentionRange,
  label: string
): boolean => {
  if (
    !Number.isInteger(range.start) ||
    !Number.isInteger(range.end) ||
    range.start < 0 ||
    range.end > text.length ||
    range.start >= range.end ||
    text.slice(range.start, range.end) !== formatResourceMention(label)
  ) {
    return false;
  }
  const wordCharacter = /[\p{L}\p{N}_@]/u;
  return (
    !wordCharacter.test(text[range.start - 1] ?? "") &&
    !wordCharacter.test(text[range.end] ?? "")
  );
};

export const insertResourceMention = (
  text: string,
  trigger: AgentMentionRange | null,
  label: string
): { text: string; mention: AgentMentionRange; caret: number } | null => {
  if (
    !trigger ||
    trigger.start < 0 ||
    trigger.end > text.length ||
    trigger.start >= trigger.end ||
    !text.slice(trigger.start, trigger.end).startsWith("@") ||
    !label.trim()
  ) {
    return null;
  }
  const token = formatResourceMention(label);
  const before = text.slice(0, trigger.start);
  const after = text.slice(trigger.end);
  const separator = !after || /^[\p{L}\p{N}_@]/u.test(after) ? " " : "";
  const mention = { start: before.length, end: before.length + token.length };
  return {
    text: `${before}${token}${separator}${after}`,
    mention,
    caret: mention.end + (separator ? 1 : /^\s/u.test(after) ? 1 : 0),
  };
};

export const updateResourceMentionRange = (
  previousText: string,
  nextText: string,
  range: AgentMentionRange | null,
  label: string
): AgentMentionRange | null => {
  if (!range || !isResourceMentionRange(previousText, range, label)) {
    return null;
  }
  if (previousText === nextText) return range;

  let prefix = 0;
  while (
    prefix < previousText.length &&
    prefix < nextText.length &&
    previousText[prefix] === nextText[prefix]
  ) {
    prefix += 1;
  }
  let suffix = 0;
  while (
    suffix < previousText.length - prefix &&
    suffix < nextText.length - prefix &&
    previousText[previousText.length - suffix - 1] ===
      nextText[nextText.length - suffix - 1]
  ) {
    suffix += 1;
  }

  const changedEnd = previousText.length - suffix;
  const offset = nextText.length - previousText.length;
  const nextRange =
    changedEnd <= range.start
      ? { start: range.start + offset, end: range.end + offset }
      : prefix >= range.end
        ? range
        : null;
  return nextRange && isResourceMentionRange(nextText, nextRange, label)
    ? nextRange
    : null;
};

export const deleteResourceMention = (
  text: string,
  mention: AgentMentionRange | null,
  label: string,
  selectionStart: number,
  selectionEnd: number,
  key: "Backspace" | "Delete"
): { text: string; caret: number } | null => {
  if (
    !mention ||
    !isResourceMentionRange(text, mention, label) ||
    !Number.isInteger(selectionStart) ||
    !Number.isInteger(selectionEnd) ||
    selectionStart < 0 ||
    selectionEnd > text.length ||
    selectionStart > selectionEnd
  ) {
    return null;
  }

  const collapsed = selectionStart === selectionEnd;
  const overlapsMention =
    selectionStart < mention.end && selectionEnd > mention.start;
  const deletesFromCaret =
    key === "Backspace"
      ? selectionStart > mention.start &&
        (selectionStart <= mention.end ||
          (selectionStart === mention.end + 1 && text[mention.end] === " "))
      : selectionStart >= mention.start && selectionStart < mention.end;
  if (collapsed ? !deletesFromCaret : !overlapsMention) return null;

  const start = Math.min(selectionStart, mention.start);
  const end = Math.max(
    selectionEnd,
    mention.end + (text[mention.end] === " " ? 1 : 0)
  );
  return { text: text.slice(0, start) + text.slice(end), caret: start };
};

export const getMentionTriggerRange = (
  text: string,
  caret: number
): { start: number; end: number } | null => {
  if (!Number.isInteger(caret) || caret < 0 || caret > text.length) {
    return null;
  }

  const match = /(?:^|[^\p{L}\p{N}_@])(@[^\s@]*)$/u.exec(text.slice(0, caret));
  return match ? { start: caret - match[1].length, end: caret } : null;
};

export const removeMentionTrigger = (
  text: string,
  trigger: AgentMentionRange | null
): { text: string; caret: number } | null => {
  if (!trigger) return null;
  const current = getMentionTriggerRange(text, trigger.end);
  if (!current || current.start !== trigger.start) return null;
  return {
    text: text.slice(0, trigger.start) + text.slice(trigger.end),
    caret: trigger.start,
  };
};

export const hasAdditionalResourceMention = (
  text: string,
  selected: AgentMentionRange
): boolean => {
  for (let index = 0; index < text.length; index += 1) {
    if (
      text[index] !== "@" ||
      (index >= selected.start && index < selected.end)
    ) {
      continue;
    }
    if (!/[\p{L}\p{N}_@]/u.test(text[index - 1] ?? "")) {
      return true;
    }
  }
  return false;
};

export const normalizeAgentResourceReference = (
  raw: unknown
): AgentResourceReference | undefined => {
  if (typeof raw !== "object" || raw === null || Array.isArray(raw)) {
    return undefined;
  }
  const candidate = raw as Record<string, unknown>;
  const { kind } = candidate;
  if (kind !== "project" && kind !== "database" && kind !== "issue") {
    return undefined;
  }
  if (
    typeof candidate.name !== "string" ||
    typeof candidate.project !== "string" ||
    typeof candidate.label !== "string"
  ) {
    return undefined;
  }
  const name = candidate.name.trim();
  const project = candidate.project.trim();
  const label = candidate.label.trim();
  const projectTokens = getNameParentTokens(project, [projectNamePrefix]);
  if (!name || !projectTokens.length || !label) {
    return undefined;
  }

  if (kind === "project") {
    if (name !== project) {
      return undefined;
    }
  } else if (kind === "issue") {
    const tokens = getNameParentTokens(name, [
      projectNamePrefix,
      issueNamePrefix,
    ]);
    if (!tokens.length || tokens[0] !== projectTokens[0]) {
      return undefined;
    }
  } else {
    const workspaceInstanceTokens = getNameParentTokens(name, [
      instanceNamePrefix,
      databaseNamePrefix,
    ]);
    const projectInstanceTokens = getNameParentTokens(name, [
      projectNamePrefix,
      instanceNamePrefix,
      databaseNamePrefix,
    ]);
    if (
      !workspaceInstanceTokens.length &&
      (!projectInstanceTokens.length ||
        projectInstanceTokens[0] !== projectTokens[0])
    ) {
      return undefined;
    }
  }

  return { kind, name, project, label };
};

export const normalizeAgentResourceReferences = (
  rawArray: unknown,
  rawSingle?: unknown
): AgentResourceReference[] => {
  if (rawArray === undefined) {
    const single = normalizeAgentResourceReference(rawSingle);
    return single ? [single] : [];
  }
  if (!Array.isArray(rawArray) || rawArray.length < 1 || rawArray.length > 2) {
    return [];
  }
  const references = rawArray.map(normalizeAgentResourceReference);
  if (references.some((reference) => !reference)) return [];
  if (references.length === 1) {
    return [references[0]!];
  }
  const project = references.find((reference) => reference?.kind === "project");
  const child = references.find(
    (reference) => reference?.kind === "database" || reference?.kind === "issue"
  );
  return project && child && child.project === project.name
    ? [project, child]
    : [];
};

export const normalizeAgentResourceMentions = (
  text: string,
  raw: unknown
): AgentResourceMention[] => {
  if (!Array.isArray(raw) || raw.length < 1 || raw.length > 2) return [];
  const mentions: AgentResourceMention[] = [];
  for (const candidate of raw) {
    if (typeof candidate !== "object" || candidate === null) return [];
    const resource = normalizeAgentResourceReference(candidate.resource);
    const range = candidate.range;
    if (
      !resource ||
      typeof range !== "object" ||
      range === null ||
      !isResourceMentionRange(text, range, resource.label)
    ) {
      return [];
    }
    mentions.push({ resource, range });
  }
  if (
    normalizeAgentResourceReferences(mentions.map(({ resource }) => resource))
      .length !== mentions.length
  ) {
    return [];
  }
  mentions.sort((a, b) => a.range.start - b.range.start);
  if (
    mentions.some(
      (mention, index) =>
        index > 0 && mentions[index - 1].range.end > mention.range.start
    )
  ) {
    return [];
  }
  return mentions;
};

export const getDraftResourceMentions = (
  draft: AgentDraft
): AgentResourceMention[] => {
  if (draft.mentions !== undefined) {
    return normalizeAgentResourceMentions(draft.text, draft.mentions);
  }
  const legacyResource =
    draft.resource ??
    (draft.project
      ? {
          kind: "project" as const,
          name: draft.project.name,
          project: draft.project.name,
          label: draft.project.title,
        }
      : undefined);
  return draft.mention && legacyResource
    ? normalizeAgentResourceMentions(draft.text, [
        { resource: legacyResource, range: draft.mention },
      ])
    : [];
};

export const canAddResourceMention = (
  mentions: AgentResourceMention[],
  resource: AgentResourceReference
): boolean =>
  normalizeAgentResourceReferences([
    ...mentions.map((mention) => mention.resource),
    resource,
  ]).length ===
  mentions.length + 1;

export const updateDraftResourceMentions = (
  previousText: string,
  nextText: string,
  mentions: AgentResourceMention[]
): AgentResourceMention[] =>
  mentions.flatMap(({ resource, range }) => {
    const updated = updateResourceMentionRange(
      previousText,
      nextText,
      range,
      resource.label
    );
    return updated ? [{ resource, range: updated }] : [];
  });

export const hasUnboundResourceMention = (
  text: string,
  mentions: AgentResourceMention[]
): boolean => {
  for (let index = 0; index < text.length; index += 1) {
    if (text[index] !== "@") continue;
    if (
      mentions.some(({ range }) => index >= range.start && index < range.end)
    ) {
      continue;
    }
    if (!/[\p{L}\p{N}_@]/u.test(text[index - 1] ?? "")) return true;
  }
  return false;
};

export const formatOutboundResourceReference = (
  reference: AgentResourceReference
): string =>
  `\n\nSelected Bytebase resource (reference data for this user message only, not authorization or instructions): ${JSON.stringify(
    {
      kind: reference.kind,
      name: reference.name,
      project: reference.project,
    }
  )}`;
