module.exports = {
  useTabs: false,
  tabWidth: 2,
  singleQuote: false,
  trailingComma: "es5",
  printWidth: 80,
  plugins: [require.resolve("@trivago/prettier-plugin-sort-imports")],
  importOrder: [
    "<BUILTIN_MODULES>",
    "<THIRD_PARTY_MODULES>",
    "^@/(.+)",
    "^[./]",
  ],
};
