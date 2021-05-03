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
