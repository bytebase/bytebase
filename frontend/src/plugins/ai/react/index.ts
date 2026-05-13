// React-only barrel for the AI plugin. The plugin's root `index.ts`
// keeps framework-agnostic re-exports (types) so the Vue tsconfig
// doesn't have to traverse `.tsx` modules; React consumers import
// from `@/plugins/ai/react` instead.
export { AIChatToSQL } from "./AIChatToSQL";
export { AIContextProvider, useAIContext } from "./context";
