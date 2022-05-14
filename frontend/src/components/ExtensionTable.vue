<template>
  <BBTable
    :column-list="COLUMN_LIST"
    :data-source="extensionList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :row-clickable="false"
  >
    <template #body="{ rowData: extension }">
      <BBTableCell :left-padding="4" class="w-16">
        {{ extension.name }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ extension.version }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ extension.schema }}
      </BBTableCell>
      <BBTableCell class="w-64">
        {{ extension.description }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import { Extension } from "../types";
import { useI18n } from "vue-i18n";

export default {
  name: "ExtensionTable",
  components: {},
  props: {
    extensionList: {
      required: true,
      type: Object as PropType<Extension[]>,
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
