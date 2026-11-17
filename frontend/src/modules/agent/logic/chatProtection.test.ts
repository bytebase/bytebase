import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  createAgentChatProtection,
  getProtectedAgentChatIds,
  supportsAgentChatProtection,
  withAgentChatEvictionLease,
} from "./chatProtection";

const deferred = () => {
  let resolve!: () => void;
  const promise = new Promise<void>((done) => {
    resolve = done;
  });
  return { promise, resolve };
};

function fakeLocks() {
  interface Request {
    name: string;
    mode: LockMode;
    start: (lock: Lock | null) => void;
    abort: () => void;
  }
  const held = new Set<Request>();
  const pending: Request[] = [];
  const drain = (name: string) => {
    for (;;) {
      const next = pending.find((request) => request.name === name);
      if (!next) return;
      const owners = [...held].filter((request) => request.name === name);
      if (
        owners.some((request) => request.mode === "exclusive") ||
        (next.mode === "exclusive" && owners.length > 0)
      ) {
        return;
      }
      pending.splice(pending.indexOf(next), 1);
      held.add(next);
      next.start({ name, mode: next.mode });
    }
  };
  const request = vi.fn(
    <T>(
      name: string,
      options: LockOptions,
      action: (lock: Lock | null) => T | PromiseLike<T>
    ): Promise<T> =>
      new Promise<T>((resolve, reject) => {
        if (options.signal?.aborted) {
          reject(new DOMException("Aborted", "AbortError"));
          return;
        }
        const mode = options.mode ?? "exclusive";
        const unavailable =
          pending.some((request) => request.name === name) ||
          [...held].some(
            (request) =>
              request.name === name &&
              (mode === "exclusive" || request.mode === "exclusive")
          );
        const entry: Request = {
          name,
          mode,
          start(lock) {
            options.signal?.removeEventListener("abort", entry.abort);
            Promise.resolve()
              .then(() => action(lock))
              .then(
                (value) => {
                  held.delete(entry);
                  drain(name);
                  resolve(value);
                },
                (error: unknown) => {
                  held.delete(entry);
                  drain(name);
                  reject(error);
                }
              );
          },
          abort() {
            const index = pending.indexOf(entry);
            if (index < 0) return;
            pending.splice(index, 1);
            options.signal?.removeEventListener("abort", entry.abort);
            reject(new DOMException("Aborted", "AbortError"));
            drain(name);
          },
        };
        if (options.ifAvailable && unavailable) {
          entry.start(null);
          return;
        }
        pending.push(entry);
        options.signal?.addEventListener("abort", entry.abort);
        queueMicrotask(() => drain(name));
      })
  );
  const query = vi.fn(async () => ({
    held: [...held].map(({ name, mode }) => ({ name, mode, clientId: "tab" })),
    pending: pending.map(({ name, mode }) => ({ name, mode, clientId: "tab" })),
  }));
  return { request, query };
}

const controllers: ReturnType<typeof createAgentChatProtection>[] = [];
const protection = (onChange: () => void = () => undefined) => {
  const controller = createAgentChatProtection(onChange);
  controllers.push(controller);
  return controller;
};

beforeEach(() => vi.stubGlobal("navigator", { locks: fakeLocks() }));
afterEach(() => {
  for (const controller of controllers.splice(0)) controller.dispose();
  vi.unstubAllGlobals();
});

