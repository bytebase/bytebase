module.exports = {
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
    "no-empty-pattern": "warn",
    "no-useless-escape": "warn",
    "@typescript-eslint/no-non-null-assertion": "off",
    "@typescript-eslint/no-unused-vars": "off",
    "@typescript-eslint/no-explicit-any": "off",
    "@typescript-eslint/no-empty-interface": "warn",
    "@typescript-eslint/no-empty-function": "warn",
    "vue/no-mutating-props": "warn",
    "vue/multi-word-component-names": "warn",
    "vue/no-unused-components": "warn",
    "vue/no-useless-template-attributes": "warn",
  },
  ignorePatterns: ["node_modules", "build", "dist", "public"],
  overrides: [
    {
      files: ["./*.js"],
      env: {
        node: true,
      },
    },
  ],
};
