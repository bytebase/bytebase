module.exports = {
  env: {
    "vue/setup-compiler-macros": true,
  },
  parser: "vue-eslint-parser",
  parserOptions: {
    parser: "@typescript-eslint/parser",
    sourceType: "module",
  },
  extends: [
    "eslint:recommended",
    "plugin:vue/vue3-recommended",
    "@vue/typescript/recommended",
    "plugin:prettier/recommended",
  ],
  rules: {
    "no-empty-pattern": "error",
    "no-useless-escape": "error",
    "prettier/prettier": [
      "error",
      {
        useTabs: false,
        tabWidth: 2,
        singleQuote: false,
        trailingComma: "es5",
        printWidth: 80,
      },
    ],
    "@typescript-eslint/consistent-type-imports": [
      "error",
      { fixStyle: "separate-type-imports", prefer: "type-imports" },
    ],
    "@typescript-eslint/no-empty-interface": "error",
    "@typescript-eslint/no-unused-vars": [
      "error",
      { varsIgnorePattern: "^_", args: "none" },
    ],
    "vue/no-mutating-props": "error",
    "vue/no-unused-components": "error",
    "vue/no-useless-template-attributes": "error",
    "@typescript-eslint/no-explicit-any": "off",
    "@typescript-eslint/no-non-null-assertion": "off",
    "vue/multi-word-component-names": "off",
  },
  ignorePatterns: [
    "node_modules",
    "build",
    "dist",
    "public",
    "components.d.ts",
  ],
  overrides: [
    {
      files: ["./*.js"],
      env: {
        node: true,
      },
    },
  ],
};
