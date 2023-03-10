import { inject, InjectionKey, provide, Ref } from "vue";
import { Mode } from "./types";

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
