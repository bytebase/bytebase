<template>
  <div ref="canvas" class="w-full h-full relative bg-gray-200 overflow-hidden">
    <div
      ref="desktop"
      class="absolute overflow-visible"
      :style="{
        transformOrigin: '0 0 0',
        transform: `matrix(${zoom}, 0, 0, ${zoom}, ${position.x}, ${position.y})`,
      }"
    >
      <slot name="desktop" />
    </div>

    <div class="!absolute right-2 bottom-2 flex items-center gap-x-2">
      <slot name="controls" />

      <ZoomButton
        :min="ZOOM_RANGE.min"
        :max="ZOOM_RANGE.max"
        @zoom-in="handleZoom(0.1)"
        @zoom-out="handleZoom(-0.1)"
      />
    </div>

    <slot />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { useEventListener } from "@vueuse/core";
import normalizeWheel from "normalize-wheel";

import { Position } from "../types";
import { useDraggable, minmax, useSchemaDiagramContext } from "../common";
import ZoomButton from "./ZoomButton.vue";

const canvas = ref<Element>();
const desktop = ref<Element>();

const { zoom, position } = useSchemaDiagramContext();

const zoomCenter = ref<Position>({ x: 0.5, y: 0.5 });
const ZOOM_RANGE = {
  max: 2, // 200%
  min: 0.05, // 5%
};

const handleZoom = (delta: number, center: Position = { x: 0.5, y: 0.5 }) => {
  if (!canvas.value) return "";

  const z = minmax(zoom.value + delta, ZOOM_RANGE.min, ZOOM_RANGE.max);
  const factor = z / zoom.value;
  zoom.value = z;

  zoomCenter.value = center;

  // While zooming, we need to adjust the `position` according to the
  // zoom level and zoom center.
  const { width, height } = canvas.value.getBoundingClientRect();
  const cx = center.x * width;
  const cy = center.y * height;
  position.value.x = cx - factor * (cx - position.value.x);
  position.value.y = cy - factor * (cy - position.value.y);
};
const handlePan = (x: number, y: number) => {
  position.value.x += x;
  position.value.y += y;
};

useDraggable(canvas, {
  exact: false,
  onPan: handlePan,
});

useEventListener(
  canvas,
  "mousewheel",
  (e: WheelEvent) => {
    const ne = normalizeWheel(e);

    const scrollDelta = -ne.pixelY;
    const delta = scrollDelta / 250; // adjust the scrolling speed
    const rect = canvas.value!.getBoundingClientRect();
    const center = {
      x: (e.pageX - rect.left) / rect.width,
      y: (e.pageY - rect.top) / rect.height,
    };
    handleZoom(delta, center);

    e.preventDefault();
    e.stopPropagation();
  },
  true
);
</script>
