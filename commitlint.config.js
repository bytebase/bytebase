module.exports = {
    // need to use relative path case our node project isn't in the same folder level as git
    // similar problem: https://github.com/conventional-changelog/commitlint/issues/613
    extends: ['./frontend/node_modules/@commitlint/config-conventional']
};
