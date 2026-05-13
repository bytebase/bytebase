import { fileURLToPath, URL } from "node:url";
import { transform as esbuildTransform } from "esbuild";
import VueI18nPlugin from "@intlify/unplugin-vue-i18n/vite";
import yaml from "@rollup/plugin-yaml";
import tailwindcss from "@tailwindcss/vite";
import legacy from "@vitejs/plugin-legacy";
import vue from "@vitejs/plugin-vue";
import vueJsx from "@vitejs/plugin-vue-jsx";
import { CodeInspectorPlugin } from "code-inspector-plugin";
import { resolve } from "path";
import Components from "unplugin-vue-components/vite";
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
      targets: ["> 0.08%, not dead"],
      additionalLegacyPolyfills: ["regenerator-runtime/runtime"],
    }),
    {
      name: "react-tsx-transform",
      enforce: "pre",
      async transform(code, id) {
        // Both the main React tree (`src/react/...`) and the AI plugin's
        // co-located React subtree (`src/plugins/ai/react/...`) compile
        // with React's automatic JSX runtime. The latter has to be
        // listed explicitly — otherwise its `.tsx` files fall through
        // to the `vueJsx()` plugin below and get compiled with Vue's
        // JSX transform, which produces Vue VNodes and dies at runtime
        // with "Cannot add property _ctx, object is not extensible"
        // when Vue tries to attach `_ctx` to a frozen value.
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
    vue(),
    vueJsx({
      include: /\.tsx$/,
      exclude: /src\/(react|plugins\/ai\/react)\//,
    }),
    // https://github.com/intlify/vite-plugin-vue-i18n
    VueI18nPlugin({
      include: [resolve(__dirname, "src/locales/**")],
      strictMessage: false,
    }),
    tailwindcss(),
    Components({
      allowOverrides: true,
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
          // Monaco Editor - separate chunk
          if (id.includes("monaco-editor") || id.includes("monaco-vscode")) {
            return "monaco-editor";
          }
          // SQL tools - separate chunk
          if (id.includes("sql-formatter") || id.includes("antlr4")) {
            return "sql-tools";
          }
          // UI framework
          if (id.includes("naive-ui")) {
            return "ui-framework";
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
