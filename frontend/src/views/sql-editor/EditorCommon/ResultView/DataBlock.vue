<template>
  <div class="space-y-5 m-2">
    <div
      v-for="(row, rowIndex) of rows"
      :key="`rows-${rowIndex + offset}`"
      class="font-mono"
    >
      <p
        class="font-medium text-gray-500 dark:text-gray-300 overflow-hidden whitespace-nowrap"
      >
        ******************************** {{ rowIndex + offset + 1 }}. row
        ********************************
      </p>
      <div class="py-2 px-3 bg-gray-50 dark:bg-gray-700 rounded-sm">
        <div
          v-for="header in table.getFlatHeaders()"
          :key="header.index"
          class="flex items-center text-gray-500 dark:text-gray-300 text-sm"
        >
          <div class="min-w-[7rem] text-left flex items-center font-medium">
            {{ header.column.columnDef.header }}
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
            :
          </div>
          <div class="flex-1">
            <TableCell
              :table="table"
              :value="
                row.getVisibleCells()[header.index].getValue() as RowValue
              "
              :keyword="keyword"
              :set-index="setIndex"
              :row-index="offset + rowIndex"
              :col-index="header.index"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
  <div v-if="rows.length === 0" class="text-center w-full my-12 textinfolabel">
    {{ $t("sql-editor.no-data-available") }}
  </div>
</template>

<script setup lang="ts">
import type { Table } from "@tanstack/vue-table";
import { computed } from "vue";
import {
  FeatureBadge,
  FeatureBadgeForInstanceLicense,
} from "@/components/FeatureGuard";
import { useSubscriptionV1Store } from "@/store";
import type { QueryRow, RowValue } from "@/types/proto/v1/sql_service";
import TableCell from "./DataTable/DataTable/TableCell.vue";
import SensitiveDataIcon from "./DataTable/common/SensitiveDataIcon.vue";
import { useSQLResultViewContext } from "./context";

const props = defineProps<{
  table: Table<QueryRow>;
  setIndex: number;
  offset: number;
  isSensitiveColumn: (index: number) => boolean;
  isColumnMissingSensitive: (index: number) => boolean;
}>();

const { keyword } = useSQLResultViewContext();
const subscriptionStore = useSubscriptionV1Store();

const rows = computed(() => props.table.getRowModel().rows);

const hasSensitiveFeature = computed(() => {
  return subscriptionStore.hasFeature("bb.feature.sensitive-data");
});
</script>
