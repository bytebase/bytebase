/// <reference types="vite/client" />

// Allow .yaml imports used by @/types
declare module "*.yaml" {
  const content: unknown;
  export default content;
}
