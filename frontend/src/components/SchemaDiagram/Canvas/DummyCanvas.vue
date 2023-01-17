<template>
  <teleport to="#capture-container">
    <div
      v-if="isCapturing"
      ref="canvas"
      class="relative overflow-hidden bg-gray-200 pointer-events-none z-10"
      :style="{
        width: `${width}px`,
        height: `${height}px`,
      }"
    >
      <div
        class="absolute overflow-visible"
        :style="{
          width: `${width}px`,
          height: `${height}px`,
        }"
      >
        <DesktopRenderer />
      </div>

      <img
        class="absolute z-50 opacity-50"
        :style="{
          width: `${logoWidth}px`,
          height: 'auto',
          left: `${LOGO_PADDING}px`,
          bottom: `${LOGO_PADDING}px`,
        }"
        src="../../../assets/logo-full.svg"
        alt="Bytebase"
      />
    </div>
  </teleport>
</template>

<script lang="ts" setup>
import { computed, defineComponent, nextTick, PropType, ref, VNode } from "vue";

import {
  calcBBox,
  minmax,
  provideSchemaDiagramContext,
  useSchemaDiagramContext,
} from "../common";

const props = defineProps({
  renderDesktop: {
    type: Function as PropType<() => VNode | VNode[] | undefined>,
    required: true,
  },
});

const LOGO_PADDING = 10;

const canvas = ref<HTMLDivElement>();

const isCapturing = ref(false);

const context = useSchemaDiagramContext();
const { geometries } = context;

const desktopBBox = computed(() => {
  return calcBBox([...geometries.value]);
});

// Automatically set the logo's width according to the canvas size.
const logoWidth = computed(() => {
  const MIN_WIDTH = 120;
  const MAX_WIDTH = 480;

  return minmax(desktopBBox.value.width / 8, MIN_WIDTH, MAX_WIDTH);
});

const width = computed(() => {
  return Math.max(
    desktopBBox.value.width + desktopBBox.value.x * 2,
    logoWidth.value + LOGO_PADDING * 2
  );
});

const height = computed(() => {
  const WIDTH_HEIGHT_RATIO = 4; // an approximate value.
  // Leave a bottom padding to avoid the logo overlapping diagram contents.
  const paddingBottom = logoWidth.value / WIDTH_HEIGHT_RATIO + LOGO_PADDING;
  return desktopBBox.value.height + desktopBBox.value.y * 2 + paddingBottom;
});

const DesktopRenderer = defineComponent({
  render: () => {
    return props.renderDesktop();
  },
});

const capture = async (filename: string, format: "png" | "jpg") => {
  isCapturing.value = true;

  await nextTick();

  const node = canvas.value;
  if (!node) return;

  const [{ toPng }, { default: download }] = await Promise.all([
    import("html-to-image"),
    import("downloadjs"),
  ]);
  const dataUrl = await toPng(node, {});
  download(dataUrl, filename, `image/${format}`);

  isCapturing.value = false;
};

provideSchemaDiagramContext({
  ...context,
  dummy: ref(true),
});

defineExpose({ capture });
</script>
