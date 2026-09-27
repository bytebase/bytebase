/// <reference types="vite/client" />

declare module "virtual:stylex:css-only";

declare module "*.yaml" {
  const content: unknown;
  export default content;
}
