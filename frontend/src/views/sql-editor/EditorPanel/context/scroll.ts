import { useMounted } from "@vueuse/core";
import type { Ref } from "vue";
import { computed, watch } from "vue";
import {
  type RichMetadataWithDB,
  type EditorPanelViewState as ViewState,
} from "../types";

export const useScroll = (viewState: Ref<ViewState | undefined>) => {
  const useConsumePendingScrollToTarget = <TTarget, TContext>(
    predicate: (target: RichMetadataWithDB<TTarget>) => boolean,
    context: Ref<TContext>,
    callback: (
      target: RichMetadataWithDB<TTarget>,
      context: TContext
    ) => Promise</* settled */ boolean>
  ) => {
    const target = computed(() => {
      return viewState.value?.pendingScroll;
    });
    const mounted = useMounted();
    const matched = computed(() => {
      const target = viewState.value?.pendingScroll;
      if (!target) return false;
      return predicate(target);
    });
    watch(
      [mounted, matched, context, target],
      ([mounted, matched, context, target]) => {
        if (!mounted) return;
        if (!matched) return;
        if (!context) return;
        if (!target) return;
        callback(target, context).then((settled) => {
          if (settled && viewState.value) {
            viewState.value.pendingScroll = undefined;
          }
        });
      },
      { immediate: true }
    );
  };
  const queuePendingScrollToTarget = <TTarget>(
    target: RichMetadataWithDB<TTarget>
  ) => {
    if (viewState.value) {
      requestAnimationFrame(() => {
        if (viewState.value) {
          viewState.value.pendingScroll = target;
        }
      });
    }
  };

  return {
    queuePendingScrollToTarget,
    useConsumePendingScrollToTarget,
  };
};
