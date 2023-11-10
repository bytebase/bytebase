import VueI18nPlugin from "@intlify/unplugin-vue-i18n/vite";
import yaml from "@rollup/plugin-yaml";
import vue from "@vitejs/plugin-vue";
import { CodeInspectorPlugin } from "code-inspector-plugin";
import { resolve } from "path";
import IconsResolver from "unplugin-icons/resolver";
import Icons from "unplugin-icons/vite";
import Components from "unplugin-vue-components/vite";
import { defineConfig } from "vite";

const SERVER_PORT = parseInt(process.env.PORT ?? "3000", 10) ?? 3000;
const HTTPS_PORT = 443;
const LOCAL_ENDPOINT = "http://localhost:8080";

export default defineConfig(() => {
  // NOTE: the following lines is to solve https://github.com/gitpod-io/gitpod/issues/6719
  // tl;dr : the HMR(hot module replacement) will behave differently when VPN is on, and by manually set its port to 443 should prevent this issue.
  const IS_RUNNING_GITPOD =
    process.env["GITPOD_WORKSPACE_ID"] !== null &&
    process.env["GITPOD_WORKSPACE_ID"] !== undefined;

  return {
    plugins: [
      vue(),
      // https://github.com/intlify/vite-plugin-vue-i18n
      VueI18nPlugin({
        include: [resolve(__dirname, "src/locales/**")],
        strictMessage: false,
      }),
      Components({
        dirs: [resolve("src/components"), resolve("src/bbkit")],
        // auto import icons
        resolvers: [
          IconsResolver({
            prefix: "",
          }),
        ],
      }),
      Icons(),
      yaml(),
      CodeInspectorPlugin({
        bundler: "vite",
      }),
    ],
    build: {
      rollupOptions: {
        input: {
          main: resolve(__dirname, "index.html"),
          "explain-visualizer": resolve(__dirname, "explain-visualizer.html"),
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
        port: IS_RUNNING_GITPOD ? HTTPS_PORT : SERVER_PORT,
      },
    },
    resolve: {
      alias: {
        "@/": `${resolve(__dirname, "src")}/`,
        "@sql-lsp/": `${resolve(__dirname, "src/plugins/sql-lsp")}/`,
        "@public/": `${resolve(__dirname, "public")}/`,
      },
    },
    test: {
      global: true,
      environment: "node",
    },
    envPrefix: "BB_",
  };
});

const extractHostPort = (url: string) => {
  const parsed = new URL(url);
  return parsed.host;
};
