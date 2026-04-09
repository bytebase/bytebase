import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, describe, expect, test } from "vitest";
import {
  normalizeLocaleFile,
  sortLocaleFiles,
  sortObjectKeys,
} from "./sort_i18n_keys.mjs";

let tempDir = "";

afterEach(() => {
  if (tempDir) {
    rmSync(tempDir, { recursive: true, force: true });
    tempDir = "";
  }
});

describe("sortObjectKeys", () => {
  test("sorts nested object keys recursively and preserves arrays", () => {
    expect(
      sortObjectKeys({
        z: "last",
        a: {
          d: "delta",
          b: "bravo",
          nested: {
            c: "charlie",
            a: "alpha",
          },
        },
        m: ["keep", { y: 2, x: 1 }],
      })
    ).toEqual({
      a: {
        b: "bravo",
        d: "delta",
        nested: {
          a: "alpha",
          c: "charlie",
        },
      },
      m: ["keep", { y: 2, x: 1 }],
      z: "last",
    });
  });
});

describe("normalizeLocaleFile", () => {
  test("rewrites files with stable 2-space JSON and trailing newline", () => {
    tempDir = mkdtempSync(join(tmpdir(), "locale-sorter-"));
    const filePath = join(tempDir, "en-US.json");
    writeFileSync(filePath, '{"z":1,"a":{"d":4,"b":2},"m":[{"y":2,"x":1}]}');

    expect(normalizeLocaleFile(filePath)).toBe(true);
    expect(readFileSync(filePath, "utf-8")).toBe(
      [
        "{",
        '  "a": {',
        '    "b": 2,',
        '    "d": 4',
        "  },",
        '  "m": [',
        "    {",
        '      "y": 2,',
        '      "x": 1',
        "    }",
        "  ],",
        '  "z": 1',
        "}",
        "",
      ].join("\n")
    );
  });

  test("does not rewrite already normalized files and fails fast on invalid JSON", () => {
    tempDir = mkdtempSync(join(tmpdir(), "locale-sorter-"));
    const validPath = join(tempDir, "ja-JP.json");
    const invalidPath = join(tempDir, "broken.json");

    const normalized = [
      "{",
      '  "a": {',
      '    "b": 2,',
      '    "d": 4',
      "  },",
      '  "z": 1',
      "}",
      "",
    ].join("\n");
    writeFileSync(validPath, normalized);
    writeFileSync(invalidPath, '{"a":');

    expect(normalizeLocaleFile(validPath)).toBe(false);
    expect(readFileSync(validPath, "utf-8")).toBe(normalized);
    expect(() => normalizeLocaleFile(invalidPath)).toThrow(
      /Failed to parse locale file .*broken\.json/
    );
  });

  test("sortLocaleFiles validates every file before writing any change", () => {
    const validPath = "/tmp/locale-sorter-valid.json";
    const invalidPath = "/tmp/locale-sorter-invalid.json";
    const writes = [];
    const readStub = (filePath) => {
      if (filePath === validPath) {
        return '{"z":1,"a":2}';
      }
      if (filePath === invalidPath) {
        return '{"a":';
      }
      throw new Error(`Unexpected file read: ${String(filePath)}`);
    };
    const writeStub = (...args) => {
      writes.push(args);
    };

    expect(
      () =>
        sortLocaleFiles([validPath, invalidPath], {
          readFileSync: readStub,
          writeFileSync: writeStub,
        })
    ).toThrow(/Failed to parse locale file .*locale-sorter-invalid\.json/);
    expect(writes).toEqual([]);
  });

  test("sortLocaleFiles rolls back earlier writes when a later write fails", () => {
    const firstPath = "/tmp/locale-sorter-first.json";
    const secondPath = "/tmp/locale-sorter-second.json";
    const contents = new Map([
      [firstPath, '{"z":1,"a":2}'],
      [secondPath, '{"y":1,"b":2}'],
    ]);
    const writes = [];
    const readStub = (filePath) => {
      const content = contents.get(filePath);
      if (content === undefined) {
        throw new Error(`Unexpected file read: ${String(filePath)}`);
      }
      return content;
    };
    const writeStub = (filePath, content) => {
      writes.push([filePath, content]);
      if (filePath === secondPath && content.includes('"b": 2')) {
        throw new Error("disk full");
      }
      contents.set(filePath, content);
    };

    expect(
      () =>
        sortLocaleFiles([firstPath, secondPath], {
          readFileSync: readStub,
          writeFileSync: writeStub,
        })
    ).toThrow(/Failed to write locale file .*locale-sorter-second\.json: disk full\. Rolled back 1 earlier file\(s\)\./);
    expect(contents.get(firstPath)).toBe('{"z":1,"a":2}');
    expect(contents.get(secondPath)).toBe('{"y":1,"b":2}');
    expect(writes).toEqual([
      [firstPath, '{\n  "a": 2,\n  "z": 1\n}\n'],
      [secondPath, '{\n  "b": 2,\n  "y": 1\n}\n'],
      [firstPath, '{"z":1,"a":2}'],
    ]);
  });
});
