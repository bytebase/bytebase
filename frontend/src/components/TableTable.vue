<template>
  <BBTable
    :column-list="columnList"
    :data-source="mixedTableList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :row-clickable="true"
    :custom-footer="true"
    @click-row="clickTable"
  >
    <template #body="{ rowData: table }: { rowData: TableMetadata }">
      <BBTableCell v-if="hasSchemaProperty" :left-padding="4" class="w-8">
        {{ schemaName }}
      </BBTableCell>
      <BBTableCell :left-padding="hasSchemaProperty ? 2 : 4" class="w-16">
        <div class="flex items-center space-x-2">
          <EllipsisText>{{ table.name }}</EllipsisText>
          <BBBadge
            v-if="isGhostTable(table)"
            text="gh-ost"
            :can-remove="false"
            class="text-xs whitespace-nowrap"
          />
        </div>
      </BBTableCell>
      <BBTableCell v-if="hasEngineProperty" class="w-8">
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
      <BBTableCell class="w-16 break-all">
        {{ table.userComment }}
      </BBTableCell>
    </template>

    <template v-if="hasReservedTables && !state.showReservedTableList" #footer>
      <tfoot>
        <tr>
          <td :colspan="columnList.length" class="p-0">
            <div
              class="flex items-center justify-center cursor-pointer hover:bg-gray-200 py-2 text-gray-400 text-sm"
              @click="state.showReservedTableList = true"
            >
              {{ $t("database.show-reserved-tables") }}
            </div>
          </td>
        </tr>
      </tfoot>
    </template>
  </BBTable>

  <TableDetailDrawer
    v-if="state.selectedTableName"
    :database-name="database.name"
    :schema-name="schemaName"
    :table-name="state.selectedTableName"
    @dismiss="state.selectedTableName = undefined"
  />
</template>

<script lang="ts" setup>
import { computed, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { ComposedDatabase } from "@/types";
import { TableMetadata } from "@/types/proto/store/database";
import { bytesToString, isGhostTable } from "@/utils";
import EllipsisText from "@/components/EllipsisText.vue";
import { Engine } from "@/types/proto/v1/common";
import { BBTableColumn } from "@/bbkit";
import TableDetailDrawer from "./TableDetailDrawer.vue";

type LocalState = {
  showReservedTableList: boolean;
  selectedTableName?: string;
};

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  schemaName: {
    type: String,
    default: "",
  },
  tableList: {
    required: true,
    type: Object as PropType<TableMetadata[]>,
  },
});

const { t } = useI18n();

const state = reactive<LocalState>({
  showReservedTableList: false,
});

const engine = computed(() => props.database.instanceEntity.engine);

const isPostgres = computed(
  () => engine.value === Engine.POSTGRES || engine.value === Engine.RISINGWAVE
);

const hasSchemaProperty = computed(() => {
  return (
    isPostgres.value ||
    engine.value === Engine.SNOWFLAKE ||
    engine.value === Engine.ORACLE ||
    engine.value === Engine.DM ||
    engine.value === Engine.MSSQL
  );
});

const hasEngineProperty = computed(() => {
  return !isPostgres.value;
});

const columnList = computed(() => {
  const SCHEMA: BBTableColumn = {
    title: t("common.schema"),
  };
  const NAME: BBTableColumn = {
    title: t("common.name"),
  };
  const ENGINE: BBTableColumn = {
    title: t("database.engine"),
  };
  const ROW_COUNT_EST: BBTableColumn = {
    title: t("database.row-count-est"),
  };
  const DATA_SIZE: BBTableColumn = {
    title: t("database.data-size"),
  };
  const INDEX_SIZE: BBTableColumn = {
    title: t("database.index-size"),
  };
  const COMMENT: BBTableColumn = {
    title: t("database.comment"),
  };
  const columns: BBTableColumn[] = [];
  if (hasSchemaProperty.value) {
    columns.push(SCHEMA);
  }
  columns.push(NAME);
  if (hasEngineProperty.value) {
    columns.push(ENGINE);
  }
  columns.push(ROW_COUNT_EST, DATA_SIZE, INDEX_SIZE, COMMENT);

  return columns;
});

const regularTableList = computed(() =>
  props.tableList.filter((table) => !isGhostTable(table))
);
const reservedTableList = computed(() =>
  props.tableList.filter((table) => isGhostTable(table))
);
const hasReservedTables = computed(() => reservedTableList.value.length > 0);

const mixedTableList = computed(() => {
  const tableList = [...regularTableList.value];
  if (state.showReservedTableList) {
    tableList.push(...reservedTableList.value);
  }

  return tableList;
});

const clickTable = (_: number, row: number, e: MouseEvent) => {
  state.selectedTableName = mixedTableList.value[row].name;
};
</script>
