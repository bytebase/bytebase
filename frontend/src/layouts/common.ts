import { type InjectionKey, inject, provide, type Ref } from "vue";
import { useClassStack } from "@/utils";

export type BodyLayoutContext = ReturnType<typeof provideBodyLayoutContext>;

const BODY_LAYOUT_CONTEXT_KEY = Symbol(
  "bb.body-layout.context"
) as InjectionKey<BodyLayoutContext>;

export const provideBodyLayoutContext = (params: {
  mainContainerRef: Ref<HTMLDivElement | undefined>;
}) => {
  const {
    classes: mainContainerClasses,
    override: overrideMainContainerClass,
  } = useClassStack();

  const context = {
    ...params,
    overrideMainContainerClass,
    mainContainerClasses,
  };
  provide(BODY_LAYOUT_CONTEXT_KEY, context);

  return context;
};

export const useBodyLayoutContext = () => {
  return inject(BODY_LAYOUT_CONTEXT_KEY)!;
};
