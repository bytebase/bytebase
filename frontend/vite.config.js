// const path = require("path");
import vue from "@vitejs/plugin-vue";

module.exports = {
  plugins: [vue()],
  optimizeDeps: {
    allowNodeBuiltins: ["postcss", "bytebase"],
  },
  proxy: {
    // // string shorthand
    // '/foo': 'http://localhost:4567/foo',
    // // with options
    // '/api': {
    //     target: 'http://jsonplaceholder.typicode.com',
    //     changeOrigin: true,
    //     rewrite: path => path.replace(/^\/api/, '')
    // }
  },
  resolve: {
    // alias: {
    //   "/@/": path.resolve(__dirname, "./src"),
    // },
  },
};
