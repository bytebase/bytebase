import { fileURLToPath } from "node:url";
import { configDefaults, defineConfig, mergeConfig } from "vitest/config";
import viteConfig from "./vite.config";

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      globals: true,
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
      exclude: [...configDefaults.exclude, "e2e/*", "tests/e2e/**"],
      root: fileURLToPath(new URL("./", import.meta.url)),
      setupFiles: ["./vitest.setup.ts"],
    },
  })
);
