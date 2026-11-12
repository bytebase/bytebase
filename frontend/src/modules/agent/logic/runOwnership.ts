export interface AgentRunLease {
  // Await nested work before returning from the owning segment.
  run<T>(action: () => Promise<T>): Promise<T>;
}

const requestedLeases = new Set<string>();
const heldLeases = new Set<string>();
let persistenceDepth = 0;

const executionKey = (scopeKey: string, chatId: string): string =>
  `bb-agent-run:${JSON.stringify([scopeKey, chatId])}`;

export const supportsAgentRunRecovery = (): boolean =>
  typeof globalThis.navigator?.locks?.request === "function";

export const hasAgentRunLease = (scopeKey: string, chatId: string): boolean =>
  heldLeases.has(executionKey(scopeKey, chatId));

export const withAgentRunLease = async <T>(
  scopeKey: string,
  chatId: string,
  action: (lease: AgentRunLease) => Promise<T>
): Promise<{ acquired: boolean; value?: T }> => {
  const key = executionKey(scopeKey, chatId);
  if (
    !supportsAgentRunRecovery() ||
    persistenceDepth > 0 ||
    requestedLeases.has(key)
  ) {
    return { acquired: false };
  }
  requestedLeases.add(key);
  try {
    return await navigator.locks.request(
      key,
      { mode: "exclusive", ifAvailable: true },
      async (lock) => {
        if (!lock) return { acquired: false };
        let active = true;
        const lease: AgentRunLease = {
          async run(nested) {
            if (!active || persistenceDepth > 0) {
              throw new Error("Agent execution lease is not available");
            }
            return nested();
          },
        };
        heldLeases.add(key);
        try {
          return { acquired: true, value: await action(lease) };
        } finally {
          active = false;
          heldLeases.delete(key);
        }
      }
    );
  } finally {
    requestedLeases.delete(key);
  }
};

// These callbacks must stay short and must not acquire an execution lease.
export const withAgentPersistenceLock = async <T>(
  scopeKey: string,
  action: () => T | Promise<T>
): Promise<T> => {
  if (!supportsAgentRunRecovery()) {
    throw new Error("Agent recovery requires Web Locks");
  }
  return navigator.locks.request(
    `bb-agent-persistence:${JSON.stringify(scopeKey)}`,
    { mode: "exclusive" },
    async () => {
      persistenceDepth++;
      try {
        return await action();
      } finally {
        persistenceDepth--;
      }
    }
  );
};
