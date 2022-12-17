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
        {{ view.name }}
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
import { ViewMetadata } from "@/types/proto/database";

export default {
  name: "ViewTable",
  components: {},
  props: {
    viewList: {
      required: true,
      type: Object as PropType<ViewMetadata[]>,
    },
  },
  setup() {
    const { t } = useI18n();
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

    return {
      columnList,
    };
  },
};
</script>
