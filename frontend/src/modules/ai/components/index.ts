// React-only barrel for the AI plugin. The plugin's root `index.ts`
// keeps framework-agnostic re-exports (types) so the Vue tsconfig
// doesn't have to traverse `.tsx` modules; React consumers import
// from `@/modules/ai/components` instead.
export { AIChatToSQL } from "./AIChatToSQL";
export { AIContextProvider, useAIContext } from "./context";
