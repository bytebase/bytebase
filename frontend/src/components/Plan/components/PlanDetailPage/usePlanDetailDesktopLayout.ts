import type { CSSProperties, Ref } from "vue";
import { computed } from "vue";
import { PLAN_DETAIL_CONTENT_MIN_HEIGHT_VAR } from "./usePlanDetailViewportVars";

export const PLAN_DETAIL_XL_BREAKPOINT = 1280;
export const PLAN_DETAIL_TASK_DRAWER_WIDTH = 640;
export const PLAN_DETAIL_METADATA_DRAWER_WIDTH = 320;

export const usePlanDetailDesktopLayout = ({
  sidebarMode,
  containerWidth,
  desktopSidebarWidth,
  hasDetailPanel,
}: {
  sidebarMode: Readonly<Ref<string>>;
  containerWidth: Readonly<Ref<number>>;
  desktopSidebarWidth: Readonly<Ref<number>>;
  hasDetailPanel: Readonly<Ref<boolean>>;
}) => {
  const supportsInlineDetailPanel = computed(() => {
    return (
      sidebarMode.value === "DESKTOP" &&
      containerWidth.value >= PLAN_DETAIL_XL_BREAKPOINT
    );
  });

  const showDesktopSidebar = computed(() => {
    return sidebarMode.value === "DESKTOP" && !hasDetailPanel.value;
  });

  const showInlineDetailPanel = computed(() => {
    return hasDetailPanel.value && supportsInlineDetailPanel.value;
  });

  const showTaskDrawer = computed(() => {
    return (
      hasDetailPanel.value &&
      !showInlineDetailPanel.value &&
      sidebarMode.value !== "NONE"
    );
  });

  const showSidebarDrawer = computed(() => {
    return sidebarMode.value === "MOBILE";
  });

  const desktopLayoutStyle = computed<CSSProperties>(() => {
    const baseStyle: CSSProperties = {
      minHeight: `var(${PLAN_DETAIL_CONTENT_MIN_HEIGHT_VAR})`,
    };

    if (sidebarMode.value !== "DESKTOP") {
      return baseStyle;
    }

    if (showInlineDetailPanel.value) {
      return {
        ...baseStyle,
        gridTemplateColumns: "minmax(0, 1fr) minmax(0, 50%)",
      };
    }

    if (showDesktopSidebar.value) {
      return {
        ...baseStyle,
        gridTemplateColumns: `minmax(0, 1fr) ${desktopSidebarWidth.value}px`,
      };
    }

    return baseStyle;
  });

  return {
    desktopLayoutStyle,
    showDesktopSidebar,
    showInlineDetailPanel,
    showTaskDrawer,
    showSidebarDrawer,
  };
};
