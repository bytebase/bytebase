import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { withAgentPersistenceLock } from "../logic/runOwnership";
import { createAgentRunPersistence } from "./runPersistence";

const deferred = () => {
  let resolve!: () => void;
  const promise = new Promise<void>((done) => {
    resolve = done;
  });
  return { promise, resolve };
};

beforeEach(() => {
  const tails = new Map<string, Promise<unknown>>();
  vi.stubGlobal("navigator", {
    locks: {
      request: <T>(
        name: string,
        _options: LockOptions,
        action: (lock: Lock) => T | Promise<T>
      ) => {
        const prior = tails.get(name) ?? Promise.resolve();
        const result = prior
          .catch(() => undefined)
          .then(() => action({ name, mode: "exclusive" }));
        tails.set(name, result);
        return result;
      },
    },
  });
});
afterEach(() => vi.unstubAllGlobals());

test("coalesces queued mutations and reads current state only after acquiring the scope lock", async () => {
  const release = deferred();
  let current = 0;
  const writes: number[] = [];
  const blocker = withAgentPersistenceLock("scope", () => release.promise);
  const writer = createAgentRunPersistence("scope", () => {
    writes.push(current);
  });
  writer.schedule();
  current = 1;
  writer.schedule();
  current = 2;
  expect(writes).toEqual([]);
  release.resolve();
  await blocker;
  expect(await writer.flush()).toBe(true);
  expect(writes).toEqual([2]);
});

test("preserves mutations scheduled while a write is in flight", async () => {
  const started = deferred();
  const release = deferred();
  let current = 1;
  const writes: number[] = [];
  const writer = createAgentRunPersistence("scope", async () => {
    writes.push(current);
    if (writes.length === 1) {
      started.resolve();
      await release.promise;
    }
  });
  writer.schedule();
  await started.promise;
  current = 2;
  writer.schedule();
  const flushed = writer.flush();
  release.resolve();
  expect(await flushed).toBe(true);
  expect(writes).toEqual([1, 2]);
});

test("serializes different local writers so fresh rebases preserve both chat ledgers", async () => {
  let stored: string[] = [];
  const persist = async (attemptId: string) => {
    const latest = [...stored, attemptId];
    await Promise.resolve();
    stored = latest;
  };
  const first = createAgentRunPersistence("scope", () =>
    persist("chat-1-attempt")
  );
  const second = createAgentRunPersistence("scope", () =>
    persist("chat-2-attempt")
  );
  first.schedule();
  second.schedule();
  expect(await Promise.all([first.flush(), second.flush()])).toEqual([
    true,
    true,
  ]);
  expect(stored).toEqual(["chat-1-attempt", "chat-2-attempt"]);
});

test("makes rejected writes observable without retrying or dropping dirty state", async () => {
  const failure = new Error("storage unavailable");
  let attempts = 0;
  let available = false;
  const writer = createAgentRunPersistence("scope", () => {
    attempts++;
    if (!available) throw failure;
  });
  writer.schedule();
  expect(await writer.flush()).toBe(false);
  expect(writer.getError()).toBe(failure);
  await Promise.resolve();
  expect(attempts).toBe(1);
  available = true;
  expect(await writer.flush()).toBe(true);
  expect(attempts).toBe(2);
  expect(writer.getError()).toBeUndefined();
});

test("does not silently write without a supported lock API", async () => {
  vi.stubGlobal("navigator", {});
  let written = false;
  const writer = createAgentRunPersistence("scope", () => {
    written = true;
  });
  writer.schedule();
  expect(await writer.flush()).toBe(false);
  expect(written).toBe(false);
  expect(writer.getError()).toBeInstanceOf(Error);
});

test("a clean flush performs no write", async () => {
  let writes = 0;
  const writer = createAgentRunPersistence("scope", () => {
    writes++;
  });
  expect(await writer.flush()).toBe(true);
  expect(writes).toBe(0);
});
