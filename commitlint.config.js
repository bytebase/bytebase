module.exports = {
    extends: ['@commitlint/config-conventional'],
    rules: {
        "body-max-line-length": [0, "always"],
        "body-max-length": [0, "always"],
        "header-max-length": [0, "always"]
    }
};
