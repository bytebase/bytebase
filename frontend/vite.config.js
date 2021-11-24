// const path = require("path");
import vue from "@vitejs/plugin-vue";

const IS_RUNNING_GITPOD = process.env["GITPOD_WORKSPACE_ID"] !== null;

module.exports = {
  plugins: [vue()],
  optimizeDeps: {
    allowNodeBuiltins: ["postcss", "bytebase"],
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080/api",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ""),
      },
    },
  },
  resolve: {
    // alias: {
    //   "/@/": path.resolve(__dirname, "./src"),
    // },
  },
};

if (IS_RUNNING_GITPOD === true) {
  module.exports.server.hmr = { port: 443 };
}
