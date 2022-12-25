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
    <template #body="{ rowData: table }">
      <BBTableCell :left-padding="4">
        <div class="flex items-center space-x-2">
          <EllipsisText>{{ getTableName(table.name) }}</EllipsisText>
          <BBBadge
            v-if="isGhostTable(table)"
            text="gh-ost"
            :can-remove="false"
            class="text-xs whitespace-nowrap"
          />
        </div>
      </BBTableCell>
      <BBTableCell v-if="!isPostgres" class="w-[14%]">
        {{ table.engine }}
      </BBTableCell>
      <BBTableCell class="w-[14%]">
        {{ table.rowCount }}
      </BBTableCell>
      <BBTableCell class="w-[14%]">
        {{ bytesToString(table.dataSize) }}
      </BBTableCell>
      <BBTableCell class="w-[14%]">
        {{ bytesToString(table.indexSize) }}
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
</template>

<script lang="ts">
import { computed, defineComponent, PropType, reactive } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { Database } from "@/types";
import { TableMetadata } from "@/types/proto/store/database";
import { bytesToString, databaseSlug, isGhostTable } from "@/utils";
import EllipsisText from "@/components/EllipsisText.vue";

type LocalState = {
  showReservedTableList: boolean;
};

export default defineComponent({
  name: "TableTable",
  components: { EllipsisText },
  props: {
    database: {
      required: true,
      type: Object as PropType<Database>,
    },
    schemaName: {
      type: String,
      default: "",
    },
    tableList: {
      required: true,
      type: Object as PropType<TableMetadata[]>,
    },
  },
  setup(props) {
    const router = useRouter();
    const { t } = useI18n();

    const state = reactive<LocalState>({
      showReservedTableList: false,
    });

    const isPostgres = props.database.instance.engine === "POSTGRES";

    const hasSchemaProperty =
      isPostgres || props.database.instance.engine === "SNOWFLAKE";

    const getTableName = (tableName: string) => {
      if (hasSchemaProperty) {
        return `"${props.schemaName}"."${tableName}"`;
      }
      return tableName;
    };

    const columnList = computed(() => {
      if (isPostgres) {
        return [
          {
            title: t("common.name"),
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
        ];
      } else {
        return [
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
        ];
      }
    });

    const regularTableList = computed(() =>
      props.tableList.filter((table) => !isGhostTable(table))
    );
    const reservedTableList = computed(() =>
      props.tableList.filter((table) => isGhostTable(table))
    );
    const hasReservedTables = computed(
      () => reservedTableList.value.length > 0
    );

    const mixedTableList = computed(() => {
      const tableList = [...regularTableList.value];
      if (state.showReservedTableList) {
        tableList.push(...reservedTableList.value);
      }

      return tableList;
    });

    const clickTable = (_: number, row: number, e: MouseEvent) => {
      const table = mixedTableList.value[row];
      let url = `/db/${databaseSlug(props.database)}/table/${table.name}`;
      if (props.schemaName !== "") {
        url = url + `?schema=${props.schemaName}`;
      }
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    };

    return {
      columnList,
      state,
      isPostgres,
      bytesToString,
      clickTable,
      hasReservedTables,
      mixedTableList,
      isGhostTable,
      getTableName,
    };
  },
});
</script>
