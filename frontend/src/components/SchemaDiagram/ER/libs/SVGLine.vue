<template>
  <svg
    version="1.1"
    xmlns="http://www.w3.org/2000/svg"
    class="absolute"
    :style="{
      left: `${bbox.x - AA_OFFSET[0]}px`,
      top: `${bbox.y - AA_OFFSET[1]}px`,
      width: `${bbox.width + AA_OFFSET[0] * 2}px`,
      height: `${bbox.height + AA_OFFSET[1] * 2}px`,
    }"
  >
    <path
      version="1.1"
      xmlns="http://www.w3.org/2000/svg"
      :d="svgLine"
      v-bind="aaProps"
      pointer-events="visibleStroke"
      fill="none"
      stroke="11.5"
      stroke-width="11.5"
      class="opacity-40"
    ></path>
    <path
      v-for="(decorator, i) in svgDecorators"
      :key="i"
      version="1.1"
      xmlns="http://www.w3.org/2000/svg"
      :d="decorator"
      v-bind="aaProps"
      stroke="rgb(40, 40, 40)"
      fill="none"
      stroke-width="1"
      pointer-events="visibleStroke"
    />
    <path
      version="1.1"
      xmlns="http://www.w3.org/2000/svg"
      :d="svgLine"
      v-bind="aaProps"
      pointer-events="visibleStroke"
      fill="none"
      stroke="rgb(40, 40, 40)"
      stroke-width="1.5"
    ></path>
  </svg>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { Path } from "../../types";
import { curveMonotoneX, line as d3Line } from "d3-shape";
import { calcBBox } from "../../common";

// For SVG anti-aliasing, especially a line aligned to the x-axis
// Just a simple workaround, not perfect.
const AA_OFFSET = [2.25, 2.25];

const props = withDefaults(
  defineProps<{
    path: Path;
    decorators?: Path[];
  }>(),
  {
    decorators: () => [],
  }
);

const bbox = computed(() => {
  const points = [...props.path];
  props.decorators.forEach((decorator) => points.push(...decorator));
  return calcBBox(points);
});

const aaProps = computed(() => ({
  transform: `translate(${AA_OFFSET.join(",")})`,
}));

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
