<template>
  <div
    v-show="show && database"
    ref="popoverRef"
    v-zindexable="{ enabled: true }"
    class="fixed border border-gray-100 rounded-sm bg-white p-2 shadow-sm transition-all text-sm"
    :class="!show && 'pointer-events-none'"
    :style="{
      left: `${displayPosition.x}px`,
      top: `${displayPosition.y}px`,
    }"
  >
    <template v-if="database">
      <div
        class="grid min-w-56 max-w-[18rem] gap-x-2 gap-y-1"
        style="grid-template-columns: auto 1fr"
      >
        <div class="contents">
          <div class="text-gray-500 font-medium">
            {{ $t("common.environment") }}
          </div>
          <div class="text-main text-right">
            <EnvironmentV1Name
              :environment="database.effectiveEnvironmentEntity"
              :link="false"
            />
          </div>
        </div>
        <div class="contents">
          <div class="text-gray-500 font-medium">
            {{ $t("common.instance") }}
          </div>
          <div class="text-main text-right">
            <InstanceV1Name
              :instance="database.instanceResource"
              :link="false"
            />
          </div>
        </div>
        <div v-if="!hasProjectContext" class="contents">
          <div class="text-gray-500 font-medium">
            {{ $t("common.project") }}
          </div>
          <div class="text-main text-right">
            <ProjectV1Name :project="database.projectEntity" :link="false" />
          </div>
        </div>
        <div class="contents">
          <div class="text-gray-500 font-medium">{{ $t("common.labels") }}</div>
          <div class="text-main flex flex-row justify-end flex-wrap gap-1">
            <div
              v-for="(value, key) in database.labels"
              :key="key"
              class="text-xs py-px px-1 bg-gray-200/75 rounded-xs"
            >
              <span>{{ key }}</span>
              <template v-if="value">
                <span>:</span>
                <span>{{ value }}</span>
              </template>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onClickOutside, useElementSize, useEventListener } from "@vueuse/core";
import { zindexable as vZindexable } from "vdirs";
import { computed, ref } from "vue";
import {
  EnvironmentV1Name,
  InstanceV1Name,
  ProjectV1Name,
} from "@/components/v2";
import { useSQLEditorStore } from "@/store";
import type { Position, SQLEditorTreeNodeTarget } from "@/types";
import { minmax } from "@/utils";
import { useHoverStateContext } from "./hover-state";

const props = defineProps<{
  offsetX: number;
  offsetY: number;
  margin: number;
}>();

const emit = defineEmits<{
  (event: "click-outside", e: MouseEvent): void;
}>();

const editorStore = useSQLEditorStore();
const { state, position, update } = useHoverStateContext();

const popoverRef = ref<HTMLDivElement>();
onClickOutside(popoverRef, (e) => {
  emit("click-outside", e);
});
const { height: popoverHeight } = useElementSize(popoverRef, undefined, {
  box: "border-box",
});

const show = computed(
  () =>
    state.value !== undefined &&
    position.value.x !== 0 &&
    position.value.y !== 0
);
const database = computed(() => {
  if (!state.value?.node) return undefined;
  const { type, target } = state.value.node.meta;
  if (type !== "database") return undefined;
  return target as SQLEditorTreeNodeTarget<"database">;
});

const hasProjectContext = computed(() => {
  return !!editorStore.project;
});

const displayPosition = computed(() => {
  const p: Position = {
    x: position.value.x + props.offsetX,
    y: position.value.y + props.offsetY,
  };
  const yMin = props.margin;
  const yMax = window.innerHeight - popoverHeight.value - props.margin;
  p.y = minmax(p.y, yMin, yMax);

  return p;
});

useEventListener(popoverRef, "mouseenter", () => {
  // Keep the hover panel visible with a small delay to prevent flicker
  update(state.value, "before", 50);
});
useEventListener(popoverRef, "mouseleave", () => {
  update(undefined, "after");
});
</script>
