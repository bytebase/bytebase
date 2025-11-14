import { type InjectionKey, inject, provide } from "vue";
import type { AIContext } from "../types";

export const KEY = Symbol("bb.plugin.ai") as InjectionKey<AIContext>;

export const useAIContext = () => {
  return inject(KEY)!;
};

export const provideAIContext = (context: AIContext) => {
  provide(KEY, context);
};
