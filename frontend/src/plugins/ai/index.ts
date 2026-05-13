// Plugin-wide barrel. Only re-exports framework-agnostic surfaces here
// so the Vue tsconfig (which still picks up this file) doesn't have to
// pull in React `.tsx` modules through imports. React callers reach the
// component entry points via `@/plugins/ai/react` directly.
export * from "./types";
