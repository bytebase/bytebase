import { type InjectionKey, inject, provide, type Ref } from "vue";

export type BodyLayoutContext = ReturnType<typeof provideBodyLayoutContext>;

const BODY_LAYOUT_CONTEXT_KEY = Symbol(
  "bb.body-layout.context"
) as InjectionKey<BodyLayoutContext>;

export const provideBodyLayoutContext = (params: {
  mainContainerRef: Ref<HTMLDivElement | undefined>;
}) => {
  const context = {
    ...params,
  };
  provide(BODY_LAYOUT_CONTEXT_KEY, context);

  return context;
};

export const useBodyLayoutContext = () => {
  return inject(BODY_LAYOUT_CONTEXT_KEY)!;
};
