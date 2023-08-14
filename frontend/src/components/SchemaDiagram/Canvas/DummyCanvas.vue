<template>
  <teleport to="#capture-container">
    <div
      v-if="isCapturing"
      ref="canvas"
      class="relative overflow-hidden bg-gray-200 pointer-events-none z-10"
      :style="{
        width: `${resizeParams.rect.width}px`,
        height: `${resizeParams.rect.height}px`,
      }"
    >
      <div
        class="absolute overflow-visible origin-top-left"
        :style="{
          width: `${width}px`,
          height: `${height}px`,
          transform: `scale(${resizeParams.zoom})`,
        }"
      >
        <Watermark />
        <DesktopRenderer />
      </div>

      <img
        class="absolute z-50 opacity-50 origin-bottom-left"
        :style="{
          width: `${logoWidth}px`,
          height: 'auto',
          left: `${LOGO_PADDING * resizeParams.zoom}px`,
          bottom: `${LOGO_PADDING * resizeParams.zoom}px`,
          transform: `scale(${resizeParams.zoom})`,
        }"
        src="../../../assets/logo-full.svg"
        alt="Bytebase"
      />

      <div
        class="absolute z-50 opacity-50 origin-bottom-right"
        :style="{
          lineHeight: '1em',
          fontSize: `${logoWidth / 6}px`,
          right: `${LOGO_PADDING * resizeParams.zoom}px`,
          bottom: `${LOGO_PADDING * resizeParams.zoom}px`,
          transform: `scale(${resizeParams.zoom})`,
        }"
      >
        {{ now }}
      </div>
    </div>
  </teleport>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { computed, defineComponent, nextTick, PropType, ref, VNode } from "vue";
import Watermark from "@/components/misc/Watermark.vue";
import { pushNotification } from "@/store";
import { minmax } from "@/utils";
import {
  calcBBox,
  fitBBox,
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

const resizeParams = computed(() => {
  // Fit the output image within a size-limited box
  // and keep W/H ratio.
  return fitBBox(
    { width: width.value, height: height.value },
    {
      width: 4096,
      height: 4096,
    },
    [0, 1]
  );
});

const now = computed((): string => {
  return dayjs().format("YYYY-MM-DD HH:mm");
});

const DesktopRenderer = defineComponent({
  render: () => {
    return props.renderDesktop();
  },
});

const capture = async (filename: string) => {
  if (isCapturing.value) {
    return;
  }

  isCapturing.value = true;

  await nextTick();

  const node = canvas.value;
  if (!node) {
    return;
  }

  try {
    const [{ toBlob }, { default: download }] = await Promise.all([
      import("html-to-image"),
      import("downloadjs"),
    ]);
    const blob = await toBlob(node, {
      pixelRatio: 1,
      quality: 0.9,
    });
    if (blob) {
      download(blob, filename, blob.type);

      const data = [new window.ClipboardItem({ [blob.type]: blob })];
      await navigator.clipboard.write(data);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: "Screenshot generated successfully and copied to the clipboard!",
      });
    }
  } finally {
    isCapturing.value = false;
  }
};

provideSchemaDiagramContext({
  ...context,
  dummy: ref(true),
});

defineExpose({ capture });
</script>
