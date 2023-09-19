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
import { NButtonGroup, NButton } from "naive-ui";
import { ref, useSlots } from "vue";
import Square2x2 from "~icons/heroicons-outline/squares-2x2";
import { pushNotification } from "@/store";
import { useSchemaDiagramContext } from "../common";
import { ZOOM_RANGE } from "../common/const";
import DummyCanvas from "./DummyCanvas.vue";
import ZoomButton from "./ZoomButton.vue";
import { useDragCanvas, useFitView, useSetCenter } from "./composables";

const slots = useSlots();

const canvas = ref<Element>();
const desktop = ref<Element>();
const dummy = ref<InstanceType<typeof DummyCanvas>>();

const { database, busy, zoom, position } = useSchemaDiagramContext();

const handleFitView = useFitView(canvas);
const { handleZoom } = useDragCanvas(canvas);
useSetCenter(canvas);

const renderDummy = () => {
  return slots["desktop"]?.();
};

const handleScreenshot = async () => {
  busy.value = true;
  try {
    await dummy.value?.capture(`${database.value.databaseName}.png`);
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