describe("Agent chat protection", () => {
  test.each([
    undefined,
    {},
    { locks: {} },
    { locks: { request: true, query: () => undefined } },
    { locks: { request: (): undefined => undefined, query: true } },
    { locks: { request: (): undefined => undefined } },
  ])("disables eviction without both callable lock APIs: %j", async (value) => {
    vi.stubGlobal("navigator", value);
    const controller = protection();
    controller.update("scope", ["chat"]);
    expect(supportsAgentChatProtection()).toBe(false);
    expect(controller.isReady("scope", "chat")).toBe(true);
    let ran = false;
    expect(
      await withAgentChatEvictionLease("scope", "chat", async () => {
        ran = true;
      })
    ).toEqual({ acquired: false });
    expect(ran).toBe(false);
    await expect(getProtectedAgentChatIds("scope")).rejects.toThrow();
  });

  test("protects the same chat in distinct tabs until every shared owner releases", async () => {
    const first = protection();
    const second = protection();
    first.update("scope", ["chat"]);
    second.update("scope", ["chat"]);
    expect(first.isReady("scope", "chat")).toBe(false);
    expect(second.isReady("scope", "chat")).toBe(false);
    await vi.waitFor(() => {
      expect(first.isReady("scope", "chat")).toBe(true);
      expect(second.isReady("scope", "chat")).toBe(true);
    });
    expect(await getProtectedAgentChatIds("scope")).toEqual(new Set(["chat"]));
    expect(
      await withAgentChatEvictionLease("scope", "chat", async () => "deleted")
    ).toEqual({ acquired: false });

    first.dispose();
    expect(first.isReady("scope", "chat")).toBe(false);
    expect(
      await withAgentChatEvictionLease("scope", "chat", async () => "deleted")
    ).toEqual({ acquired: false });
    second.dispose();
    await vi.waitFor(async () => {
      expect(await getProtectedAgentChatIds("scope")).toEqual(new Set());
    });
    expect(
      await withAgentChatEvictionLease("scope", "chat", async () => "deleted")
    ).toEqual({ acquired: true, value: "deleted" });
  });

  test("does not reacquire or notify recursively for unchanged requested chats", async () => {
    let controller!: ReturnType<typeof protection>;
    const onChange = vi.fn(() => controller.update("scope", ["keep", "drop"]));
    controller = protection(onChange);
    controller.update("scope", ["keep", "drop", "keep"]);
    await vi.waitFor(() => expect(onChange).toHaveBeenCalledTimes(2));
    controller.update("scope", ["drop", "keep"]);
    expect(onChange).toHaveBeenCalledTimes(2);
    expect((await navigator.locks.query()).held).toHaveLength(2);

    onChange.mockImplementation(() => undefined);
    controller.update("scope", ["keep"]);
    expect(controller.isReady("scope", "drop")).toBe(false);
    expect(onChange).toHaveBeenCalledTimes(3);
    await vi.waitFor(async () => {
      expect(await getProtectedAgentChatIds("scope")).toEqual(
        new Set(["keep"])
      );
    });
    expect(controller.isReady("scope", "keep")).toBe(true);
  });

  test("uses collision-free scope and chat keys", async () => {
    const controller = protection();
    controller.update("a:b", ["c"]);
    await vi.waitFor(() => expect(controller.isReady("a:b", "c")).toBe(true));
    expect(controller.isReady("a", "b:c")).toBe(false);
    expect(await getProtectedAgentChatIds("a")).toEqual(new Set());
    expect(await withAgentChatEvictionLease("a", "b:c", async () => 1)).toEqual(
      { acquired: true, value: 1 }
    );
    expect(await getProtectedAgentChatIds("a:b")).toEqual(new Set(["c"]));
  });

  test("lists queued claims and cancels them promptly when identity changes", async () => {
    const release = deferred();
    const blocker = navigator.locks.request(
      'bb-agent-chat-protection:"old":"chat"',
      { mode: "exclusive" },
      () => release.promise
    );
    const controller = protection();
    controller.update("old", ["chat"]);
    expect(controller.isReady("old", "chat")).toBe(false);
    expect(await getProtectedAgentChatIds("old")).toEqual(new Set(["chat"]));
    expect((await navigator.locks.query()).pending).toHaveLength(1);

    controller.update("new", ["chat"]);
    expect(controller.isReady("old", "chat")).toBe(false);
    await vi.waitFor(() =>
      expect(controller.isReady("new", "chat")).toBe(true)
    );
    expect((await navigator.locks.query()).pending).toEqual([]);
    release.resolve();
    await blocker;
    expect(await getProtectedAgentChatIds("old")).toEqual(new Set());
    controller.update(null, ["chat"]);
    expect(controller.isReady("new", "chat")).toBe(false);
    await vi.waitFor(async () => {
      expect(await getProtectedAgentChatIds("new")).toEqual(new Set());
    });
  });

  test("cancels a queued claim on disposal without later announcing readiness", async () => {
    const release = deferred();
    const blocker = navigator.locks.request(
      'bb-agent-chat-protection:"scope":"chat"',
      { mode: "exclusive" },
      () => release.promise
    );
    const onChange = vi.fn();
    const controller = protection(onChange);
    controller.update("scope", ["chat"]);
    await navigator.locks.query();
    controller.dispose();
    expect((await navigator.locks.query()).pending).toEqual([]);
    const notifications = onChange.mock.calls.length;
    release.resolve();
    await blocker;
    expect(controller.isReady("scope", "chat")).toBe(false);
    expect(onChange).toHaveBeenCalledTimes(notifications);
    controller.update("scope", ["ignored"]);
    expect(await getProtectedAgentChatIds("scope")).toEqual(new Set());
  });

  test("releases a granted claim even when disposed before its callback runs", async () => {
    const controller = protection();
    controller.update("scope", ["chat"]);
    await navigator.locks.query();
    controller.dispose();
    await vi.waitFor(async () => {
      expect(await getProtectedAgentChatIds("scope")).toEqual(new Set());
    });
    expect(controller.isReady("scope", "chat")).toBe(false);
  });

  test.each([false, true])(
    "fails closed after a browser acquisition error (sync=%s)",
    async (sync) => {
      const locks = fakeLocks();
      locks.request.mockImplementation(() => {
        if (sync) throw new Error("locks unavailable");
        return Promise.reject(new Error("locks unavailable"));
      });
      vi.stubGlobal("navigator", { locks });
      const onChange = vi.fn();
      const controller = protection(onChange);
      controller.update("scope", ["chat"]);
      await vi.waitFor(() => expect(onChange).toHaveBeenCalledTimes(1));
      expect(controller.isReady("scope", "chat")).toBe(false);
      controller.update("scope", ["chat"]);
      expect(onChange).toHaveBeenCalledTimes(1);
      let ran = false;
      expect(
        await withAgentChatEvictionLease("scope", "chat", async () => {
          ran = true;
        })
      ).toEqual({ acquired: false });
      expect(ran).toBe(false);
    }
  );

  test("propagates query failure rather than treating protected chats as absent", async () => {
    const locks = fakeLocks();
    locks.query.mockRejectedValue(new Error("query unavailable"));
    vi.stubGlobal("navigator", { locks });
    await expect(getProtectedAgentChatIds("scope")).rejects.toThrow(
      "query unavailable"
    );
  });

  test("filters exact scope prefixes before decoding chat IDs", async () => {
    const locks = fakeLocks();
    locks.query.mockResolvedValue({
      held: [
        {
          name: 'bb-agent-chat-protection:"scope":"chat"',
          mode: "shared",
          clientId: "tab",
        },
        {
          name: 'bb-agent-chat-protection:"scope:other":invalid-json',
          mode: "shared",
          clientId: "tab",
        },
        {
          name: 'bb-agent-chat-protection:"other":invalid-json',
          mode: "shared",
          clientId: "tab",
        },
        {
          name: 'bb-agent-run:["scope","not-protection"]',
          mode: "exclusive",
          clientId: "tab",
        },
      ],
      pending: [
        {
          name: 'bb-agent-chat-protection:"scope":"pending"',
          mode: "shared",
          clientId: "tab",
        },
      ],
    });
    vi.stubGlobal("navigator", { locks });
    expect(await getProtectedAgentChatIds("scope")).toEqual(
      new Set(["chat", "pending"])
    );
  });

  test("serializes eviction and releases its exclusive claim after action errors", async () => {
    const release = deferred();
    const first = withAgentChatEvictionLease("scope", "chat", async () => {
      await release.promise;
      throw new Error("delete failed");
    });
    await vi.waitFor(async () => {
      expect(await getProtectedAgentChatIds("scope")).toEqual(
        new Set(["chat"])
      );
    });
    let ran = false;
    expect(
      await withAgentChatEvictionLease("scope", "chat", async () => {
        ran = true;
      })
    ).toEqual({ acquired: false });
    expect(ran).toBe(false);
    const failed = expect(first).rejects.toThrow("delete failed");
    release.resolve();
    await failed;
    expect(
      await withAgentChatEvictionLease("scope", "chat", async () => 2)
    ).toEqual({ acquired: true, value: 2 });
  });
});
