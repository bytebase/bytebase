// Redefine global to use globalThis for compatibility
declare global {
  var global: typeof globalThis;
}
globalThis.global = globalThis;

export {};
