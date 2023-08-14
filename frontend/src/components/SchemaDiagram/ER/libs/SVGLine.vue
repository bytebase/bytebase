<template>
  <svg
    version="1.1"
    xmlns="http://www.w3.org/2000/svg"
    class="absolute cursor-pointer"
    pointer-events="none"
    :viewBox="[viewBox.x, viewBox.y, viewBox.width, viewBox.height].join(' ')"
    :style="{
      left: `${bbox.x + viewBox.x}px`,
      top: `${bbox.y + viewBox.y}px`,
      width: `${viewBox.width}px`,
      height: `${viewBox.height}px`,
      'z-index': hover ? 1 : 0,
    }"
  >
    <path
      ref="track"
      version="1.1"
      xmlns="http://www.w3.org/2000/svg"
      :d="svgLine"
      pointer-events="visibleStroke"
      fill="none"
      :stroke="hover ? COLORS.GLOW.HOVER : COLORS.GLOW.NORMAL"
      :stroke-width="GLOW_WIDTH"
    />
    <path
      v-for="(decorator, i) in svgDecorators"
      :key="i"
      version="1.1"
      xmlns="http://www.w3.org/2000/svg"
      :d="decorator"
      fill="none"
      :stroke="hover ? COLORS.LINE.HOVER : COLORS.LINE.NORMAL"
      stroke-width="1"
      pointer-events="none"
    />
    <path
      ref="line"
      version="1.1"
      xmlns="http://www.w3.org/2000/svg"
      :d="svgLine"
      pointer-events="visibleStroke"
      fill="none"
      :stroke="hover ? COLORS.LINE.HOVER : COLORS.LINE.NORMAL"
      :stroke-width="hover ? 2 : 1.5"
    />
  </svg>
</template>

<script lang="ts" setup>
import { useElementHover } from "@vueuse/core";
import { curveMonotoneX, line as d3Line } from "d3-shape";
import { computed, ref } from "vue";
import { calcBBox } from "../../common";
import { Path, Rect } from "../../types";

const GLOW_WIDTH = 12;
const PADDING = GLOW_WIDTH / 2;
const COLORS = {
  GLOW: {
    NORMAL: "transparent",
    HOVER: "rgba(55,48,163,0.1)",
  },
  LINE: {
    NORMAL: "#1f2937",
    HOVER: "#4f46e5",
  },
};

const props = withDefaults(
  defineProps<{
    path: Path;
    decorators?: Path[];
  }>(),
  {
    decorators: () => [],
  }
);

const track = ref<SVGPathElement>();
const line = ref<SVGPathElement>();

const trackHover = useElementHover(track);
const lineHover = useElementHover(line);
const hover = computed(() => trackHover.value || lineHover.value);

const bbox = computed(() => {
  const points = [...props.path];
  props.decorators.forEach((decorator) => points.push(...decorator));
  return calcBBox(points);
});

const viewBox = computed((): Rect => {
  return {
    x: -PADDING,
    y: -PADDING,
    width: Math.max(bbox.value.width, 0) + PADDING * 2,
    height: Math.max(bbox.value.height, 0) + PADDING * 2,
  };
});

const normalize = (x: number, y: number): [number, number] => {
  const dx = bbox.value.x;
  const dy = bbox.value.y;
  return [x - dx, y - dy];
};

const svgLine = computed(() => {
  return (
    d3Line().curve(curveMonotoneX)(
      props.path.map((p) => normalize(p.x, p.y))
    ) ?? ""
  );
});

const svgDecorators = computed(() => {
  return props.decorators.map((path) => {
    return d3Line()(path.map((p) => normalize(p.x, p.y))) ?? "";
  });
});
</script>
