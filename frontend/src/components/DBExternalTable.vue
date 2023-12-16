<template>
  <BBTable
    :column-list="columnList"
    :data-source="dbExternalTableList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :row-clickable="false"
  >
    <template #body="{ rowData: dbExternalTable }">
      <BBTableCell :left-padding="4" class="w-16">
        {{ dbExternalTable.name }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ dbExternalTable.externalServerName }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ dbExternalTable.externalDatabaseName }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { ExternalTableMetadata } from "@/types/proto/v1/database_service";

export default {
  name: "DbExtensionTable",
  components: {},
  props: {
    dbExternalTableList: {
      required: true,
      type: Object as PropType<ExternalTableMetadata[]>,
      schemaName: "",
    },
  },
  setup() {
    const { t } = useI18n();
    const columnList = computed(() => [
      {
        title: t("common.name"),
      },
      {
        title: t("common.external-server-name"),
      },
      {
        title: t("common.external-database-name"),
      },
    ]);
    return {
      columnList,
    };
  },
};
</script>
