// https://vitejs.dev/guide/features.html#glob-import
const modules = Object.fromEntries(
  Object.entries(import.meta.globEager("./*.ts")).map(
    ([key, value]) => {
      const moduleName = key.replace("./", "").replace(".ts", "");
      return [moduleName, value.default];
    }
  )
);

export default modules;
