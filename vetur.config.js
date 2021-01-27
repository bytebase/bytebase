module.exports = {
  projects: [
    {
      // **required**
      // Where is your project?
      // It is relative to `vetur.config.js`.
      root: "./frontend",
      // **optional** default: `'package.json'`
      // Where is `package.json` in the project?
      // We use it to determine the version of vue.
      // It is relative to root property.
      package: "./package.json",
      // **optional**
      // Where is TypeScript config file in the project?
      // It is relative to root property.
      tsconfig: "./tsconfig.json",
      // **optional** default: `'./.vscode/vetur/snippets'`
      // Where is vetur custom snippets folders?
      snippetFolder: "./.vscode/vetur/snippets",
    },
  ],
};
