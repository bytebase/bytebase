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
      if (pkg.name === "@types/splitpanes") {
        // https://github.com/bytebase/bytebase/security/dependabot/65
        // @types/splitpanes depends on vue@2
        // but it's just a type reference, and will not be actually compiled
        // into our build.
        // So we won't resolve it, to avoid security vulnerability.
        const key = "vue";
        if (pkg.dependencies) {
          Reflect.deleteProperty(pkg.dependencies, key);
        }
        if (pkg.devDependencies) {
          Reflect.deleteProperty(pkg.devDependencies, key);
        }
        if (pkg.peerDependencies) {
          Reflect.deleteProperty(pkg.peerDependencies, key);
        }
        if (pkg.peerDependenciesMeta) {
          Reflect.deleteProperty(pkg.peerDependenciesMeta, key);
        }
      }
      return pkg;
    },
  },
};
