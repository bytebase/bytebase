/// <reference types="vite/client" />

// Allow tsconfig.react.json to resolve .vue imports transitively
// (React pages import from @/store, @/router which re-export .vue modules)
declare module "*.vue" {
  const component: unknown;
  export default component;
}

// Allow .yaml imports used by @/types
declare module "*.yaml" {
  const content: unknown;
  export default content;
}
