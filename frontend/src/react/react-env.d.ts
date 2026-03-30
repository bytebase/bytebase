/// <reference types="vite/client" />

// Allow tsconfig.react.json to resolve .vue imports transitively
// (React pages import from @/store, @/router which re-export .vue modules)
declare module "*.vue" {
  import type { DefineComponent } from "vue";
  const component: DefineComponent<
    Record<string, unknown>,
    Record<string, unknown>,
    // biome-ignore lint/suspicious/noExplicitAny: Vue module shim
    any // eslint-disable-line @typescript-eslint/no-explicit-any
  >;
  export default component;
}

// Allow .yaml imports used by @/types
declare module "*.yaml" {
  const content: unknown;
  export default content;
}
