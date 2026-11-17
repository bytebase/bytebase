import { compressToUTF16, decompressFromUTF16 } from "lz-string";

const CONTENT_PLACEHOLDER = "[bb-agent-compressed-content-v1]";

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

function* storedMessages(state: unknown): Generator<Record<string, unknown>> {
  if (!isRecord(state) || !isRecord(state.messagesByChatId)) return;
  for (const messages of Object.values(state.messagesByChatId)) {
    if (!Array.isArray(messages)) continue;
    for (const message of messages) {
      if (isRecord(message)) yield message;
    }
  }
}

export function parseAgentStoredState(
  serialized: string,
  {
    strictCompressionMarkers = false,
  }: { strictCompressionMarkers?: boolean } = {}
): unknown {
  const state: unknown = JSON.parse(serialized);
  for (const message of storedMessages(state)) {
    const metadata = message.metadata;
    if (!isRecord(metadata) || !("agentCompressedContent" in metadata))
      continue;
    const marker = metadata.agentCompressedContent;
    if (
      strictCompressionMarkers &&
      (message.content !== CONTENT_PLACEHOLDER ||
        !isRecord(marker) ||
        Object.keys(marker).some(
          (key) => !["version", "value", "metadataAbsent"].includes(key)
        ))
    ) {
      throw new Error("Invalid compressed agent message content.");
    }
    if (message.content === CONTENT_PLACEHOLDER) {
      if (
        !isRecord(marker) ||
        marker.version !== 1 ||
        typeof marker.value !== "string" ||
        !marker.value ||
        (marker.metadataAbsent !== undefined && marker.metadataAbsent !== true)
      ) {
        throw new Error("Invalid compressed agent message content.");
      }
      const content = decompressFromUTF16(marker.value);
      if (typeof content !== "string" || !content) {
        throw new Error("Invalid compressed agent message content.");
      }
      message.content = content;
    }
    delete metadata.agentCompressedContent;
    if (
      isRecord(marker) &&
      marker.metadataAbsent === true &&
      Object.keys(metadata).length === 0
    ) {
      delete message.metadata;
    }
  }
  return state;
}

export function isCompressedAgentState(serialized: string | null): boolean {
  if (!serialized) return false;
  try {
    for (const message of storedMessages(JSON.parse(serialized))) {
      if (
        isRecord(message.metadata) &&
        "agentCompressedContent" in message.metadata
      ) {
        return true;
      }
    }
  } catch {
    return false;
  }
  return false;
}

function compressAgentState(
  json: string,
  previous: string | null = null
): string {
  const previousContents = new Map<string, string>();
  if (previous) {
    for (const message of storedMessages(JSON.parse(previous))) {
      const marker = isRecord(message.metadata)
        ? message.metadata.agentCompressedContent
        : undefined;
      if (
        message.content !== CONTENT_PLACEHOLDER ||
        !isRecord(marker) ||
        marker.version !== 1 ||
        typeof marker.value !== "string"
      ) {
        continue;
      }
      const content = decompressFromUTF16(marker.value);
      if (typeof content === "string" && content) {
        previousContents.set(content, marker.value);
      }
    }
  }
  const state: unknown = JSON.parse(json);
  let changed = false;
  for (const message of storedMessages(state)) {
    const metadata = message.metadata;
    if (
      typeof message.content !== "string" ||
      (metadata !== undefined && !isRecord(metadata)) ||
      metadata?.pending === true ||
      (metadata && "agentCompressedContent" in metadata)
    ) {
      continue;
    }
    // Older clients preserve unknown message metadata while rewriting the JSON envelope.
    const compressed = {
      ...message,
      content: CONTENT_PLACEHOLDER,
      metadata: {
        ...metadata,
        agentCompressedContent: {
          version: 1,
          value:
            previousContents.get(message.content) ??
            compressToUTF16(message.content),
          ...(metadata === undefined ? { metadataAbsent: true } : {}),
        },
      },
    };
    if (JSON.stringify(compressed).length < JSON.stringify(message).length) {
      message.content = compressed.content;
      message.metadata = compressed.metadata;
      changed = true;
    }
  }
  return changed ? JSON.stringify(state) : json;
}

export function encodeAgentStoredState(
  json: string,
  previous: string | null = null,
  compress = false
): string {
  return compress || isCompressedAgentState(previous)
    ? compressAgentState(json, previous)
    : json;
}

export const isAgentStorageQuotaError = (error: unknown): boolean =>
  typeof error === "object" &&
  error !== null &&
  "name" in error &&
  (error.name === "QuotaExceededError" ||
    error.name === "NS_ERROR_DOM_QUOTA_REACHED");

export function writeAgentStoredState(
  storage: Storage,
  key: string,
  json: string,
  previous: string | null
): string {
  if (isCompressedAgentState(previous)) {
    const compressed = encodeAgentStoredState(json, previous);
    if (compressed !== previous) {
      storage.setItem(key, compressed);
    }
    return compressed;
  }
  try {
    if (json !== previous) {
      storage.setItem(key, json);
    }
    return json;
  } catch (error) {
    if (!isAgentStorageQuotaError(error)) {
      throw error;
    }
    const compressed = encodeAgentStoredState(json, null, true);
    if (compressed.length >= json.length) {
      throw error;
    }
    storage.setItem(key, compressed);
    return compressed;
  }
}
