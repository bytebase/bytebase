import { fileURLToPath, URL } from "node:url";
import importMetaUrlPlugin from "@codingame/esbuild-import-meta-url-plugin";
import VueI18nPlugin from "@intlify/unplugin-vue-i18n/vite";
import yaml from "@rollup/plugin-yaml";
import tailwindcss from "@tailwindcss/vite";
import legacy from "@vitejs/plugin-legacy";
import vue from "@vitejs/plugin-vue";
import vueJsx from "@vitejs/plugin-vue-jsx";
import { CodeInspectorPlugin } from "code-inspector-plugin";
import { resolve } from "path";
import IconsResolver from "unplugin-icons/resolver";
import Icons from "unplugin-icons/vite";
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
    vue(),
    vueJsx(),
    // https://github.com/intlify/vite-plugin-vue-i18n
    VueI18nPlugin({
      include: [resolve(__dirname, "src/locales/**")],
      strictMessage: false,
    }),
    tailwindcss(),
    Components({
      allowOverrides: true,
      // auto import icons
      resolvers: [
        IconsResolver({
          prefix: "",
        }),
      ],
    }),
    Icons({
      compiler: "vue3",
    }),
    yaml(),
    CodeInspectorPlugin({
      bundler: "vite",
    }),
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
    esbuildOptions: {
      plugins: [importMetaUrlPlugin],
    },
  },
  envPrefix: ["BB_", "GIT_COMMIT"],
  define: {
    _global: {},
  },
  worker: {
    format: "es",
  },
});
