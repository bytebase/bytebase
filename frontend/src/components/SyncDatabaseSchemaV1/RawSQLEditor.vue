<template>
  <div class="w-full h-auto flex flex-col justify-start items-start">
    <div class="w-full h-auto shrink-0 flex flex-row justify-between items-end">
      <div class="flex flex-col justify-start items-start gap-y-2">
        <div class="flex flex-row justify-start items-center">
          <span class="mr-2 shrink-0 text-sm">{{ $t("database.engine") }}</span>
          <NSelect
            v-model:value="state.engine"
            :consistent-menu-width="false"
            :options="engineSelectorOptions"
            @update:value="handleEngineChange"
          />
        </div>
      </div>
      <div class="flex flex-row justify-end items-center gap-x-3">
        <SQLUploadButton @update:sql="handleStatementChange">
          {{ $t("issue.upload-sql") }}
        </SQLUploadButton>
      </div>
    </div>
    <div class="mt-4 w-full h-96 overflow-hidden">
      <MonacoEditor
        class="w-full h-full border"
        :content="state.statement"
        :auto-focus="true"
        :dialect="dialectOfEngineV1(state.engine)"
        @update:content="handleStatementChange"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NSelect } from "naive-ui";
import { computed, nextTick, reactive } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import SQLUploadButton from "@/components/misc/SQLUploadButton.vue";
import { dialectOfEngineV1 } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { engineNameV1 } from "@/utils";
import { ALLOWED_ENGINES, type RawSQLState } from "./types";

interface LocalState {
  engine: Engine;
  statement: string;
}

const props = defineProps<{
  project: Project;
  engine: Engine;
  statement?: string;
}>();

const emit = defineEmits<{
  (event: "update", rawSQLState: RawSQLState): void;
}>();

const state = reactive<LocalState>({
  engine: props.engine || Engine.MYSQL,
  statement: props.statement || "",
});

const engineSelectorOptions = computed(() => {
  return ALLOWED_ENGINES.map((engine) => {
    return {
      label: engineNameV1(engine),
      value: engine,
    };
  });
});

const handleEngineChange = () => {
  nextTick(() => {
    update();
  });
};

const handleStatementChange = (statement: string) => {
  state.statement = statement;
  nextTick(() => {
    update();
  });
};

const update = () => {
  emit("update", {
    engine: state.engine,
    statement: state.statement,
  });
};
</script>
