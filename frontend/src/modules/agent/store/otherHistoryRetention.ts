import {
  supportsAgentChatProtection,
  withAgentChatEvictionLease,
} from "../logic/chatProtection";
import {
  withAgentPersistenceLock,
  withAgentRunLease,
  withAgentStorageBudgetLock,
} from "../logic/runOwnership";
import type { PersistedAgentState } from "./agent";
import { removeAgentHistoryChats } from "./historyRetention";
import {
  canDiscardStoredAgentHistory,
  canRetireLegacyAgentState,
} from "./legacyStateRetention";
import { encodeAgentStoredState, parseAgentStoredState } from "./stateStorage";
import {
  AGENT_STORAGE_CLEANUP_TARGET_BYTES,
  estimateAgentStorageBytes,
} from "./storageBudget";
import { omitAgentQuerySamples } from "./storageRetention";

interface ReclaimOptions {
  currentKey: string;
  userEmail: string;
  mode: "automatic" | "confirmed";
  isCurrent: () => boolean;
  force?: boolean;
}

interface PeerHistory {
  key: string;
  value: string;
  state: PersistedAgentState;
  legacyKey: string;
  legacyValue: string | null;
  skipped?: boolean;
}

const roots = ["bb-agent-state-v2", "bb-agent-state-v3"];
const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);
const historyRoot = (key: string) =>
  roots.find((root) => key === root || key.startsWith(`${root}.`));
const aliases = (key: string) => {
  const root = historyRoot(key);
  return root
    ? roots
        .map((other) => other + key.slice(root.length))
        .sort((a, b) => a.localeCompare(b))
    : [key];
};
const scopedIdentity = (key: string) => {
  const root = historyRoot(key);
  if (!root || !key.startsWith(`${root}.`)) return undefined;
  const suffix = key.slice(root.length + 1);
  const prefix = /^([1-9]\d*):/.exec(suffix);
  if (!prefix) return undefined;
  const length = Number(prefix[1]);
  if (!Number.isSafeInteger(length)) return undefined;
  const workspace = suffix.slice(prefix[0].length, prefix[0].length + length);
  const separator = prefix[0].length + length;
  if (workspace.length !== length || suffix[separator] !== ".")
    return undefined;
  const userEmail = suffix.slice(separator + 1);
  return userEmail ? { root, workspace, userEmail } : undefined;
};
const entryBytes = (key: string, value: string | null) =>
  value === null ? 0 : 2 * (key.length + value.length);
const decode = (value: string) =>
  parseAgentStoredState(value, { strictCompressionMarkers: true });
const chatIds = (raw: unknown): string[] => {
  if (!isRecord(raw)) return [];
  const chats = Array.isArray(raw.chats) ? raw.chats : raw.threads;
  if (!Array.isArray(chats)) return [];
  return [
    ...new Set(
      chats.flatMap((chat) =>
        isRecord(chat) && typeof chat.id === "string" && chat.id
          ? [chat.id]
          : []
      )
    ),
  ].sort((a, b) => a.localeCompare(b));
};
const hasDrafts = (state: PersistedAgentState) =>
  Object.values(state.draftsByChatId).some((draft) =>
    Object.entries(draft).some(([key, value]) => {
      if (key === "text") return value !== "";
      if (key === "mentions") return !Array.isArray(value) || value.length > 0;
      return true;
    })
  );

const hasLiveLocks = async (
  scopes: readonly string[],
  owned: ReadonlySet<string>,
  unscoped: boolean
) => {
  const snapshot = await navigator.locks.query();
  const matches = (name: string | undefined) => {
    if (!name) return false;
    if (
      scopes.some((scope) =>
        name.startsWith(`bb-agent-chat-protection:${JSON.stringify(scope)}:`)
      )
    )
      return true;
    if (!name.startsWith("bb-agent-run:")) return false;
    if (unscoped) return true;
    try {
      const identity: unknown = JSON.parse(name.slice("bb-agent-run:".length));
      return !Array.isArray(identity) || scopes.includes(identity[0]);
    } catch {
      return true;
    }
  };
  return (
    (snapshot.held ?? []).some(
      (lock) => !owned.has(lock.name ?? "") && matches(lock.name)
    ) || (snapshot.pending ?? []).some((lock) => matches(lock.name))
  );
};

