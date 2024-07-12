<template>
  <div class="w-full h-auto flex flex-col justify-start items-start">
    <div class="w-full h-auto shrink-0 flex flex-row justify-between items-end">
      <div class="flex flex-col justify-start items-start gap-y-2">
        <div
          v-if="!disableProjectSelect"
          class="flex flex-row justify-start items-center"
        >
          <span class="mr-2 w-20 shrink-0 text-sm">{{
            $t("common.project")
          }}</span>
          <ProjectSelect
            :project-name="state.projectName"
            :disabled="viewMode"
            @update:project-name="handleProjectSelect"
          />
        </div>
        <div class="flex flex-row justify-start items-center">
          <span class="mr-2 w-20 shrink-0 text-sm">{{
            $t("database.engine")
          }}</span>
          <NSelect
            v-model:value="state.engine"
            :disabled="viewMode"
            :consistent-menu-width="false"
            :options="engineSelectorOptions"
            @update:value="handleEngineChange"
          />
        </div>
      </div>
      <div
        v-if="!viewMode"
        class="flex flex-row justify-end items-center space-x-3"
      >
        <SQLUploadButton
          :loading="state.isUploadingFile"
          @update:sql="handleSQLUpload"
        >
          {{ $t("issue.upload-sql") }}
        </SQLUploadButton>
      </div>
    </div>
    <BBAttention
      v-if="isSheetOversized && !viewMode"
      :class="'my-2'"
      type="warning"
      :title="$t('issue.statement-from-sheet-warning')"
    >
      <template v-if="sheet?.name" #action>
        <DownloadSheetButton :sheet="sheet.name" size="small" />
      </template>
    </BBAttention>
    <div class="mt-4 w-full h-96 overflow-hidden">
      <MonacoEditor
        class="w-full h-full border"
        :content="state.editStatement"
        :auto-focus="true"
        :readonly="viewMode"
        :dialect="dialectOfEngineV1(state.engine)"
        @update:content="handleStatementChange"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import { NSelect } from "naive-ui";
import { computed, onMounted, nextTick, reactive } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import DownloadSheetButton from "@/components/Sheet/DownloadSheetButton.vue";
import SQLUploadButton from "@/components/misc/SQLUploadButton.vue";
import { ProjectSelect } from "@/components/v2";
import { useSheetV1Store } from "@/store";
import {
  DEFAULT_PROJECT_NAME,
  UNKNOWN_ID,
  dialectOfEngineV1,
  isValidProjectName,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  engineNameV1,
  extractSheetUID,
  getStatementSize,
  getSheetStatement,
} from "@/utils";
import type { RawSQLState } from "./types";

interface LocalState {
  projectName?: string;
  engine: Engine;
  editStatement: string;
  isUploadingFile: boolean;
}

const props = defineProps<{
  projectName?: string;
  engine: Engine;
  statement?: string;
  sheetId?: number;
  viewMode?: boolean;
  disableProjectSelect?: boolean;
}>();

const emit = defineEmits<{
  (event: "update", rawSQLState: RawSQLState): void;
}>();

const sheetStore = useSheetV1Store();
const state = reactive<LocalState>({
  projectName: props.projectName,
  engine: props.engine || Engine.MYSQL,
  editStatement: props.statement || "",
  isUploadingFile: false,
});

const availableEngines = computed(() => {
  return [Engine.MYSQL, Engine.POSTGRES];
});

const engineSelectorOptions = computed(() => {
  return availableEngines.value.map((engine) => {
    return {
      label: engineNameV1(engine),
      value: engine,
    };
  });
});

const sheet = computed(() => {
  return sheetStore.getSheetByUID(String(props.sheetId || UNKNOWN_ID), "FULL");
});

const isSheetOversized = computed(() => {
  if (!sheet.value) {
    return false;
  }
  return getStatementSize(getSheetStatement(sheet.value)).lt(
    sheet.value.contentSize
  );
});

onMounted(async () => {
  if (props.sheetId) {
    const sheet = await sheetStore.getOrFetchSheetByUID(
      String(props.sheetId),
      "FULL"
    );
    if (sheet) {
      const statement = new TextDecoder().decode(sheet.content);
      state.editStatement = statement;
    }
  }
});

const handleProjectSelect = (name: string | undefined) => {
  if (!isValidProjectName(name)) return;
  state.projectName = name;
  update();
};

const handleEngineChange = () => {
  nextTick(() => {
    update();
  });
};

const handleSQLUpload = async (statement: string, filename: string) => {
  if (state.isUploadingFile) {
    return;
  }

  state.isUploadingFile = true;
  try {
    const sheet = await sheetStore.createSheet(DEFAULT_PROJECT_NAME, {
      title: filename,
      engine: state.engine,
      content: new TextEncoder().encode(statement),
    });
    const sheetId = Number(extractSheetUID(sheet.name));
    await handleStatementChange(statement);
    update(sheetId);
  } finally {
    state.isUploadingFile = false;
  }
};

const handleStatementChange = (statement: string) => {
  state.editStatement = statement;
  handleUpdateSheet(statement);
  update();
};

const handleUpdateSheet = useDebounceFn(async (statement: string) => {
  if (!sheet.value) {
    return;
  }

  await sheetStore.patchSheetContent({
    name: sheet.value.name,
    content: new TextEncoder().encode(statement),
  });
}, 1000);

const update = (sheetId?: number) => {
  if (sheet.value) {
    sheetId = Number(extractSheetUID(sheet.value.name));
  }
  emit("update", {
    projectName: state.projectName,
    engine: state.engine,
    statement: state.editStatement,
    sheetId: sheetId,
  });
};
</script>
