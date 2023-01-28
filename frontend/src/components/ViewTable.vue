<template>
  <BBTable
    :column-list="columnList"
    :data-source="viewList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :row-clickable="false"
  >
    <template #body="{ rowData: view }">
      <BBTableCell :left-padding="4" class="w-16">
        {{ getViewName(view.name) }}
      </BBTableCell>
      <BBTableCell class="w-64">
        {{ view.definition }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ view.comment }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { ViewMetadata } from "@/types/proto/store/database";
import { Database } from "@/types";

export default {
  name: "ViewTable",
  components: {},
  props: {
    database: {
      required: true,
      type: Object as PropType<Database>,
    },
    schemaName: {
      type: String,
      default: "",
    },
    viewList: {
      required: true,
      type: Object as PropType<ViewMetadata[]>,
    },
  },
  setup(props) {
    const { t } = useI18n();

    const hasSchemaProperty =
      props.database.instance.engine === "POSTGRES" ||
      props.database.instance.engine === "SNOWFLAKE";

    const columnList = computed(() => [
      {
        title: t("common.name"),
      },
      {
        title: t("common.definition"),
      },
      {
        title: t("database.comment"),
      },
    ]);

    const getViewName = (viewName: string) => {
      if (hasSchemaProperty) {
        return `"${props.schemaName}"."${viewName}"`;
      }
      return viewName;
    };

    return {
      columnList,
      hasSchemaProperty,
      getViewName,
    };
  },
};
</script>
