const path = require("path");

module.exports = {
  "**/*.{js,jsx,ts,tsx,vue}": (filenames) => {
    const cwd = process.cwd();
    const files = filenames
      .map((abs) => path.relative(cwd, abs))
      .map((n) => `"${n}"`)
      .join(" ");
    return [`eslint --fix ${files}`];
  },
};
