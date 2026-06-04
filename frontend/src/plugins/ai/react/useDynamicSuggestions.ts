import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { head, uniq, values } from "lodash-es";
import { useMemo, useSyncExternalStore } from "react";
import { sqlServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import {
  type AICompletionRequest_Message,
  AICompletionRequest_MessageSchema,
  AICompletionRequestSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { storageKeyAiSuggestions } from "@/utils";
import { hashCode } from "@/utils/string";
import * as promptUtils from "../logic/prompt";

export type SuggestionState = "LOADING" | "IDLE" | "ENDED";

export type SuggestionContext = {
  metadata: string;
  key: string;
  suggestions: string[];
  ready: boolean;
  state: SuggestionState;
  current: () => string | undefined;
  consume: () => void;
  fetch: () => Promise<string[]>;
};

const MAX_STORED_SUGGESTIONS = 10;

function saveSuggestionCache(key: string, value: string[]) {
  try {
    localStorage.setItem(storageKeyAiSuggestions(key), JSON.stringify(value));
  } catch {
    // ignore
  }
}

function loadSuggestionCache(key: string): string[] {
  try {
    const raw = localStorage.getItem(storageKeyAiSuggestions(key));
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}

const keyOf = (metadata: string) => String(hashCode(metadata));

// ---------------------------------------------------------------------------
// Framework-agnostic module store (replaces the Vue `reactive` cache). Entries
// are replaced IMMUTABLY on every mutation so React's `useSyncExternalStore`
// snapshot comparison detects the change.
// ---------------------------------------------------------------------------
type Entry = {
  metadata: string;
  key: string;
  suggestions: string[];
  used: Set<string>;
  state: SuggestionState;
};

const entries = new Map<string, Entry>();
const listeners = new Set<() => void>();

function emit() {
  for (const listener of listeners) listener();
}

function ensureEntry(metadata: string): Entry {
  const existing = entries.get(metadata);
  if (existing) return existing;
  const stored = loadSuggestionCache(keyOf(metadata));
  const entry: Entry = {
    metadata,
    key: keyOf(metadata),
    suggestions: stored.length > 0 ? stored : [],
    used: new Set(),
    state: "IDLE",
  };
  entries.set(metadata, entry);
  return entry;
}

function patchEntry(metadata: string, patch: Partial<Entry>) {
  const current = entries.get(metadata);
  if (!current) return;
  entries.set(metadata, { ...current, ...patch });
  emit();
}

async function requestAI(messages: AICompletionRequest_Message[]) {
  await new Promise((resolve) => setTimeout(resolve, 1000));
  try {
    const request = createProto(AICompletionRequestSchema, { messages });
    // Silent mode avoids error notifications for AI completion failures.
    const response = await sqlServiceClientConnect.aICompletion(request, {
      contextValues: createContextValues().set(silentContextKey, true),
    });
    const text =
      head(head(response.candidates)?.content?.parts)?.text?.trim() ?? "";
    const card = JSON.parse(text) as Record<string, string>;
    return values(card ?? {});
  } catch {
    return [];
  }
}

async function fetchSuggestions(metadata: string): Promise<string[]> {
  const entry = entries.get(metadata);
  if (!entry || entry.state === "ENDED") return [];

  const { command, prompt } = promptUtils.dynamicSuggestions(
    metadata,
    new Set([...entry.used.values(), ...entry.suggestions])
  );
  const messages: AICompletionRequest_Message[] = [
    createProto(AICompletionRequest_MessageSchema, {
      role: "system",
      content: command,
    }),
    createProto(AICompletionRequest_MessageSchema, {
      role: "user",
      content: prompt,
    }),
  ];

  patchEntry(metadata, { state: "LOADING" });

  const response = await requestAI(messages);
  const current = entries.get(metadata);
  if (!current) return [];
  const more = uniq(
    response.filter(
      (sug) => !current.used.has(sug) && !current.suggestions.includes(sug)
    )
  );
  const suggestions = [...current.suggestions, ...more];
  patchEntry(metadata, {
    suggestions,
    state: more.length === 0 ? "ENDED" : "IDLE",
  });

  const combined = uniq([...suggestions, ...current.used.values()]).slice(
    0,
    MAX_STORED_SUGGESTIONS
  );
  if (combined.length > 0) {
    saveSuggestionCache(current.key, combined);
  }
  return more;
}

function consumeSuggestion(metadata: string) {
  const entry = entries.get(metadata);
  if (!entry) return;
  const sug = head(entry.suggestions);
  if (!sug) return;
  const suggestions = entry.suggestions.slice(1);
  const used = new Set(entry.used);
  used.add(sug);
  patchEntry(metadata, { suggestions, used });
  if (suggestions.length === 0) {
    void fetchSuggestions(metadata);
  }
}

const subscribe = (onChange: () => void) => {
  listeners.add(onChange);
  return () => {
    listeners.delete(onChange);
  };
};

/**
 * React port of the former Vue `useDynamicSuggestions` composable.
 *
 * Per-(database, engine, schema) cache of LLM-suggested prompt chips. The
 * suggestion state lives in a module store updated immutably; this hook
 * subscribes via `useSyncExternalStore` so the consumer re-renders when the
 * current metadata's suggestions/state change. Returns `undefined` until a
 * database metadata + engine are available.
 */
export const useDynamicSuggestions = (params: {
  databaseMetadata: DatabaseMetadata | undefined;
  engine: Engine | undefined;
  schema: string | undefined;
}): SuggestionContext | undefined => {
  const { databaseMetadata, engine, schema } = params;
  const metadata = useMemo(() => {
    if (databaseMetadata && engine) {
      return promptUtils.databaseMetadataToText(
        databaseMetadata,
        engine,
        schema
      );
    }
    return "";
  }, [databaseMetadata, engine, schema]);

  // Lazily create the entry (idempotent, keyed by metadata) before reading the
  // snapshot, so `getSnapshot` stays pure and returns a stable reference.
  if (metadata) ensureEntry(metadata);
  const snapshot = useSyncExternalStore(subscribe, () =>
    metadata ? entries.get(metadata) : undefined
  );

  return useMemo<SuggestionContext | undefined>(() => {
    if (!metadata || !snapshot) return undefined;
    return {
      metadata: snapshot.metadata,
      key: snapshot.key,
      suggestions: snapshot.suggestions,
      ready: true,
      state: snapshot.state,
      current: () => head(snapshot.suggestions),
      consume: () => consumeSuggestion(metadata),
      fetch: () => fetchSuggestions(metadata),
    };
  }, [metadata, snapshot]);
};
