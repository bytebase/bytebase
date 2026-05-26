import vueI18n from "@intlify/eslint-plugin-vue-i18n";
import vueTsEslintConfig from "@vue/eslint-config-typescript";
import pluginVue from "eslint-plugin-vue";

export default [
  ...pluginVue.configs["flat/essential"],
  ...vueTsEslintConfig({
    extends: ["recommended"],
    supportedScriptLangs: {
      ts: true,
      tsx: true,
    },
    rootDir: import.meta.dirname,
  }),
  ...vueI18n.configs["flat/recommended"],
  {
    ignores: ["**/dist/**", "**/node_modules/**", "**/proto-es/**"],
  },
  {
    rules: {
      "no-console": ["error", { allow: ["warn", "error", "debug", "assert"] }],
      "no-debugger": "error",
      "no-empty-pattern": "error",
      "no-restricted-properties": [
        "error",
        {
          object: "crypto",
          property: "randomUUID",
          message: "Use v4 from 'uuid' package instead of crypto.randomUUID() for compatibility.",
        },
      ],
      "vue/no-ref-as-operand": "error",
      "no-useless-escape": "error",
      "@typescript-eslint/no-empty-interface": "error",
      "@typescript-eslint/no-unused-vars": [
        "error",
        { varsIgnorePattern: "^_", argsIgnorePattern: "^_" },
      ],
      "@intlify/vue-i18n/no-unused-keys": [
        "error",
        {
          src: "./src",
          extensions: [".js", ".vue", ".ts", ".tsx"],
          enableFix: true,
        },
      ],
      "@intlify/vue-i18n/no-missing-keys": "error",
      "@intlify/vue-i18n/no-raw-text": "off",
      "@typescript-eslint/no-explicit-any": "error",
      "vue/no-mutating-props": "error",
      "vue/no-unused-components": "error",
      "vue/no-useless-template-attributes": "error",
      "vue/no-undef-components": [
        "error",
        {
          ignorePatterns: [
            /^heroicons(-solid|-outline)?:/,
            /^carbon:/,
            /^tabler:/,
            /^octicon:/,
            /^router-view$/,
            /^router-link$/,
            /^i18n-t$/,
            /^highlight-code-block$/,
          ],
        },
      ],
      "vue/multi-word-component-names": "off",
    },
    settings: {
      "vue-i18n": {
        localeDir: "./src/locales/*.json",
        messageSyntaxVersion: "^9.0.0",
      },
    },
  },
  // React code uses its own locale files (src/react/locales/) loaded through
  // react-i18next, so the vue-i18n linter has no visibility into those keys.
  // Disable missing-keys checks for every React surface — .ts and .tsx under
  // both src/react/ and src/plugins/ai/react/ (the AI plugin's React tree).
  {
    files: [
      "src/react/**/*.ts",
      "src/react/**/*.tsx",
      "src/plugins/ai/react/**/*.ts",
      "src/plugins/ai/react/**/*.tsx",
    ],
    rules: {
      "@intlify/vue-i18n/no-missing-keys": "off",
    },
  },
];
