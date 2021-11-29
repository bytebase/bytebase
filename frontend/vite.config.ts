import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

const SERVER_PORT = process.env.PORT || 3000;
const HTTPS_PORT = 443;

export default defineConfig(() => {
  // NOTE: the following lines is to solve https://github.com/gitpod-io/gitpod/issues/6719
  // tl;dr : the HMR(hot module replacement) will behave differently when VPN is on, and by manually set its port to 443 should prevent this issue.
  const IS_RUNNING_GITPOD =
    process.env["GITPOD_WORKSPACE_ID"] !== null &&
    process.env["GITPOD_WORKSPACE_ID"] !== undefined;
  return {
    plugins: [vue()],
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
      // alias: {
      //   "/@/": path.resolve(__dirname, "./src"),
      // },
    },
  };
});
