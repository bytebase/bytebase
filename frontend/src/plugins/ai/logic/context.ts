import { inject, provide, type InjectionKey } from "vue";
import { AIContext } from "../types";

export const KEY = Symbol("bb.plugin.ai") as InjectionKey<AIContext>;

export const useAIContext = () => {
  return inject(KEY)!;
};

export const provideAIContext = (context: AIContext) => {
  provide(KEY, context);
};
