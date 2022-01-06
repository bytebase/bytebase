import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import { resolve } from "path";
import VueI18n from "@intlify/vite-plugin-vue-i18n";
import ViteIcons, { ViteIconsResolver } from "vite-plugin-icons";
import ViteComponents from "vite-plugin-components";

const SERVER_PORT = process.env.PORT || 3000;
const HTTPS_PORT = 443;
const r = (...args: string[]) => resolve(__dirname, ...args);

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
      VueI18n({
        include: [resolve(__dirname, "src/locales/**")],
      }),
      ViteComponents({
        dirs: [r("src/components"), r("src/bbkit")],
        // generate `components.d.ts` for ts support with Volar
        globalComponentsDeclaration: true,
        // auto import icons
        customComponentResolvers: [
          // https://github.com/antfu/vite-plugin-icons
          ViteIconsResolver({
            componentPrefix: "",
          }),
        ],
      }),
      ViteIcons(),
    ],
    optimizeDeps: {
      allowNodeBuiltins: ["postcss", "bytebase"],
    },
    server: {
      port: SERVER_PORT,
      proxy: {
        "/api": {
          target: "http://localhost:8080/api",
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, ""),
        },
      },
      hmr: {
        port: IS_RUNNING_GITPOD ? HTTPS_PORT : SERVER_PORT,
      },
    },
    resolve: {
      alias: {
        "@/": `${resolve(__dirname, "src")}/`,
      },
    },
  };
});
