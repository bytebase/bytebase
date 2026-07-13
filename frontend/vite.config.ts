import { fileURLToPath, URL } from "node:url";
import { transform as esbuildTransform } from "esbuild";
import yaml from "@rollup/plugin-yaml";
import stylex from "@stylexjs/unplugin";
import tailwindcss from "@tailwindcss/vite";
import legacy from "@vitejs/plugin-legacy";
import { CodeInspectorPlugin } from "code-inspector-plugin";
import { resolve } from "path";
import { defineConfig } from "vite";
import { exportCspHashes } from "./vite-plugin-export-csp-hashes";

const SERVER_PORT = parseInt(process.env.PORT ?? "3000", 10) ?? 3000;
const LOCAL_ENDPOINT = "http://localhost:8080";

const extractHostPort = (url: string) => {
  const parsed = new URL(url);
  return parsed.host;
};

export default defineConfig({
  plugins: [
    legacy({
      // Explicit version floors, matching what "> 0.08%, not dead" resolved
      // to as of caniuse-lite 1.0.30001792. Usage-based queries re-resolve
      // against every caniuse-lite/browserslist update; a 2026-07 data update
      // pulled Chrome 39-60 above the threshold, which made the babel pass
      // in plugin-legacy's renderChunk down-level every chunk to ES5 and
      // took the release build from ~2.5min to ~32min (13x).
      targets: [
        "chrome >= 103",
        "edge >= 100",
        "firefox >= 115",
        "safari >= 15",
        "ios >= 11",
      ],
      additionalLegacyPolyfills: ["regenerator-runtime/runtime"],
    }),
    {
      name: "react-tsx-transform",
      enforce: "pre",
      async transform(code, id) {
        // Both the main React tree (`src/react/...`) and the AI plugin's
        // co-located React subtree (`src/plugins/ai/react/...`) compile
        // with React's automatic JSX runtime.
        if (!/\/(src\/react|src\/plugins\/ai\/react)\/.+\.tsx$/.test(id)) {
          return undefined;
        }
        const result = await esbuildTransform(code, {
          loader: "tsx",
          jsx: "automatic",
          jsxImportSource: "react",
          tsconfigRaw: { compilerOptions: {} },
          sourcemap: true,
          sourcefile: id,
        });
        return { code: result.code, map: result.map || null };
      },
    },
    tailwindcss(),
    stylex.vite({
      // Keep StyleX after Tailwind so production extraction can append into the
      // linked Vite CSS asset loaded by the main app.
      cssInjectionTarget: (fileName) => /(^|\/)main-[^/]+\.css$/.test(fileName),
      devMode: "css-only",
      runtimeInjection: false,
      useCSSLayers: true,
    }),
    yaml(),
    ...(process.env.VITEST
      ? []
      : [
          CodeInspectorPlugin({
            bundler: "vite",
            exclude: [/src\/react\//],
          }),
        ]),
    // Export CSP hashes from @vitejs/plugin-legacy for backend to use
    exportCspHashes(),
  ],
  build: {
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      input: {
        main: resolve(__dirname, "index.html"),
        "explain-visualizer": resolve(__dirname, "explain-visualizer.html"),
      },
      output: {
        manualChunks: (id) => {
          const normalizedId = id.replaceAll("\\", "/");
          if (
            normalizedId.includes("/node_modules/vue/") ||
            normalizedId.includes("/node_modules/@vue/") ||
            normalizedId.includes("/node_modules/pev2/")
          ) {
            return "explain-visualizer-vue";
          }
          // Monaco Editor - separate chunk
          if (id.includes("monaco-editor") || id.includes("monaco-vscode")) {
            return "monaco-editor";
          }
          // SQL tools - separate chunk
          if (id.includes("sql-formatter") || id.includes("antlr4")) {
            return "sql-tools";
          }
          // Utilities
          if (id.includes("lodash") || id.includes("dayjs")) {
            return "utils";
          }
        },
      },
    },
  },
  server: {
    port: SERVER_PORT,
    host: "0.0.0.0",
    proxy: {
      "/v1:adminExecute": {
        target: `ws://${extractHostPort(LOCAL_ENDPOINT)}/`,
        changeOrigin: true,
        ws: true,
      },
      "/lsp": {
        target: `ws://${extractHostPort(LOCAL_ENDPOINT)}/`,
        changeOrigin: true,
        ws: true,
      },
      "/api": {
        target: `${LOCAL_ENDPOINT}/api`,
        changeOrigin: true,
        rewrite: (path: string) => path.replace(/^\/api/, ""),
      },
      "/hook": {
        target: LOCAL_ENDPOINT,
        changeOrigin: true,
      },
      "/v1": {
        target: `${LOCAL_ENDPOINT}/v1`,
        changeOrigin: true,
        rewrite: (path: string) => path.replace(/^\/v1/, ""),
      },
    },
    hmr: {
      port: SERVER_PORT,
    },
  },
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  optimizeDeps: {
    include: ["vscode-textmate", "vscode-oniguruma"],
  },
  envPrefix: ["BB_", "GIT_COMMIT"],
  define: {
    _global: {},
  },
  worker: {
    format: "es",
  },
});
