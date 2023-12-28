import { useElementSize } from "@vueuse/core";
import { InjectionKey, Ref, computed, inject, provide, ref } from "vue";

export type SidebarMode = "NONE" | "DESKTOP" | "MOBILE";

export type SidebarContext = {
  mode: Ref<SidebarMode>;
  desktopSidebarWidth: Ref<number>;
  mobileSidebarOpen: Ref<boolean>;
};

const KEY = Symbol("bb.issue.detail.sidebar") as InjectionKey<SidebarContext>;

export const useIssueSidebarContext = () => {
  return inject(KEY)!;
};

export const provideIssueSidebarContext = (
  containerRef: Ref<HTMLElement | undefined>
) => {
  const { width: containerWidth } = useElementSize(containerRef);
  const MOBILE_SIDEBAR_THRESHOLD = 780;
  const mode = computed((): SidebarMode => {
    const cw = containerWidth.value;
    if (cw === 0) {
      // First render
      return "NONE";
    }
    if (cw < MOBILE_SIDEBAR_THRESHOLD) {
      return "MOBILE";
    }
    return "DESKTOP";
  });
  const desktopSidebarWidth = computed(() => {
    const cw = containerWidth.value;
    if (cw < 1280) return 240;
    return 320;
  });
  const mobileSidebarOpen = ref(false);
  const context: SidebarContext = {
    mode,
    desktopSidebarWidth,
    mobileSidebarOpen,
  };

  provide(KEY, context);
  return context;
};
