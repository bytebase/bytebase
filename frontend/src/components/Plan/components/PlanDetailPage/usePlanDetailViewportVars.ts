import { useElementSize } from "@vueuse/core";
import { computed, type Ref } from "vue";
import { useBodyLayoutContext } from "@/layouts/common";

export const PLAN_DETAIL_VIEWPORT_HEIGHT_VAR =
  "--bb-plan-detail-viewport-height";
export const PLAN_DETAIL_LAYOUT_MIN_HEIGHT_VAR =
  "--bb-plan-detail-layout-min-height";
export const PLAN_DETAIL_CONTENT_MIN_HEIGHT_VAR =
  "--bb-plan-detail-content-min-height";

export const usePlanDetailViewportVars = (
  headerRef: Ref<HTMLElement | undefined>
) => {
  const { mainContainerRef } = useBodyLayoutContext();
  const { height: mainContainerHeight } = useElementSize(mainContainerRef);
  const { height: headerHeight } = useElementSize(headerRef);

  const viewportVars = computed(() => {
    if (!mainContainerHeight.value) {
      return undefined;
    }

    const contentMinHeight = Math.max(
      mainContainerHeight.value - headerHeight.value,
      0
    );

    return {
      [PLAN_DETAIL_VIEWPORT_HEIGHT_VAR]: `${mainContainerHeight.value}px`,
      [PLAN_DETAIL_LAYOUT_MIN_HEIGHT_VAR]: `${mainContainerHeight.value}px`,
      [PLAN_DETAIL_CONTENT_MIN_HEIGHT_VAR]: `${contentMinHeight}px`,
    };
  });

  return {
    viewportVars,
  };
};