export const withAgentHistoryReclamationLocks = async (
  key: string,
  ids: readonly string[],
  isCurrent: () => boolean,
  action: (lockedChatIds: ReadonlySet<string>, ownsScope: boolean) => number,
  { allowPartial = false }: { allowPartial?: boolean } = {}
): Promise<number> => {
  const scopes = aliases(key);
  const claims = scopes.flatMap((scope) => ids.map((id) => ({ scope, id })));
  const lockedChatIds = new Set(ids);
  const owned = new Set<string>();
  const withPersistence = async (index: number): Promise<number> => {
    if (!isCurrent()) return 0;
    if (index < scopes.length)
      return withAgentPersistenceLock(scopes[index], () =>
        withPersistence(index + 1)
      );
    const liveLocks = await hasLiveLocks(scopes, owned, !scopedIdentity(key));
    // Retiring a legacy scope requires ownership of every chat and no other live claims.
    const ownsScope = lockedChatIds.size === new Set(ids).size && !liveLocks;
    if (!ownsScope && !allowPartial) return 0;
    return withAgentStorageBudgetLock(() =>
      isCurrent() ? action(lockedChatIds, ownsScope) : 0
    );
  };
  const withClaims = async (
    index: number,
    execution: boolean
  ): Promise<number> => {
    if (!isCurrent()) return 0;
    if (index === claims.length)
      return execution ? withPersistence(0) : withClaims(0, true);
    const { scope, id } = claims[index];
    if (!lockedChatIds.has(id)) return withClaims(index + 1, execution);
    const name = execution
      ? `bb-agent-run:${JSON.stringify([scope, id])}`
      : `bb-agent-chat-protection:${JSON.stringify(scope)}:${JSON.stringify(id)}`;
    const nested = async () => {
      owned.add(name);
      try {
        return await withClaims(index + 1, execution);
      } finally {
        owned.delete(name);
      }
    };
    const result = execution
      ? await withAgentRunLease(scope, id, nested)
      : await withAgentChatEvictionLease(scope, id, nested);
    if (result.acquired) return result.value ?? 0;
    if (!allowPartial) return 0;
    lockedChatIds.delete(id);
    return withClaims(index + 1, execution);
  };
  return withClaims(0, false);
};

const readPeer = (storage: Storage, key: string): PeerHistory | undefined => {
  const value = storage.getItem(key);
  if (value === null) return undefined;
  const raw = decode(value);
  if (!canRetireLegacyAgentState(raw, raw)) return undefined;
  const state = raw as PersistedAgentState;
  const complete = {
    ...state,
    currentChatId: state.currentChatId ?? null,
    draftsByChatId: state.draftsByChatId ?? {},
    pendingAskByChatId: state.pendingAskByChatId ?? {},
    queuedUserMessagesByChatId: state.queuedUserMessagesByChatId ?? {},
  };
  if (hasDrafts(complete)) return undefined;
  const legacyKey = aliases(key).find((alias) => alias !== key)!;
  const legacyValue = storage.getItem(legacyKey);
  if (legacyValue !== null) {
    const source = decode(legacyValue);
    const identity = scopedIdentity(key);
    if (
      !canDiscardStoredAgentHistory(source, legacyKey) ||
      !canRetireLegacyAgentState(
        source,
        raw,
        identity ? { scopeKey: key, userEmail: identity.userEmail } : undefined
      )
    )
      return undefined;
  }
  return { key, value, state: complete, legacyKey, legacyValue };
};

