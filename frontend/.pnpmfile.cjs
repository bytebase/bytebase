module.exports = {
  hooks: {
    readPackage(pkg) {
      if (pkg.name === "unplugin-icons") {
        // https://github.com/bytebase/bytebase/security/dependabot/63
        ["vue-template-compiler", "vue-template-es2015-compiler"].forEach(
          (key) => {
            if (pkg.peerDependencies) {
              Reflect.deleteProperty(pkg.peerDependencies, key);
            }
            if (pkg.peerDependenciesMeta) {
              Reflect.deleteProperty(pkg.peerDependenciesMeta, key);
            }
          }
        );
      }
      return pkg;
    },
  },
};
