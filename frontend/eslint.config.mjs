import { writeFileSync } from "fs";
import { resolve } from "path";
import vueI18n from "@intlify/eslint-plugin-vue-i18n";
import vueTsEslintConfig from "@vue/eslint-config-typescript";
import i18nextNoUndefined from "eslint-plugin-i18next-no-undefined-translation-keys";
import pluginVue from "eslint-plugin-vue";

// Generate namespace mapping with absolute path for the no-undefined-translation-keys plugin.
// The plugin uses require() internally, which needs absolute paths.
const reactLocalesDir = resolve(import.meta.dirname, "src/react/locales");
const namespaceMappingPath = resolve(
  import.meta.dirname,
  "react-i18n-namespace-mapping.json"
);
writeFileSync(
  namespaceMappingPath,
  JSON.stringify({ default: resolve(reactLocalesDir, "en-US.json") }) + "\n"
);

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
  // React i18n: detect missing translation keys in .tsx files
  {
    files: ["src/react/**/*.tsx"],
    plugins: {
      "i18next-no-undefined-translation-keys": i18nextNoUndefined,
    },
    rules: {
      "i18next-no-undefined-translation-keys/no-undefined-translation-keys": [
        "error",
        {
          namespaceTranslationMappingFile: namespaceMappingPath,
          defaultNamespace: "default",
        },
      ],
    },
  },
  {
    rules: {
      "no-console": ["error", { allow: ["warn", "error", "debug", "assert"] }],
      "no-debugger": "error",
      "no-empty-pattern": "error",
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
          ignores: [
            // Used in React .tsx — vue-i18n linter can't detect these
            "project.batch.selected",
            "project.batch.archive.title",
            "project.batch.archive.success",
            "project.batch.delete.title",
            "project.batch.delete.success",
          ],
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
];
