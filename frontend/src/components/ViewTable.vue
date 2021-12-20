<template>
  <BBTable
    :column-list="COLUMN_LIST"
    :data-source="viewList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :row-clickable="false"
  >
    <template #body="{ rowData: view }">
      <BBTableCell :left-padding="4" class="w-16">
        {{ view.name }}
      </BBTableCell>
      <BBTableCell class="w-64">
        {{ view.definition }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ view.comment }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ humanizeTs(view.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import { View } from "../types";
import { useI18n } from "vue-i18n";

export default {
  name: "ViewTable",
  components: {},
  props: {
    viewList: {
      required: true,
      type: Object as PropType<View[]>,
    },
  },
  setup() {
    const { t } = useI18n();
    const COLUMN_LIST: BBTableColumn[] = [
      {
        title: t("common.name"),
      },
      {
        title: t("common.definition"),
      },
      {
        title: t("database.comment"),
      },
      {
        title: t("common.created-at"),
      },
    ];
    return {
      COLUMN_LIST,
    };
  },
};
</script>
