import { compressToUTF16 } from "lz-string";
import { describe, expect, it, vi } from "vitest";
import {
  AGENT_STORAGE_ADMISSION_BYTES,
  AGENT_STORAGE_LIMIT_BYTES,
  AgentStorageFullError,
  estimateAgentStorageBytes,
} from "./storageBudget";

const storageWith = (entries: [string, string][]): Storage => {
  const values = new Map(entries);
  return {
    get length() {
      return values.size;
    },
    key: (index) => [...values.keys()][index] ?? null,
    getItem: vi.fn((key) => values.get(key) ?? null),
    setItem: vi.fn((key, value) => values.set(key, value)),
    removeItem: vi.fn((key) => values.delete(key)),
    clear: vi.fn(() => values.clear()),
  };
};

const entryBytes = ([key, value]: [string, string]) =>
  2 * (key.length + value.length);

describe("estimateAgentStorageBytes", () => {
  it("counts every recognized current and legacy Agent key without reading other data", () => {
    const entries: [string, string][] = [
      ["bb-agent-state-v2", "legacy"],
      ["bb-agent-state-v2.legacy", "interim"],
      ["bb-agent-state-v2-migration-owner", "owner"],
      ["bb-agent-state-v2.example", "workspace"],
      ["bb-agent-state-v2.7:example.member@example.com", "v2"],
      ["bb-agent-state-v3.5:other.other@example.com", "not JSON"],
      ["bb-agent-preferences-v1.7:example.member@example.com", "preferences"],
      ["bb-agent-window", "window"],
      ["bb-agent-window.example", "layout"],
      ["bb-agent-ui-v1.bb-agent-state-v3.example", "shadow"],
    ];
    const storage = storageWith([
      ...entries,
      ["bb-agent-state-v30.example", "unrelated"],
      ["bb-agent-state-v3-backup", "unrelated"],
      ["bb-agent-unknown", "unrelated"],
      ["application-setting", "unrelated"],
    ]);

    expect(estimateAgentStorageBytes(storage)).toBe(
      entries.reduce((sum, entry) => sum + entryBytes(entry), 0)
    );
    expect(vi.mocked(storage.getItem).mock.calls.map(([key]) => key)).toEqual(
      entries.map(([key]) => key)
    );
    expect(storage.setItem).not.toHaveBeenCalled();
    expect(storage.removeItem).not.toHaveBeenCalled();
    expect(storage.clear).not.toHaveBeenCalled();
  });

  it("replaces the actual encoded value once and includes a new recognized key", () => {
    const first: [string, string] = ["bb-agent-state-v3.first", "before"];
    const other: [string, string] = ["bb-agent-state-v3.other", "protected"];
    const storage = storageWith([first, other]);

    expect(
      estimateAgentStorageBytes(storage, { key: first[0], value: "after" })
    ).toBe(entryBytes([first[0], "after"]) + entryBytes(other));
    expect(
      estimateAgentStorageBytes(storage, {
        key: "bb-agent-state-v3.new",
        value: "new",
      })
    ).toBe(
      entryBytes(first) +
        entryBytes(other) +
        entryBytes(["bb-agent-state-v3.new", "new"])
    );
    expect(
      estimateAgentStorageBytes(storage, { key: "unrelated", value: "ignored" })
    ).toBe(entryBytes(first) + entryBytes(other));
  });

  it("uses UTF-16 key and encoded value lengths, including compressed contents", () => {
    const key = "bb-agent-state-v3.\u90ae\u7bb1";
    const plain = JSON.stringify({
      content: "\ud83d\ude80\u00e9\ud800".repeat(300),
    });
    const encoded = compressToUTF16(plain);
    const storage = storageWith([[key, plain]]);

    expect(estimateAgentStorageBytes(storage)).toBe(
      2 * (key.length + plain.length)
    );
    expect(estimateAgentStorageBytes(storage, { key, value: encoded })).toBe(
      2 * (key.length + encoded.length)
    );
    expect(encoded.length).toBeLessThan(plain.length);
  });

  it.each([2 * 1024 * 1024, 3 * 1024 * 1024])(
    "measures the exact %i-byte boundary and one code unit beyond it",
    (limit) => {
      const key = "bb-agent-state-v3.example";
      const value = "x".repeat(limit / 2 - key.length);
      const storage = storageWith([[key, value]]);
      expect(estimateAgentStorageBytes(storage)).toBe(limit);
      expect(
        estimateAgentStorageBytes(storage, { key, value: `${value}x` })
      ).toBe(limit + 2);
    }
  );

  it("does not decode or expose another scope's raw contents", () => {
    const key = "bb-agent-state-v3.5:other.other@example.com";
    const storage = storageWith([[key, "{not a decodable state"]]);

    expect(estimateAgentStorageBytes(storage)).toBe(
      entryBytes([key, "{not a decodable state"])
    );
  });

  it("propagates unavailable storage rather than treating it as empty", () => {
    const storage = storageWith([["bb-agent-state-v3.example", "value"]]);
    vi.mocked(storage.getItem).mockImplementation(() => {
      throw new Error("Storage unavailable");
    });
    expect(() => estimateAgentStorageBytes(storage)).toThrow(
      "Storage unavailable"
    );
  });
});

describe("AgentStorageFullError", () => {
  it.each(["budget", "quota"] as const)(
    "carries %s pressure and estimates without including stored content",
    (reason) => {
      const error = new AgentStorageFullError(reason, 100, 80);
      expect(error).toBeInstanceOf(Error);
      expect(error.name).toBe("AgentStorageFullError");
      expect(error.reason).toBe(reason);
      expect(error.bytes).toBe(100);
      expect(error.limitBytes).toBe(80);
      expect(error.message).toContain("100 bytes");
      if (reason === "budget") expect(error.message).toContain("80 bytes");
      else {
        expect(error.message).toContain("browser");
        expect(error.message).not.toContain("80 bytes");
      }
    }
  );

  it("exports the configured admission and ceiling limits", () => {
    expect(AGENT_STORAGE_ADMISSION_BYTES).toBe(2 * 1024 * 1024);
    expect(AGENT_STORAGE_LIMIT_BYTES).toBe(3 * 1024 * 1024);
    expect(new AgentStorageFullError("quota", 1).limitBytes).toBe(
      AGENT_STORAGE_LIMIT_BYTES
    );
  });
});
