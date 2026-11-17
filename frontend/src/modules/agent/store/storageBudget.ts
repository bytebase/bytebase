export const AGENT_STORAGE_ADMISSION_BYTES = 2 * 1024 * 1024;
export const AGENT_STORAGE_LIMIT_BYTES = 3 * 1024 * 1024;
export const AGENT_STORAGE_CLEANUP_TRIGGER_BYTES = 1.75 * 1024 * 1024;
export const AGENT_STORAGE_CLEANUP_TARGET_BYTES = 1.5 * 1024 * 1024;

const AGENT_STORAGE_KEY_ROOTS = [
  "bb-agent-state-v2",
  "bb-agent-state-v3",
  "bb-agent-window",
  "bb-agent-preferences-v1",
  "bb-agent-ui-v1",
];

const isAgentStorageKey = (key: string): boolean =>
  key === "bb-agent-state-v2-migration-owner" ||
  AGENT_STORAGE_KEY_ROOTS.some(
    (root) => key === root || key.startsWith(`${root}.`)
  );

export class AgentStorageFullError extends Error {
  constructor(
    readonly reason: "budget" | "quota",
    readonly bytes: number,
    readonly limitBytes = AGENT_STORAGE_LIMIT_BYTES,
    cause?: unknown
  ) {
    super(
      reason === "budget"
        ? `Agent storage is full. Agent data requires ${bytes} bytes; the limit is ${limitBytes} bytes.`
        : `Agent storage is full. The browser rejected the write for ${bytes} bytes of Agent data.`,
      cause === undefined ? undefined : { cause }
    );
    this.name = "AgentStorageFullError";
  }
}

export function estimateAgentStorageBytes(
  storage: Storage,
  replacement?: { key: string; value: string }
): number {
  let bytes = 0;
  let replacementSeen = false;
  for (let index = 0; index < storage.length; index++) {
    const key = storage.key(index);
    if (key === null || !isAgentStorageKey(key)) continue;
    const replacing = replacement?.key === key;
    const value = replacing ? replacement.value : storage.getItem(key);
    if (replacing) replacementSeen = true;
    if (value !== null) bytes += 2 * (key.length + value.length);
  }
  if (replacement && !replacementSeen && isAgentStorageKey(replacement.key)) {
    bytes += 2 * (replacement.key.length + replacement.value.length);
  }
  return bytes;
}
