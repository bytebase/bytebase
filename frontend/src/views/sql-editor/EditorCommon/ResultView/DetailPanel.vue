<template>
  <DrawerContent :title="$t('common.detail')">
    <!-- eslint-disable vue/no-v-html -->
    <div
      class="w-[100vw-8rem] min-w-[20rem] md:max-w-[40rem] overflow-auto whitespace-pre-wrap text-sm"
      :class="dark ? 'text-white' : 'text-main'"
      v-html="html"
    ></div>
  </DrawerContent>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { escape, get } from "lodash-es";

import { DrawerContent } from "@/components/v2";
import { useSQLResultViewContext } from "./context";
import { SQLResultSetV1 } from "@/types";
import { extractSQLRowValue } from "@/utils";
import { RowValue } from "@/types/proto/v1/sql_service";

const props = defineProps<{
  resultSet?: SQLResultSetV1;
}>();

const { dark, detail } = useSQLResultViewContext();

const value = computed(() => {
  const { resultSet } = props;
  const { set, row, col } = detail.value;
  const cell: RowValue =
    get(resultSet, `results.${set}.rows.${row}.values.${col}`) ??
    RowValue.fromJSON({});
  return extractSQLRowValue(cell);
});

const html = computed(() => {
  const str = String(value.value);
  if (str.length === 0) {
    return `<br style="min-width: 1rem; display: inline-flex;" />`;
  }

  return escape(str);
});
</script>
