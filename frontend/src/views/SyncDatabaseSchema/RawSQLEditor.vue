<template>
  <div class="w-full h-auto flex flex-col justify-start items-start">
    <div class="w-full h-auto shrink-0 flex flex-row justify-between items-end">
      <div class="flex flex-col justify-start items-start gap-y-2">
        <div class="flex flex-row justify-start items-center">
          <span class="mr-2 w-20 shrink-0 text-sm">{{
            $t("common.project")
          }}</span>
          <ProjectSelect
            :project="state.projectId"
            :disabled="viewMode"
            @update:project="handleProjectSelect"
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
        <label
          for="sql-file-input"
          class="text-sm border px-3 leading-8 flex items-center rounded cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
        >
          <heroicons-outline:arrow-up-tray
            class="w-4 h-auto mr-1 text-gray-500"
          />
          {{ $t("issue.upload-sql") }}
          <input
            id="sql-file-input"
            type="file"
            accept=".sql,.txt,application/sql,text/plain"
            class="hidden"
            @change="handleUploadFile"
          />
        </label>
      </div>
    </div>
    <BBAttention
      v-if="isSheetOversized && !viewMode"
      :class="'my-2'"
      :style="`WARN`"
      :title="$t('issue.statement-from-sheet-warning')"
    >
      <template v-if="sheet?.name" #action>
        <DownloadSheetButton :sheet="sheet.name" size="small" />
      </template>
    </BBAttention>
    <div class="mt-4 w-full h-96 overflow-hidden">
      <MonacoEditor
        class="w-full h-full border"
        :value="state.editStatement"
        :auto-focus="true"
        :language="'sql'"
        :readonly="viewMode"
        :dialect="dialectOfEngineV1(state.engine)"
        @change="handleStatementChange"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NSelect } from "naive-ui";
import { computed, onMounted, nextTick, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useNotificationStore, useSheetV1Store } from "@/store";
import {
  DEFAULT_PROJECT_V1_NAME,
  UNKNOWN_ID,
  dialectOfEngineV1,
} from "@/types";
import DownloadSheetButton from "@/components/Sheet/DownloadSheetButton.vue";
import { Engine } from "@/types/proto/v1/common";
import { engineNameV1 } from "@/utils";
import {
  Sheet_Source,
  Sheet_Type,
  Sheet_Visibility,
} from "@/types/proto/v1/sheet_service";
import { useDebounceFn } from "@vueuse/core";
import MonacoEditor from "@/components/MonacoEditor";
import { ProjectSelect } from "@/components/v2";
import { RawSQLState } from "./types";

const MAX_UPLOAD_FILE_SIZE_MB = 1;

interface LocalState {
  projectId?: string;
  engine: Engine;
  editStatement: string;
}

const props = defineProps<{
  projectId?: string;
  engine: Engine;
  statement?: string;
  sheetId?: number;
  viewMode?: boolean;
}>();

const emit = defineEmits<{
  (event: "update", rawSQLState: RawSQLState): void;
}>();

const { t } = useI18n();
const notificationStore = useNotificationStore();
const sheetStore = useSheetV1Store();
const state = reactive<LocalState>({
  projectId: props.projectId,
  engine: props.engine || Engine.MYSQL,
  editStatement: props.statement || "",
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
  return sheetStore.getSheetByUid(props.sheetId || UNKNOWN_ID);
});

const isSheetOversized = computed(() => {
  if (!sheet.value) {
    return false;
  }
  return (
    new TextDecoder().decode(sheet.value.content).length <
    sheet.value.contentSize
  );
});

onMounted(async () => {
  if (props.sheetId) {
    const sheet = await sheetStore.getOrFetchSheetByUid(props.sheetId);
    if (sheet) {
      const statement = new TextDecoder().decode(sheet.content);
      state.editStatement = statement;
    }
  }
});

const handleProjectSelect = (uid: string | undefined) => {
  if (!uid || uid === String(UNKNOWN_ID)) return;
  state.projectId = uid;
  update();
};

const handleEngineChange = () => {
  nextTick(() => {
    update();
  });
};

const handleUploadFile = (e: Event) => {
  const target = e.target as HTMLInputElement;
  const file = (target.files || [])[0];
  const cleanup = () => {
    // Note that once selected a file, selecting the same file again will not
    // trigger <input type="file">'s change event.
    // So we need to do some cleanup stuff here.
    target.files = null;
    target.value = "";
  };

  if (!file) {
    return cleanup();
  }
  if (file.size > MAX_UPLOAD_FILE_SIZE_MB * 1024 * 1024) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("issue.upload-sql-file-max-size-exceeded", {
        size: `${MAX_UPLOAD_FILE_SIZE_MB}MB`,
      }),
    });
    return cleanup();
  }
  const fr = new FileReader();
  fr.onload = async () => {
    const statement = fr.result as string;
    const sheet = await sheetStore.createSheet(DEFAULT_PROJECT_V1_NAME, {
      name: file.name,
      content: new TextEncoder().encode(statement),
      visibility: Sheet_Visibility.VISIBILITY_PROJECT,
      source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
      type: Sheet_Type.TYPE_SQL,
    });
    const sheetId = sheetStore.getSheetUid(sheet.name);
    await handleStatementChange(statement);
    update(sheetId);
  };
  fr.onerror = () => {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "WARN",
      title: `Read file error`,
      description: String(fr.error),
    });
    return;
  };
  fr.readAsText(file);

  cleanup();
};

const handleStatementChange = async (statement: string) => {
  await handleUpdateSheet(statement);
  state.editStatement = statement;
  update();
};

const handleUpdateSheet = useDebounceFn(async (statement: string) => {
  if (!sheet.value) {
    return;
  }

  await sheetStore.patchSheet({
    name: sheet.value.name,
    content: new TextEncoder().encode(statement),
  });
}, 1000);

const update = (sheetId?: number) => {
  if (sheet.value) {
    sheetId = sheetStore.getSheetUid(sheet.value?.name);
  }
  emit("update", {
    projectId: state.projectId,
    engine: state.engine,
    statement: state.editStatement,
    sheetId: sheetId,
  });
};
</script>
