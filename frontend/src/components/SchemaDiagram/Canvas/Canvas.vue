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

      <NButtonGroup size="tiny" class="bg-white rounded">
        <NTooltip>
          <template #trigger>
            <NButton tooltip>
              <heroicons-outline:photo @click="handleScreenshot" />
            </NButton>
          </template>
          <div class="whitespace-nowrap">Screenshot</div>
        </NTooltip>
      </NButtonGroup>

      <NButtonGroup size="tiny" class="bg-white rounded">
        <NTooltip>
          <template #trigger>
            <NButton tooltip>
              <Square2x2 @click="handleFitView" />
            </NButton>
          </template>
          <div class="whitespace-nowrap">
            {{ $t("schema-diagram.fit-content-with-view") }}
          </div>
        </NTooltip>
      </NButtonGroup>

      <ZoomButton
        :min="ZOOM_RANGE.min"
        :max="ZOOM_RANGE.max"
        @zoom-in="handleZoom(0.1)"
        @zoom-out="handleZoom(-0.1)"
      />
    </div>

    <DummyCanvas ref="dummy" :render-desktop="renderDummy" />

    <slot />
  </div>
</template>

<script lang="ts" setup>
import { ref, useSlots } from "vue";
import { useEventListener } from "@vueuse/core";
import normalizeWheel from "normalize-wheel";
import { NButtonGroup, NButton } from "naive-ui";
import Square2x2 from "~icons/heroicons-outline/squares-2x2";

import { Point } from "../types";
import { useDraggable, minmax, useSchemaDiagramContext } from "../common";
import ZoomButton from "./ZoomButton.vue";
import { fitView } from "./libs/fitView";
import DummyCanvas from "./DummyCanvas.vue";
import { pushNotification } from "@/store";

const slots = useSlots();

const canvas = ref<Element>();
const desktop = ref<Element>();
const dummy = ref<InstanceType<typeof DummyCanvas>>();

const { database, busy, zoom, position, panning, geometries, events } =
  useSchemaDiagramContext();

const zoomCenter = ref<Point>({ x: 0.5, y: 0.5 });
const ZOOM_RANGE = {
  max: 2, // 200%
  min: 0.05, // 5%
};

const handleFitView = () => {
  if (!canvas.value) return;
  const layout = fitView(
    canvas.value,
    [...geometries.value],
    [10, 20, 40, 20] /* paddings [T,R,B,L] */,
    [ZOOM_RANGE.min, 1.0] /* [zoomMin, zoomMax] */
  );
  zoom.value = layout.zoom;
  position.value = {
    x: layout.rect.x,
    y: layout.rect.y,
  };
};
events.on("fit-view", handleFitView);

const handleZoom = (delta: number, center: Point = { x: 0.5, y: 0.5 }) => {
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
  panning.value = true;
};
const handlePanEnd = () => {
  requestAnimationFrame(() => {
    panning.value = false;
  });
};

useDraggable(canvas, {
  exact: false,
  onPan: handlePan,
  onEnd: handlePanEnd,
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

const renderDummy = () => {
  return slots["desktop"]?.();
};

const handleScreenshot = async () => {
  busy.value = true;
  try {
    await dummy.value?.capture(`${database.value.name}.png`, "png");
  } catch {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Screenshot request failed",
    });
  } finally {
    busy.value = false;
  }
};
</script>
