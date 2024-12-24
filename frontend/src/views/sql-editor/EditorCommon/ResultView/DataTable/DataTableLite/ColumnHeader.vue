<template>
  <div
    class="group relative px-2 h-[32px] min-w-[2rem] whitespace-nowrap text-left bg-gray-50 dark:bg-gray-700 text-xs font-medium text-gray-500 dark:text-gray-300 tracking-wider"
    :class="{
      'cursor-pointer': !selectionDisabled,
    }"
    @click.stop="selectColumn(header.index)"
  >
    <div
      v-if="!selectionDisabled"
      class="absolute inset-0 group-hover:bg-accent/10 pointer-events-none"
      :class="selectionState.column === header.index && 'bg-accent/10'"
    />
    <div class="h-full flex items-center overflow-hidden">
      <span class="flex flex-row items-center select-none">
        <template v-if="String(header.column.columnDef.header).length > 0">
          {{ header.column.columnDef.header }}
        </template>
        <br v-else class="min-h-[1rem] inline-flex" />
      </span>

      <SensitiveDataIcon
        v-if="isSensitiveColumn(header.index)"
        class="ml-0.5 shrink-0"
      />
      <template v-else-if="isColumnMissingSensitive(header.index)">
        <FeatureBadgeForInstanceLicense
          v-if="hasSensitiveFeature"
          :show="true"
          custom-class="ml-0.5 shrink-0"
          feature="bb.feature.sensitive-data"
        />
        <FeatureBadge
          v-else
          feature="bb.feature.sensitive-data"
          custom-class="ml-0.5 shrink-0"
        />
      </template>

      <ColumnSortedIcon
        :is-sorted="header.column.getIsSorted()"
        @click="header.column.getToggleSortingHandler()?.($event)"
      />
    </div>

    <!-- The drag-to-resize handler -->
    <div
      class="group absolute w-[8px] right-0 top-0 bottom-0 cursor-col-resize"
      @pointerdown="$emit('start-resizing')"
      @dblclick="$emit('auto-resize')"
    >
      <div
        class="absolute w-[3px] right-0 top-0 bottom-0 group-hover:bg-accent/30"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Header } from "@tanstack/vue-table";
import FeatureBadge from "@/components/FeatureGuard/FeatureBadge.vue";
import FeatureBadgeForInstanceLicense from "@/components/FeatureGuard/FeatureBadgeForInstanceLicense.vue";
import { featureToRef } from "@/store";
import type { QueryRow } from "@/types/proto/v1/sql_service";
import ColumnSortedIcon from "../common/ColumnSortedIcon.vue";
import SensitiveDataIcon from "../common/SensitiveDataIcon.vue";
import { useSelectionContext } from "../common/selection-logic";

defineProps<{
  header: Header<QueryRow, any>;
  isSensitiveColumn: (index: number) => boolean;
  isColumnMissingSensitive: (index: number) => boolean;
}>();

defineEmits<{
  (event: "start-resizing"): void;
  (event: "auto-resize"): void;
}>();

const {
  state: selectionState,
  disabled: selectionDisabled,
  selectColumn,
} = useSelectionContext();
const hasSensitiveFeature = featureToRef("bb.feature.sensitive-data");
</script>
