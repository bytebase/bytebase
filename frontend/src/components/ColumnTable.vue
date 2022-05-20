<template>
  <BBTable
    :column-list="columnNameList"
    :data-source="columnList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
  >
    <template #body="{ rowData: column }">
      <BBTableCell :left-padding="4" class="w-16">
        {{ column.name }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ column.type }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ column.default }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ column.nullable }}
      </BBTableCell>
      <BBTableCell
        v-if="
          engine != 'POSTGRES' &&
          engine != 'CLICKHOUSE' &&
          engine != 'SNOWFLAKE'
        "
        class="w-8"
      >
        {{ column.characterSet }}
      </BBTableCell>
      <BBTableCell
        v-if="engine != 'CLICKHOUSE' && engine != 'SNOWFLAKE'"
        class="w-8"
      >
        {{ column.collation }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ column.comment }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, defineComponent, PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import { Column, EngineType } from "../types";
import { useI18n } from "vue-i18n";

export default defineComponent({
  name: "ColumnTable",
  components: {},
  props: {
    columnList: {
      required: true,
      type: Object as PropType<Column[]>,
    },
    engine: {
      required: true,
      type: String as PropType<EngineType>,
    },
  },
  setup(props) {
    const { t } = useI18n();
    const NORMAL_COLUMN_LIST: BBTableColumn[] = [
      {
        title: t("common.name"),
      },
      {
        title: t("common.type"),
      },
      {
        title: t("common.Default"),
      },
      {
        title: t("database.nullable"),
      },
      {
        title: t("db.character-set"),
      },
      {
        title: t("db.collation"),
      },
      {
        title: t("database.comment"),
      },
    ];
    const POSTGRES_COLUMN_LIST: BBTableColumn[] = [
      {
        title: t("common.name"),
      },
      {
        title: t("common.type"),
      },
      {
        title: t("common.Default"),
      },
      {
        title: t("database.nullable"),
      },
      {
        title: t("db.collation"),
      },
      {
        title: t("database.comment"),
      },
    ];

    const CLICKHOUSE_SNOWFLAKE_COLUMN_LIST: BBTableColumn[] = [
      {
        title: t("common.name"),
      },
      {
        title: t("common.type"),
      },
      {
        title: t("common.Default"),
      },
      {
        title: t("database.nullable"),
      },
      {
        title: t("database.comment"),
      },
    ];
    const columnNameList = computed(() => {
      switch (props.engine) {
        case "POSTGRES":
          return POSTGRES_COLUMN_LIST;
        case "CLICKHOUSE":
        case "SNOWFLAKE":
          return CLICKHOUSE_SNOWFLAKE_COLUMN_LIST;
        default:
          return NORMAL_COLUMN_LIST;
      }
    });
    return {
      columnNameList,
    };
  },
});
</script>
