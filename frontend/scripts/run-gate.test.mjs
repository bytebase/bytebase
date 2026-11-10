// @vitest-environment node
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";

const GATE = fileURLToPath(new URL("./run-gate.mjs", import.meta.url));

// Runs `finish` in a child whose stdout is a pipe, as it is on CI. A stage's
// `padBytes` is expanded in the child, ahead of its output, to exceed the pipe
// buffer without hitting the argv size limit.
function finishThroughPipe(results) {
  const script = `
    import { finish } from ${JSON.stringify(GATE)};
    const results = ${JSON.stringify(results)}.map(({ padBytes = 0, ...r }) => ({
      ...r,
      output: "x".repeat(padBytes) + r.output,
    }));
    finish(results);
  `;
  return new Promise((done, fail) => {
    const child = spawn(process.execPath, ["--input-type=module", "-e", script], {
      stdio: ["ignore", "pipe", "inherit"],
    });
    const chunks = [];
    child.stdout.on("data", (c) => chunks.push(c));
    child.on("error", fail);
    child.on("close", (code) => done({ code, stdout: Buffer.concat(chunks).toString() }));
  });
}

describe("run-gate finish", () => {
  test("a failing stage's full output reaches a piped stdout, last, before the summary", async () => {
    const { code, stdout } = await finishThroughPipe([
      { name: "biome", code: 0, output: "biome ok\n" },
      {
        name: "vitest",
        code: 1,
        padBytes: 4 * 1024 * 1024,
        output: "\nFAIL src/example.test.tsx > the failing case\n",
      },
      { name: "tsc", code: 0, output: "tsc ok\n" },
    ]);

    expect(code).toBe(1);
    expect(stdout).toContain("FAIL src/example.test.tsx > the failing case");
    expect(stdout.indexOf("tsc ok")).toBeLessThan(stdout.indexOf("\nvitest\n"));
    expect(stdout.endsWith("\n1 stage(s) failed: vitest\n")).toBe(true);
  });

  test("a passing gate exits 0 and prints every stage", async () => {
    const { code, stdout } = await finishThroughPipe([
      { name: "biome", code: 0, output: "biome ok\n" },
      { name: "tsc", code: 0, output: "tsc ok\n" },
    ]);

    expect(code).toBe(0);
    expect(stdout).toContain("biome ok");
    expect(stdout).toContain("tsc ok");
    expect(stdout).not.toContain("stage(s) failed");
  });
});
