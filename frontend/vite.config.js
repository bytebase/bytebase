// const path = require("path");
import vue from "@vitejs/plugin-vue";

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

// NOTE: the following lines is to solve https://github.com/gitpod-io/gitpod/issues/6719
// tl;dr : the HMR(hot module replacement) will behave differently when VPN is on, and by manually set its prot to 443 should prevent this issue. 
const IS_RUNNING_GITPOD = process.env["GITPOD_WORKSPACE_ID"] !== null;
if (IS_RUNNING_GITPOD === true) {
  module.exports.server.hmr = { port: 443 };
}
