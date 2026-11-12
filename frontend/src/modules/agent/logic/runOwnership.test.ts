import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  hasAgentRunLease,
  supportsAgentRunRecovery,
  withAgentPersistenceLock,
  withAgentRunLease,
} from "./runOwnership";

const deferred = () => {
  let resolve!: () => void;
  const promise = new Promise<void>((done) => {
    resolve = done;
  });
  return { promise, resolve };
};

function fakeLocks() {
  const held = new Set<string>();
  const queues = new Map<string, (() => void)[]>();
  const request = vi.fn(
    async <T>(
      name: string,
      options: LockOptions,
      action: (lock: Lock | null) => Promise<T>
    ) => {
      if (held.has(name)) {
        if (options.ifAvailable) return action(null);
        await new Promise<void>((resolve) => {
          const queue = queues.get(name) ?? [];
          queue.push(resolve);
          queues.set(name, queue);
        });
      }
      held.add(name);
      try {
        return await action({ name, mode: "exclusive" });
      } finally {
        const next = queues.get(name)?.shift();
        if (next) next();
        else held.delete(name);
      }
    }
  );
  return { request };
}

beforeEach(() => vi.stubGlobal("navigator", { locks: fakeLocks() }));
afterEach(() => vi.unstubAllGlobals());

describe("Agent execution leases", () => {
  test.each([undefined, {}, { locks: {} }, { locks: { request: true } }])(
    "fails closed without a callable lock API: %j",
    async (navigator) => {
      vi.stubGlobal("navigator", navigator);
      let ran = false;
      expect(supportsAgentRunRecovery()).toBe(false);
      expect(
        await withAgentRunLease("scope", "chat", async () => {
          ran = true;
        })
      ).toEqual({ acquired: false });
      expect(ran).toBe(false);
      await expect(
        withAgentPersistenceLock("scope", () => undefined)
      ).rejects.toThrow();
    }
  );

  test("refuses two simultaneous same-tab resumes instead of sharing execution", async () => {
    const release = deferred();
    let calls = 0;
    const first = withAgentRunLease("scope", "chat", async () => {
      calls++;
      expect(hasAgentRunLease("scope", "chat")).toBe(true);
      await release.promise;
      return "first";
    });
    expect(
      await withAgentRunLease("scope", "chat", async () => {
        calls++;
      })
    ).toEqual({ acquired: false });
    expect(calls).toBe(1);
    release.resolve();
    expect(await first).toEqual({ acquired: true, value: "first" });
    expect(hasAgentRunLease("scope", "chat")).toBe(false);
  });

  test("refuses a cross-tab lease immediately without queueing", async () => {
    vi.stubGlobal("navigator", {
      locks: {
        request: async (
          _name: string,
          options: LockOptions,
          action: (lock: null) => unknown
        ) => {
          expect(options).toEqual({ mode: "exclusive", ifAvailable: true });
          return action(null);
        },
      },
    });
    let ran = false;
    expect(
      await withAgentRunLease("scope", "chat", async () => {
        ran = true;
      })
    ).toEqual({ acquired: false });
    expect(ran).toBe(false);
  });

  test("allows only one simultaneous resume across isolated tab owners", async () => {
    const release = deferred();
    let segments = 0;
    const first = withAgentRunLease("scope", "chat", async () => {
      segments++;
      await release.promise;
    });
    vi.resetModules();
    const follower = await import("./runOwnership");
    expect(follower.hasAgentRunLease("scope", "chat")).toBe(false);
    expect(
      await follower.withAgentRunLease("scope", "chat", async () => {
        segments++;
      })
    ).toEqual({ acquired: false });
    expect(segments).toBe(1);
    release.resolve();
    await first;
    expect(
      await follower.withAgentRunLease("scope", "chat", async () => 2)
    ).toEqual({ acquired: true, value: 2 });
  });

  test("reserves local acquisition before the browser invokes its callback", async () => {
    const acquired = deferred();
    const locks = fakeLocks();
    vi.stubGlobal("navigator", {
      locks: {
        request: async <T>(
          name: string,
          options: LockOptions,
          action: (lock: Lock | null) => Promise<T>
        ) => {
          await acquired.promise;
          return locks.request(name, options, action);
        },
      },
    });
    const first = withAgentRunLease("scope", "chat", async () => 1);
    expect(hasAgentRunLease("scope", "chat")).toBe(false);
    expect(await withAgentRunLease("scope", "chat", async () => 2)).toEqual({
      acquired: false,
    });
    acquired.resolve();
    expect(await first).toEqual({ acquired: true, value: 1 });
  });

  test("reuses only an explicit active owner handle without a nested lock", async () => {
    const locks = fakeLocks();
    vi.stubGlobal("navigator", { locks });
    let reuse!: () => Promise<string>;
    const result = await withAgentRunLease("scope", "chat", async (lease) => {
      reuse = () => lease.run(async () => "nested");
      return reuse();
    });
    expect(result).toEqual({ acquired: true, value: "nested" });
    expect(locks.request).toHaveBeenCalledTimes(1);
    await expect(reuse()).rejects.toThrow();
  });

  test("releases after errors so a later explicit invocation can acquire", async () => {
    await expect(
      withAgentRunLease("scope", "chat", async () => {
        throw new Error("stopped");
      })
    ).rejects.toThrow("stopped");
    expect(hasAgentRunLease("scope", "chat")).toBe(false);
    expect(await withAgentRunLease("scope", "chat", async () => 7)).toEqual({
      acquired: true,
      value: 7,
    });
  });

  test("cleans up a failed browser acquisition without running its action", async () => {
    vi.stubGlobal("navigator", {
      locks: {
        request: async () => {
          throw new Error("lock unavailable");
        },
      },
    });
    let ran = false;
    await expect(
      withAgentRunLease("scope", "chat", async () => {
        ran = true;
      })
    ).rejects.toThrow("lock unavailable");
    expect(ran).toBe(false);
    expect(hasAgentRunLease("scope", "chat")).toBe(false);
    vi.stubGlobal("navigator", { locks: fakeLocks() });
    expect(await withAgentRunLease("scope", "chat", async () => 7)).toEqual({
      acquired: true,
      value: 7,
    });
  });

  test("uses collision-free scope and chat keys", async () => {
    const release = deferred();
    const first = withAgentRunLease("a:b", "c", async () => release.promise);
    expect(await withAgentRunLease("a", "b:c", async () => 2)).toEqual({
      acquired: true,
      value: 2,
    });
    expect(await withAgentRunLease("a:b", "d", async () => 3)).toEqual({
      acquired: true,
      value: 3,
    });
    release.resolve();
    await first;
  });

  test("allows execution then persistence but refuses acquiring execution from persistence", async () => {
    expect(
      await withAgentRunLease("scope", "chat", async () =>
        withAgentPersistenceLock("scope", () => 4)
      )
    ).toEqual({ acquired: true, value: 4 });
    await withAgentPersistenceLock("scope", async () => {
      expect(await withAgentRunLease("scope", "chat", async () => 5)).toEqual({
        acquired: false,
      });
    });
  });

  test("releases persistence ordering protection after a failed write", async () => {
    await expect(
      withAgentPersistenceLock("scope", () => {
        throw new Error("storage unavailable");
      })
    ).rejects.toThrow("storage unavailable");
    expect(await withAgentRunLease("scope", "chat", async () => 1)).toEqual({
      acquired: true,
      value: 1,
    });
  });

  test("queues short persistence locks while different scopes remain independent", async () => {
    const release = deferred();
    const events: string[] = [];
    const first = withAgentPersistenceLock("scope", async () => {
      events.push("first");
      await release.promise;
    });
    const second = withAgentPersistenceLock("scope", () => {
      events.push("second");
    });
    await withAgentPersistenceLock("other", () => {
      events.push("other");
    });
    expect(events).toEqual(["first", "other"]);
    release.resolve();
    await Promise.all([first, second]);
    expect(events).toEqual(["first", "other", "second"]);
  });
});
