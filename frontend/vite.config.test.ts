// @vitest-environment node
import { fileURLToPath } from "node:url";
import { build } from "vite";
import { expect, test } from "vitest";
import config from "./vite.config";

test("refresh worker keeps its URL across builds with different worker contents", async () => {
  const bundles = [];
  for (const version of ["old", "new"]) {
    const result = await build({
      configFile: false,
      logLevel: "silent",
      resolve: config.resolve,
      worker: {
        ...config.worker,
        plugins: () => [
          {
            name: "worker-version-fixture",
            transform(code, id) {
              if (id.endsWith("/refreshToken.worker.ts")) {
                return `${code}\nconsole.info(${JSON.stringify(version)});`;
              }
            },
          },
        ],
      },
      build: {
        write: false,
        rollupOptions: {
          input: fileURLToPath(
            new URL("./src/api/refreshTokenSharedWorker.ts", import.meta.url)
          ),
          preserveEntrySignatures: "strict",
        },
      },
    });
    if (Array.isArray(result) || !("output" in result)) {
      throw new Error("Expected one production bundle");
    }
    bundles.push(result.output);
  }

  const workers = bundles.map((bundle) => {
    const worker = bundle.find(
      (output) => output.fileName === "refreshToken.worker.js"
    );
    expect(worker).toBeDefined();
    return worker;
  });
  expect(workers[0]).not.toEqual(workers[1]);
  const entries = bundles.map((bundle) =>
    bundle.find((output) => output.type === "chunk" && output.isEntry)
  );
  expect(entries[0]).toEqual(entries[1]);
});
