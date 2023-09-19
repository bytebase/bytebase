<template>
  <BBTable
    :column-list="columnList"
    :data-source="streamList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :row-clickable="true"
    :custom-footer="true"
  >
    <template #body="{ rowData: stream }: { rowData: StreamMetadata }">
      <BBTableCell :left-padding="4" class="w-24">
        {{ schemaName }}
      </BBTableCell>
      <BBTableCell>
        {{ stream.name }}
      </BBTableCell>
      <BBTableCell>
        {{ stream.tableName }}
      </BBTableCell>
      <BBTableCell>
        {{ stringifyStreamType(stream.type) }}
      </BBTableCell>
      <BBTableCell>
        {{ stringifyStreamMode(stream.mode) }}
      </BBTableCell>
      <BBTableCell>
        <DefinitionView :definition="stream.definition" />
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import DefinitionView from "@/components/DefinitionView.vue";
import { ComposedDatabase } from "@/types";
import {
  StreamMetadata,
  StreamMetadata_Mode,
  StreamMetadata_Type,
} from "@/types/proto/v1/database_service";

defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  schemaName: {
    type: String,
    default: "",
  },
  streamList: {
    required: true,
    type: Object as PropType<StreamMetadata[]>,
  },
});

const { t } = useI18n();

const columnList = computed(() => {
  return [
    {
      title: t("common.schema"),
    },
    {
      title: t("common.name"),
    },
    {
      title: t("common.table"),
    },
    {
      title: t("common.type"),
    },
    {
      title: t("common.mode"),
    },
    {
      title: t("common.definition"),
    },
  ];
});

const stringifyStreamType = (t: StreamMetadata_Type): string => {
  if (t === StreamMetadata_Type.TYPE_DELTA) {
    return "Delta";
  }
  return "-";
};

const stringifyStreamMode = (mode: StreamMetadata_Mode): string => {
  if (mode === StreamMetadata_Mode.MODE_APPEND_ONLY) {
    return "Append only";
  } else if (mode === StreamMetadata_Mode.MODE_INSERT_ONLY) {
    return "Insert only";
  } else if (mode === StreamMetadata_Mode.MODE_DEFAULT) {
    return "default";
  }
  return "-";
};
</script>
