<template>
  <BBTable
    :column-list="columnList"
    :data-source="functionList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :row-clickable="true"
    :custom-footer="true"
  >
    <template #body="{ rowData: functionItem }">
      <BBTableCell v-if="hasSchemaProperty" :left-padding="4" class="w-[10%]">
        {{ schemaName }}
      </BBTableCell>
      <BBTableCell :left-padding="hasSchemaProperty ? 2 : 4">
        <div class="flex items-center space-x-2">
          <EllipsisText>{{ functionItem.name }}</EllipsisText>
        </div>
      </BBTableCell>
      <BBTableCell>
        <DefinitionView :definition="functionItem.definition" />
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import DefinitionView from "@/components/DefinitionView.vue";
import EllipsisText from "@/components/EllipsisText.vue";
import { ComposedDatabase } from "@/types";
import { FunctionMetadata } from "@/types/proto/store/database";
import { Engine } from "@/types/proto/v1/common";

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  schemaName: {
    type: String,
    default: "",
  },
  functionList: {
    required: true,
    type: Object as PropType<FunctionMetadata[]>,
  },
});

const { t } = useI18n();

const engine = computed(() => props.database.instanceEntity.engine);

const isPostgres = computed(() => engine.value === Engine.POSTGRES);

const hasSchemaProperty = computed(() => {
  return (
    isPostgres.value ||
    engine.value === Engine.SNOWFLAKE ||
    engine.value === Engine.ORACLE ||
    engine.value === Engine.DM ||
    engine.value === Engine.MSSQL
  );
});

const columnList = computed(() => {
  if (hasSchemaProperty.value) {
    return [
      {
        title: t("common.schema"),
      },
      {
        title: t("common.name"),
      },
      {
        title: t("common.definition"),
      },
    ];
  } else {
    return [
      {
        title: t("common.name"),
      },
      {
        title: t("common.definition"),
      },
    ];
  }
});
</script>
