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
  {
    ignores: ["**/dist/**", "**/node_modules/**", "**/proto/**"],
  },
  {
    rules: {
      "no-empty-pattern": "error",
      "vue/no-ref-as-operand": "error",
      "no-useless-escape": "error",
      "@typescript-eslint/no-empty-interface": "error",
      "@typescript-eslint/no-unused-vars": [
        "error",
        { varsIgnorePattern: "^_", argsIgnorePattern: "^_" },
      ],
      "@typescript-eslint/no-explicit-any": "off",
      "vue/no-mutating-props": "error",
      "vue/no-unused-components": "error",
      "vue/no-useless-template-attributes": "error",
      "vue/no-undef-components": [
        "warn",
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
  },
];
