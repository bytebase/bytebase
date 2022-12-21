import { ref, unref } from "vue";
import { MaybeRef, useEventListener } from "@vueuse/core";

import { Position } from "../types";

export type DraggableOptions = {
  exact?: boolean;
  onPan?: (dx: number, dy: number) => void;
  capture?: boolean;
};

export const useDraggable = (
  target: MaybeRef<Element | undefined>,
  options: DraggableOptions
) => {
  const startPointerPosition = ref<Position>();
  const lastPointerPosition = ref<Position>();

  const start = (e: PointerEvent) => {
    if (unref(options.exact) && e.target !== unref(target)) return;
    startPointerPosition.value = {
      x: e.screenX,
      y: e.screenY,
    };
    lastPointerPosition.value = {
      x: e.screenX,
      y: e.screenY,
    };
    e.stopPropagation();
    e.preventDefault();
  };
  const move = (e: PointerEvent) => {
    if (!startPointerPosition.value) return;
    if (!lastPointerPosition.value) return;
    const pointer = { x: e.screenX, y: e.screenY };
    e.stopPropagation();
    e.preventDefault();
    options.onPan?.(
      pointer.x - lastPointerPosition.value.x,
      pointer.y - lastPointerPosition.value.y
    );
    lastPointerPosition.value = pointer;
  };
  const end = (e: PointerEvent) => {
    startPointerPosition.value = undefined;
    e.stopPropagation();
    e.preventDefault();
  };

  const capture = options.capture ?? false;
  useEventListener(target, "pointerdown", start, capture);
  useEventListener(window, "pointermove", move, capture);
  useEventListener(window, "pointerup", end, capture);
};
