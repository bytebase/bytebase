<template>
  <div
    v-show="database"
    ref="popoverRef"
    v-zindexable="{ enabled: true }"
    class="fixed border border-gray-100 rounded bg-white p-2 shadow transition-all text-sm"
    :class="!show && 'pointer-events-none'"
    :style="{
      left: `${x}px`,
      top: `${y}px`,
    }"
  >
    <template v-if="database">
      <div
        class="grid min-w-[14rem] max-w-[18rem] gap-x-2 gap-y-1"
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
            <InstanceV1Name :instance="database.instanceEntity" :link="false" />
          </div>
        </div>
        <div class="contents">
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
              class="text-xs py-px px-1 bg-gray-200/75 rounded-sm"
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
import { onClickOutside, useEventListener } from "@vueuse/core";
import { zindexable as vZindexable } from "vdirs";
import { computed, ref } from "vue";
import {
  EnvironmentV1Name,
  InstanceV1Name,
  ProjectV1Name,
} from "@/components/v2";
import { ComposedDatabase } from "@/types";
import { useHoverStateContext } from "./hover-state";

const props = defineProps<{
  database: ComposedDatabase | undefined;
  x: number;
  y: number;
}>();

const emit = defineEmits<{
  (event: "click-outside", e: MouseEvent): void;
}>();

const { node, update } = useHoverStateContext();

const popoverRef = ref<HTMLDivElement>();
onClickOutside(popoverRef, (e) => {
  emit("click-outside", e);
});

const show = computed(
  () => props.database !== undefined && props.x !== 0 && props.y !== 0
);

useEventListener(popoverRef, "mouseenter", () => {
  // Reset the value immediately to cancel other pending setting values
  update(node.value, "before" /* overrideDelay */);
});
useEventListener(popoverRef, "mouseleave", () => {
  update(undefined, "before");
});
</script>
