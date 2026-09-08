import { fileURLToPath } from "node:url";
import { configDefaults, defineConfig, mergeConfig } from "vitest/config";
import viteConfig from "./vite.config";

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      globals: true,
      // Worker threads reuse a V8 isolate per file instead of forking a
      // process, which cuts ~10% off the suite (82s -> 74s). Each file still
      // gets its own jsdom and module registry; only the process is shared.
      pool: "threads",
      environment: "jsdom",
      // jsdom rejects localStorage / sessionStorage access when the document
      // origin is opaque (the default for the about:blank URL), so give it a
      // concrete URL.
      environmentOptions: {
        jsdom: {
          url: "http://localhost/",
        },
      },
      // Vitest's 5s/10s defaults are tuned for a fast machine. Setting up a
      // jsdom environment and rendering a page in one of the heavier
      // beforeEach hooks runs comfortably under them locally but not on the
      // self-hosted CI runner, where the suite is several times slower. These
      // are still far below any real hang.
      testTimeout: 15000,
      hookTimeout: 30000,
      // @stylexjs/unplugin (0.19.0, the current release) leaves a handle open
      // that never lets the Vite server close, so every run -- including a
      // single-file one -- used to sit for the default 10s on "close timed
      // out" after the results were already reported. Tests need the StyleX
      // transform (skipping the plugin fails 82 files), so bound the wait
      // instead: vitest force-exits either way, this just stops it idling.
      teardownTimeout: 1000,
      exclude: [...configDefaults.exclude, "e2e/*", "tests/e2e/**"],
      root: fileURLToPath(new URL("./", import.meta.url)),
      setupFiles: ["./vitest.setup.ts"],
      globalSetup: ["./vitest.globalSetup.ts"],
    },
  })
);
