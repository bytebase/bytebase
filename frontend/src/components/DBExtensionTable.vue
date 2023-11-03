<template>
  <BBTable
    :column-list="columnList"
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
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { ExtensionMetadata } from "@/types/proto/v1/database_service";

export default {
  name: "DbExtensionTable",
  components: {},
  props: {
    dbExtensionList: {
      required: true,
      type: Object as PropType<ExtensionMetadata[]>,
    },
  },
  setup() {
    const { t } = useI18n();
    const columnList = computed(() => [
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
    ]);
    return {
      columnList,
    };
  },
};
</script>