const rewritePeer = async (
  storage: Storage,
  peer: PeerHistory,
  options: ReclaimOptions,
  removeChatId?: string
) => {
  let freed = 0;
  try {
    await withAgentHistoryReclamationLocks(
      peer.key,
      chatIds(peer.state),
      options.isCurrent,
      () => {
        if (
          storage.getItem(peer.key) !== peer.value ||
          storage.getItem(peer.legacyKey) !== peer.legacyValue
        ) {
          peer.skipped = true;
          return 0;
        }
        if (peer.legacyValue !== null) {
          storage.removeItem(peer.legacyKey);
          if (storage.getItem(peer.legacyKey) !== null) {
            peer.skipped = true;
            return 0;
          }
          freed += entryBytes(peer.legacyKey, peer.legacyValue);
          peer.legacyValue = null;
        }
        let candidate = removeChatId
          ? removeAgentHistoryChats(
              peer.state,
              new Set([removeChatId]),
              peer.key
            )
          : omitAgentQuerySamples(peer.state);
        if (!removeChatId && !canRetireLegacyAgentState(peer.state, candidate))
          candidate = peer.state;
        const serialized = encodeAgentStoredState(
          JSON.stringify(candidate),
          peer.value,
          true
        );
        if (serialized.length >= peer.value.length) return freed;
        const previous = peer.value;
        storage.setItem(peer.key, serialized);
        if (storage.getItem(peer.key) !== serialized) {
          peer.skipped = true;
          return freed;
        }
        peer.value = serialized;
        peer.state = decode(serialized) as PersistedAgentState;
        freed += 2 * (previous.length - serialized.length);
        return freed;
      }
    );
  } catch {
    if (storage.getItem(peer.key) !== peer.value) peer.skipped = true;
  }
  return freed;
};

// Automatic reclamation requires an inactive, recognized same-account history;
// confirmed reclamation can discard saved history but never live lease holders.
export async function reclaimOtherAgentHistory(
  storage: Storage,
  options: ReclaimOptions
): Promise<number> {
  if (!supportsAgentChatProtection() || !options.isCurrent()) return 0;
  let freed = 0;
  const currentKeys = new Set(aliases(options.currentKey));
  const keys = Array.from({ length: storage.length }, (_, index) =>
    storage.key(index)
  ).filter(
    (key): key is string =>
      key !== null && !currentKeys.has(key) && Boolean(historyRoot(key))
  );
  if (options.mode === "confirmed") {
    for (const key of keys) {
      if (!options.isCurrent()) break;
      try {
        const source = storage.getItem(key);
        if (source === null) continue;
        let raw: unknown;
        try {
          raw = decode(source);
        } catch {
          continue;
        }
        if (!canDiscardStoredAgentHistory(raw, key)) continue;
        freed += await withAgentHistoryReclamationLocks(
          key,
          chatIds(raw),
          options.isCurrent,
          () => {
            if (storage.getItem(key) !== source) return 0;
            storage.removeItem(key);
            return storage.getItem(key) === null ? entryBytes(key, source) : 0;
          }
        );
      } catch {
        // A failed deletion leaves that scope available for explicit recovery later.
      }
    }
    return freed;
  }
  const current = scopedIdentity(options.currentKey);
  if (!current || current.userEmail !== options.userEmail) return 0;
  const peers = keys.flatMap((key) => {
    const identity = scopedIdentity(key);
    if (
      !identity ||
      identity.root !== roots[1] ||
      identity.userEmail !== options.userEmail ||
      identity.workspace === current.workspace
    )
      return [];
    try {
      const peer = readPeer(storage, key);
      return peer ? [peer] : [];
    } catch {
      return [];
    }
  });
  const enough = () =>
    !options.isCurrent() ||
    (estimateAgentStorageBytes(storage) <= AGENT_STORAGE_CLEANUP_TARGET_BYTES &&
      (!options.force || freed > 0));
  try {
    for (const peer of peers) {
      if (enough()) return freed;
      freed += await rewritePeer(storage, peer, options);
    }
    const oldest = peers
      .flatMap((peer) => peer.state.chats.map((chat) => ({ peer, chat })))
      .sort(
        (a, b) =>
          (a.chat.updatedTs ?? 0) - (b.chat.updatedTs ?? 0) ||
          (a.chat.createdTs ?? 0) - (b.chat.createdTs ?? 0) ||
          a.peer.key.localeCompare(b.peer.key) ||
          a.chat.id.localeCompare(b.chat.id)
      );
    for (const { peer, chat } of oldest) {
      if (enough()) break;
      if (!peer.skipped)
        freed += await rewritePeer(storage, peer, options, chat.id);
    }
  } catch {
    return freed;
  }
  return freed;
}
