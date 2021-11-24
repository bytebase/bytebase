// const path = require("path");
import vue from "@vitejs/plugin-vue";
import { resolve } from 'path';
import { defineConfig } from 'vite'
import VueI18n from '@intlify/vite-plugin-vue-i18n';
import ViteIcons, { ViteIconsResolver } from 'vite-plugin-icons';
import ViteComponents from 'vite-plugin-components';

const r = (...args: string[]) => resolve(__dirname, ...args)

export default defineConfig({
  plugins: [
    vue(),
    // https://github.com/intlify/vite-plugin-vue-i18n
    VueI18n({
      include: [resolve(__dirname, 'src/locales/**')],
    }),
    ViteComponents({
      dirs: [r('src/components'), r('src/bbkit')],
      // generate `components.d.ts` for ts support with Volar
      globalComponentsDeclaration: true,
      // auto import icons
      customComponentResolvers: [
        // https://github.com/antfu/vite-plugin-icons
        ViteIconsResolver({
          componentPrefix: '',
        }),
      ],
    }),
    ViteIcons()
  ],
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
  }
});
