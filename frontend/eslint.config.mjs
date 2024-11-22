import { FlatCompat } from "@eslint/eslintrc";
import js from "@eslint/js";
import pluginVue from "eslint-plugin-vue";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const compat = new FlatCompat({
  baseDirectory: __dirname,
  recommendedConfig: js.configs.recommended,
});

// Reference: https://github.com/vuejs/eslint-config-typescript/issues/76#issuecomment-2051234597
export default [
  js.configs.recommended,
  ...pluginVue.configs["flat/recommended"],
  ...compat.extends("@vue/eslint-config-typescript/recommended"),
  ...compat.extends("@vue/eslint-config-prettier/skip-formatting"),
  {
    files: ["**/*.js", "**/*.vue", "**/*.ts", "**/*.tsx"],
    languageOptions: {
      sourceType: "module",
    },
  },
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
        { varsIgnorePattern: "^_", args: "none" },
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
            /^mdi:/,
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
