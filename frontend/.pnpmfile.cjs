module.exports = {
  hooks: {
    readPackage(pkg) {
      if (pkg.name === 'unplugin-icons' && pkg.peerDependencies) {
        ["vue-template-compiler", "vue-template-es2015-compiler"].forEach((key) => {
          Reflect.deleteProperty(pkg.peerDependencies, key);
          Reflect.deleteProperty(pkg.peerDependenciesMeta, key);
        })
      }
      return pkg;
    }
  }
}
