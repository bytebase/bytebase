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
        <FunctionDefinitionView :definition="functionItem.definition" />
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { Database } from "@/types";
import { FunctionMetadata } from "@/types/proto/store/database";
import EllipsisText from "@/components/EllipsisText.vue";
import FunctionDefinitionView from "@/components/FunctionDefinitionView.vue";

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<Database>,
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

const isPostgres = props.database.instance.engine === "POSTGRES";

const hasSchemaProperty =
  isPostgres ||
  props.database.instance.engine === "SNOWFLAKE" ||
  props.database.instance.engine === "ORACLE" ||
  props.database.instance.engine === "MSSQL";

const columnList = computed(() => {
  if (hasSchemaProperty) {
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
