<template>
  <div class="w-full mb-2">
    <NRadioGroup
      v-model:value="state.exportMethod"
      class="w-full !flex flex-row justify-start items-center gap-4"
    >
      <NRadio :value="'SQL'" label="SQL" />
      <NTooltip :disabled="allowSelectTableResource">
        <template #trigger>
          <NRadio
            :disabled="!allowSelectTableResource"
            :value="'DATABASE'"
            :label="$t('common.database')"
          />
        </template>
        {{ $t("issue.grant-request.please-select-database-first") }}
      </NTooltip>
    </NRadioGroup>
  </div>
  <div
    v-show="state.exportMethod === 'SQL'"
    class="w-full h-[300px] border rounded"
  >
    <MonacoEditor
      class="w-full h-full py-2"
      :value="state.statement"
      :auto-focus="false"
      :language="'sql'"
      :dialect="dialect"
      @change="handleStatementChange"
    />
  </div>
  <div
    v-if="state.exportMethod === 'DATABASE'"
    class="w-full flex flex-row justify-start items-center"
  >
    <DatabaseResourceSelector
      :project-id="project!.uid"
      :database-id="(props.databaseId as string)"
      :database-resources="state.databaseResources"
      @update="handleTableResourceUpdate"
    />
  </div>
</template>

<script lang="ts" setup>
import { isUndefined } from "lodash-es";
import { NRadioGroup, NRadio, NTooltip } from "naive-ui";
import { computed, onMounted, reactive, watch } from "vue";
import MonacoEditor from "@/components/MonacoEditor/MonacoEditor.vue";
import { useDatabaseV1Store, useProjectV1Store } from "@/store";
import {
  DatabaseResource,
  SQLDialect,
  UNKNOWN_ID,
  dialectOfEngineV1,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { stringifyDatabaseResources } from "@/utils/issue/cel";
import DatabaseResourceSelector from "./DatabaseResourceSelector.vue";

const props = defineProps<{
  projectId?: string;
  databaseId?: string;
  statement?: string;
  databaseResources?: DatabaseResource[];
}>();

const emit = defineEmits<{
  (event: "update:condition", value?: string): void;
  (
    event: "update:database-resources",
    databaseResources: DatabaseResource[]
  ): void;
}>();

interface LocalState {
  exportMethod: "SQL" | "DATABASE";
  statement: string;
  databaseResources: DatabaseResource[];
}

const projectStore = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const state = reactive<LocalState>({
  exportMethod: "SQL",
  statement: props.statement || "",
  databaseResources: props.databaseResources || [],
});

const project = computed(() => {
  return props.projectId
    ? projectStore.getProjectByUID(props.projectId)
    : undefined;
});

const selectedDatabase = computed(() => {
  if (!props.databaseId || props.databaseId === String(UNKNOWN_ID)) {
    return undefined;
  }
  return databaseStore.getDatabaseByUID(props.databaseId);
});

const allowSelectTableResource = computed(() => {
  return props.databaseId !== undefined;
});

const dialect = computed((): SQLDialect => {
  const db = selectedDatabase.value;
  return dialectOfEngineV1(db?.instanceEntity.engine ?? Engine.MYSQL);
});

onMounted(() => {
  if (
    props.databaseResources &&
    props.databaseResources.length > 0 &&
    isUndefined(props.statement)
  ) {
    state.exportMethod = "DATABASE";
  }
});

watch(
  () => [state.exportMethod, state.statement, state.databaseResources],
  () => {
    if (state.exportMethod === "SQL") {
      if (state.statement === "") {
        emit("update:condition", undefined);
      } else {
        emit(
          "update:condition",
          `request.statement == "${btoa(
            unescape(encodeURIComponent(state.statement))
          )}"`
        );
      }
    } else {
      if (state.databaseResources.length === 0) {
        emit("update:condition", undefined);
      } else {
        const condition = stringifyDatabaseResources(state.databaseResources);
        emit("update:condition", condition);
      }
      emit("update:database-resources", state.databaseResources);
    }
  },
  {
    immediate: true,
  }
);

const handleStatementChange = (value: string) => {
  state.statement = value;
};

const handleTableResourceUpdate = (
  databaseResourceList: DatabaseResource[]
) => {
  state.databaseResources = databaseResourceList;
};
</script>
