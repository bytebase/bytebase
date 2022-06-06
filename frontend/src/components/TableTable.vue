<template>
  <BBTable
    :column-list="COLUMN_LIST"
    :data-source="tableList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :row-clickable="true"
    @click-row="clickTable"
  >
    <template #body="{ rowData: table }">
      <BBTableCell :left-padding="4" class="w-16">
        {{ table.name }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ table.engine }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ table.rowCount }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ bytesToString(table.dataSize) }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ bytesToString(table.indexSize) }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ humanizeTs(table.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import { Table } from "../types";
import { bytesToString, databaseSlug } from "../utils";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";

export default defineComponent({
  name: "TableTable",
  props: {
    tableList: {
      required: true,
      type: Object as PropType<Table[]>,
    },
  },
  setup(props) {
    const router = useRouter();
    const { t } = useI18n();

    const COLUMN_LIST: BBTableColumn[] = [
      {
        title: t("common.name"),
      },
      {
        title: t("database.engine"),
      },
      {
        title: t("database.row-count-est"),
      },
      {
        title: t("database.data-size"),
      },
      {
        title: t("database.index-size"),
      },
      {
        title: t("common.created-at"),
      },
    ];

    const clickTable = (section: number, row: number) => {
      const table = props.tableList[row];
      router.push(`/db/${databaseSlug(table.database)}/table/${table.name}`);
    };

    return {
      COLUMN_LIST,
      bytesToString,
      clickTable,
    };
  },
});
</script>
