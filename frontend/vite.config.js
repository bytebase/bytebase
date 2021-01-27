// const path = require("path");

module.exports = {
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
  alias: {
    // "/@/": path.resolve(__dirname, "./src"),
  },
};
