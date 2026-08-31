import { v4 as uuidv4 } from "uuid";

const STORAGE_PREFIX = "bb.plan.initial-sql.";
const STORAGE_TTL_MS = 24 * 60 * 60 * 1000;
const MAX_STORED_VALUES = 20;

type StoredInitialSQL = {
  createdAt: number;
  value: unknown;
};

// React StrictMode replays the initial fetch in development. Keep consumed
// values in memory so the replay sees the same SQL after durable storage has
// already been cleared.
const consumedValues = new Map<string, StoredInitialSQL>();

const parseStoredValue = (raw: string): StoredInitialSQL | undefined => {
  try {
    const parsed = JSON.parse(raw) as Partial<StoredInitialSQL>;
    if (
      typeof parsed !== "object" ||
      parsed === null ||
      typeof parsed.createdAt !== "number" ||
      !("value" in parsed)
    ) {
      return undefined;
    }
    return parsed as StoredInitialSQL;
  } catch {
    return undefined;
  }
};

const pruneStoredValues = (maxStoredValues = MAX_STORED_VALUES) => {
  const now = Date.now();
  for (const [key, stored] of consumedValues) {
    if (now - stored.createdAt > STORAGE_TTL_MS) {
      consumedValues.delete(key);
    }
  }
  const storedValues: Array<{ createdAt: number; key: string }> = [];
  for (let i = localStorage.length - 1; i >= 0; i--) {
    const key = localStorage.key(i);
    if (!key?.startsWith(STORAGE_PREFIX)) continue;
    const stored = parseStoredValue(localStorage.getItem(key) ?? "");
    if (!stored || now - stored.createdAt > STORAGE_TTL_MS) {
      localStorage.removeItem(key);
    } else {
      storedValues.push({ createdAt: stored.createdAt, key });
    }
  }
  storedValues
    .sort((a, b) => a.createdAt - b.createdAt)
    .slice(0, Math.max(0, storedValues.length - maxStoredValues))
    .forEach(({ key }) => localStorage.removeItem(key));
};

export const saveInitialSQL = (value: unknown): string => {
  pruneStoredValues(MAX_STORED_VALUES - 1);
  const key = `${STORAGE_PREFIX}${uuidv4()}`;
  localStorage.setItem(key, JSON.stringify({ createdAt: Date.now(), value }));
  return key;
};

export const consumeInitialSQL = <T>(key: string): T | undefined => {
  if (!key.startsWith(STORAGE_PREFIX)) return undefined;
  pruneStoredValues();
  const consumed = consumedValues.get(key);
  if (consumed) {
    return consumed.value as T;
  }
  try {
    const stored = parseStoredValue(localStorage.getItem(key) ?? "");
    if (!stored) {
      return undefined;
    }
    consumedValues.set(key, stored);
    return stored.value as T;
  } finally {
    localStorage.removeItem(key);
  }
};
