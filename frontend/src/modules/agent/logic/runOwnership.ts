export interface AgentRunLease {
  // Await nested work before returning from the owning segment.
  run<T>(action: () => Promise<T>): Promise<T>;
}

const requestedLeases = new Set<string>();
const heldLeases = new Set<string>();
const heldChatScopes = new Map<string, string>();
const leaseListeners = new Set<() => void>();
let persistenceDepth = 0;
let storageBudgetDepth = 0;

const executionKey = (scopeKey: string, chatId: string): string =>
  `bb-agent-run:${JSON.stringify([scopeKey, chatId])}`;

export const supportsAgentRunRecovery = (): boolean =>
  typeof globalThis.navigator?.locks?.request === "function";

export const hasAgentRunLease = (scopeKey: string, chatId: string): boolean =>
  heldLeases.has(executionKey(scopeKey, chatId));

export const isAgentRunLeasePending = (
  scopeKey: string,
  chatId: string
): boolean => requestedLeases.has(executionKey(scopeKey, chatId));

export const subscribeAgentRunLeases = (listener: () => void): (() => void) => {
  leaseListeners.add(listener);
  return () => {
    leaseListeners.delete(listener);
  };
};

const notifyLeaseListeners = () => {
  for (const listener of leaseListeners) listener();
};

export const withAgentRunLease = async <T>(
  scopeKey: string,
  chatId: string,
  action: (lease: AgentRunLease) => Promise<T>
): Promise<{ acquired: boolean; value?: T }> => {
  const key = executionKey(scopeKey, chatId);
  if (
    !supportsAgentRunRecovery() ||
    persistenceDepth > 0 ||
    storageBudgetDepth > 0 ||
    requestedLeases.has(key)
  ) {
    return { acquired: false };
  }
  requestedLeases.add(key);
  notifyLeaseListeners();
  try {
    return await navigator.locks.request(
      key,
      { mode: "exclusive", ifAvailable: true },
      async (lock) => {
        if (!lock) return { acquired: false };
        let active = true;
        const lease: AgentRunLease = {
          async run(nested) {
            if (!active || persistenceDepth > 0 || storageBudgetDepth > 0) {
              throw new Error("Agent execution lease is not available");
            }
            return nested();
          },
        };
        heldLeases.add(key);
        heldChatScopes.set(key, scopeKey);
        try {
          return { acquired: true, value: await action(lease) };
        } finally {
          active = false;
          heldLeases.delete(key);
          heldChatScopes.delete(key);
        }
      }
    );
  } finally {
    requestedLeases.delete(key);
    notifyLeaseListeners();
  }
};

const actionKey = (scopeKey: string, actionId: string) =>
  `bb-agent-api-action:${JSON.stringify([scopeKey, actionId])}`;

export async function withAgentApiActionLease<T>(
  scopeKey: string,
  actionId: string,
  action: () => Promise<T>
): Promise<{ acquired: boolean; value?: T }> {
  if (
    !supportsAgentRunRecovery() ||
    persistenceDepth ||
    storageBudgetDepth ||
    ![...heldChatScopes.values()].includes(scopeKey)
  )
    return { acquired: false };
  const key = actionKey(scopeKey, actionId);
  return navigator.locks.request(
    key,
    { mode: "exclusive", ifAvailable: true },
    async (lock) => {
      if (!lock) return { acquired: false };
      return { acquired: true, value: await action() };
    }
  );
}

export async function getAgentApiActionLiveness(
  scopeKey: string,
  actionId: string
): Promise<"live" | "absent" | "unavailable"> {
  if (!supportsAgentRunRecovery() || persistenceDepth || storageBudgetDepth)
    return "unavailable";
  try {
    if (typeof navigator.locks.query === "function") {
      const state = await navigator.locks.query();
      return !state.held
        ? "unavailable"
        : state.held.some((lock) => lock.name === actionKey(scopeKey, actionId))
          ? "live"
          : "absent";
    }
    return await navigator.locks.request(
      actionKey(scopeKey, actionId),
      { mode: "exclusive", ifAvailable: true },
      (lock) => (lock ? ("absent" as const) : ("live" as const))
    );
  } catch {
    return "unavailable";
  }
}

// These callbacks must stay short and must not acquire an execution lease.
export const withAgentPersistenceLock = async <T>(
  scopeKey: string,
  action: () => T | Promise<T>
): Promise<T> => {
  if (!supportsAgentRunRecovery()) {
    throw new Error("Agent recovery requires Web Locks");
  }
  if (storageBudgetDepth > 0) {
    throw new Error("Agent storage accounting cannot acquire a scope lock");
  }
  return navigator.locks.request(
    `bb-agent-persistence:${JSON.stringify(scopeKey)}`,
    { mode: "exclusive" },
    () => {
      persistenceDepth++;
      try {
        return action();
      } finally {
        persistenceDepth--;
      }
    }
  );
};

// Acquire after the scope lock. Accounting and replacement stay synchronous and acquire no locks.
export const withAgentStorageBudgetLock = async <T>(
  action: () => T & (T extends PromiseLike<unknown> ? never : unknown)
): Promise<T> => {
  if (!supportsAgentRunRecovery()) {
    throw new Error("Agent storage accounting requires Web Locks");
  }
  return navigator.locks.request(
    "bb-agent-storage-budget-v1",
    { mode: "exclusive" },
    () => {
      storageBudgetDepth++;
      try {
        return action();
      } finally {
        storageBudgetDepth--;
      }
    }
  );
};
