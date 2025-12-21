// Global initialization code that must run before any other application code
// This is necessary for compatibility with certain dependencies that expect a global object

// Redefine global to use globalThis for compatibility
(globalThis as typeof globalThis & Record<string, unknown>).global = globalThis;
