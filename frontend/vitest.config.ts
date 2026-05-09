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
      exclude: [...configDefaults.exclude, "e2e/*", "tests/e2e/**"],
      root: fileURLToPath(new URL("./", import.meta.url)),
      setupFiles: ["./vitest.setup.ts"],
    },
  })
);
