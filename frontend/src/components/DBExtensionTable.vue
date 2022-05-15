<template>
  <BBTable
    :column-list="COLUMN_LIST"
    :data-source="dbExtensionList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :row-clickable="false"
  >
    <template #body="{ rowData: dbExtension }">
      <BBTableCell :left-padding="4" class="w-16">
        {{ dbExtension.name }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ dbExtension.version }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ dbExtension.schema }}
      </BBTableCell>
      <BBTableCell class="w-64">
        {{ dbExtension.description }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import { DBExtension } from "../types";
import { useI18n } from "vue-i18n";

export default {
  name: "DbExtensionTable",
  components: {},
  props: {
    dbExtensionList: {
      required: true,
      type: Object as PropType<DBExtension[]>,
    },
  },
  setup() {
    const { t } = useI18n();
    const COLUMN_LIST: BBTableColumn[] = [
      {
        title: t("common.name"),
      },
      {
        title: t("common.version"),
      },
      {
        title: t("common.schema"),
      },
      {
        title: t("common.description"),
      },
    ];
    return {
      COLUMN_LIST,
    };
  },
};
</script>
