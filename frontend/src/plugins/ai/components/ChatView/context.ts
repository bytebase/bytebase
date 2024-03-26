import type { InjectionKey, Ref } from "vue";
import { inject, provide } from "vue";
import type { Mode } from "./types";

export type ChatViewContext = {
  mode: Ref<Mode>;
};

const KEY = Symbol("bb.plugin.ai.chat-view") as InjectionKey<ChatViewContext>;

export const provideChatViewContext = (context: ChatViewContext) => {
  provide(KEY, context);
};

export const useChatViewContext = () => {
  return inject(KEY)!;
};
