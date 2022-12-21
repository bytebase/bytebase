<template>
  <div
    class="flex flex-col md:flex-row md:justify-between md:items-center gap-2 md:gap-4"
  >
    <div class="flex space-x-4 flex-1">
      <div
        class="text-sm font-medium"
        :class="isEmpty(state.editStatement) ? 'text-red-600' : 'text-control'"
      >
        {{ $t("common.sql") }}
        <span v-if="create" class="text-red-600">*</span>
        <span v-if="sqlHint" class="text-accent">{{ `(${sqlHint})` }}</span>
      </div>
      <button
        v-if="allowApplyStatementToOtherTasks"
        :disabled="isEmpty(state.editStatement)"
        type="button"
        class="btn-small"
        @click.prevent="applyStatementToOtherTasks(state.editStatement)"
      >
        {{ $t("issue.apply-to-other-tasks") }}
      </button>
    </div>

    <div class="space-x-2 flex items-center">
      <template v-if="create || state.editing">
        <!-- mt-0.5 is to prevent jiggling between switching edit/none-edit -->
        <label class="mt-0.5 mr-2 inline-flex items-center gap-1">
          <input
            v-model="formatOnSave"
            type="checkbox"
            class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
          />
          <span class="textlabel">{{ $t("issue.format-on-save") }}</span>
        </label>
        <button
          v-if="state.editing && allowUploadSheetForTask"
          type="button"
          class="cursor-pointer border border-control-border rounded text-control bg-control-bg hover:bg-control-bg-hover disabled:bg-control-bg-hover disabled:cursor-not-allowed disabled:opacity-60 text-sm font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
          :disabled="state.isUploadingFile"
        >
          <label
            for="sql-file-with-sheet-input"
            class="px-3 py-1 w-full flex flex-row justify-center items-center cursor-pointer"
            :class="state.isUploadingFile && 'cursor-wait'"
          >
            <heroicons-outline:document-text class="w-4 h-auto mr-1" />
            {{ $t("issue.upload-sql-as-sheet") }}
            <input
              id="sql-file-with-sheet-input"
              type="file"
              accept=".sql,.txt,application/sql,text/plain"
              class="hidden"
              @change="handleUploadLocalFileAsSheet"
            />
          </label>
        </button>
        <button
          v-if="state.editing"
          type="button"
          class="cursor-pointer border border-control-border rounded text-control bg-control-bg hover:bg-control-bg-hover text-sm font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
        >
          <label
            for="sql-file-input"
            class="px-3 py-1 w-full flex flex-row justify-center items-center cursor-pointer"
          >
            <heroicons-outline:arrow-up-tray class="w-4 h-auto mr-1" />
            {{ $t("issue.upload-sql") }}
            <input
              id="sql-file-input"
              type="file"
              accept=".sql,.txt,application/sql,text/plain"
              class="hidden"
              @change="handleUploadLocalFile"
            />
          </label>
        </button>
      </template>

      <button
        v-if="shouldShowStatementEditButton"
        type="button"
        class="btn-icon"
        @click.prevent="beginEdit"
      >
        <heroicons-solid:pencil class="h-5 w-5" />
      </button>
      <template v-else-if="!create">
        <button
          v-if="state.editing"
          type="button"
          class="px-3 py-1 cursor-pointer border border-control-border rounded text-control bg-control-bg hover:bg-control-bg-hover text-sm font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
          :disabled="!allowSaveSQL"
          @click.prevent="saveEdit"
        >
          {{ $t("common.save") }}
        </button>
        <button
          v-if="state.editing"
          type="button"
          class="px-3 py-1 cursor-pointer rounded text-control hover:bg-control-bg-hover text-sm font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
          @click.prevent="cancelEdit"
        >
          {{ $t("common.cancel") }}
        </button>
      </template>

      <template
        v-if="!create && (issue as Issue).project.workflowType === 'VCS'"
      >
        <!--
          Show a virtual pencil icon in VCS workflow which opens a guide
          when clicked.
        -->
        <button
          type="button"
          class="btn-icon"
          @click.prevent="state.showVCSGuideModal = true"
        >
          <heroicons-solid:pencil class="h-5 w-5" />
        </button>
      </template>

      <IssueRollbackButton />
    </div>
  </div>
  <label class="sr-only">{{ $t("common.sql-statement") }}</label>
  <BBAttention
    v-if="isTaskHasSheetId"
    :class="'my-2'"
    :style="`WARN`"
    :title="$t('issue.statement-from-sheet-warning')"
  />
  <div
    class="whitespace-pre-wrap mt-2 w-full overflow-hidden"
    :class="state.editing ? 'border-t border-x' : 'border-t border-x'"
  >
    <MonacoEditor
      ref="editorRef"
      class="w-full h-auto max-h-[360px]"
      data-label="bb-issue-sql-editor"
      :value="state.editStatement"
      :readonly="!state.editing || !allowEditStatement || isTaskHasSheetId"
      :auto-focus="false"
      :dialect="dialect"
      @change="onStatementChange"
      @ready="handleMonacoEditorReady"
    />
  </div>

  <BBModal
    v-if="state.showVCSGuideModal"
    :title="$t('issue.edit-sql-statement')"
    @close="state.showVCSGuideModal = false"
  >
    <div class="space-y-4 max-w-[32rem] divide-y divide-block-border">
      <div class="whitespace-pre-wrap">
        {{ $t("issue.edit-sql-statement-in-vcs") }}
      </div>

      <div class="flex justify-end pt-4 gap-x-2">
        <button
          type="button"
          class="btn-normal"
          @click.prevent="state.showVCSGuideModal = false"
        >
          {{ $t("common.cancel") }}
        </button>

        <button type="button" class="btn-primary" @click.prevent="goToVCS">
          {{ $t("common.go-now") }}
        </button>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { useDialog } from "naive-ui";
import { onMounted, reactive, watch, computed, ref, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useDBSchemaStore,
  useRepositoryStore,
  useSheetStore,
  useUIStateStore,
} from "@/store";
import { useIssueLogic, TaskTypeWithSheetId, sheetIdOfTask } from "./logic";
import type {
  Database,
  Issue,
  Repository,
  SQLDialect,
  Task,
  TaskCreate,
  TaskId,
} from "@/types";
import { baseDirectoryWebUrl, UNKNOWN_ID } from "@/types";
import { TableMetadata } from "@/types/proto/database";
import MonacoEditor from "../MonacoEditor/MonacoEditor.vue";
import IssueRollbackButton from "./IssueRollbackButton.vue";
import { isUndefined } from "lodash-es";

interface LocalState {
  editing: boolean;
  editStatement: string;
  showVCSGuideModal: boolean;
  isUploadingFile: boolean;
}

type LocalEditState = Pick<LocalState, "editing" | "editStatement">;

const EDITOR_MIN_HEIGHT = {
  READONLY: 0, // not limited to keep the UI compact
  EDITABLE: 120, // ~= 6 lines, a reasonable size to start writing SQL
};

defineProps({
  sqlHint: {
    required: false,
    type: String,
    default: undefined,
  },
});

const {
  issue,
  create,
  allowEditStatement,
  selectedDatabase,
  selectedStatement: statement,
  selectedTask,
  updateStatement,
  updateSheetId,
  allowApplyStatementToOtherTasks,
  applyStatementToOtherTasks,
} = useIssueLogic();

const { t } = useI18n();
const overrideSQLDialog = useDialog();
const uiStateStore = useUIStateStore();
const dbSchemaStore = useDBSchemaStore();
const sheetStore = useSheetStore();
const editorRef = ref<InstanceType<typeof MonacoEditor>>();

const state = reactive<LocalState>({
  editing: false,
  editStatement: statement.value,
  showVCSGuideModal: false,
  isUploadingFile: false,
});

const useTempEditState = (state: LocalState) => {
  const { create, selectedTask, selectedStatement } = useIssueLogic();

  let stopWatching: (() => void) | null = null;

  const startWatching = () => {
    const tempEditStateMap = new Map<TaskId, LocalEditState>();
    const isSwitchingTask = ref(false);

    // The issue page is polling the issue entity, making the reference obj
    // of `selectedTask` changes every time.
    // So we need to watch the id instead of the object ref.
    const selectedTaskId = computed(() => {
      if (create.value) return UNKNOWN_ID;
      return (selectedTask.value as Task).id;
    });

    watch(selectedTaskId, () => {
      isSwitchingTask.value = true;
      nextTick(() => {
        isSwitchingTask.value = false;
      });
    });

    const handleEditChange = () => {
      // When we are switching between tasks, this will also be triggered.
      // But we shouldn't update the temp store.
      if (isSwitchingTask.value) {
        return;
      }
      // Save the temp edit state before switching task.
      tempEditStateMap.set(selectedTaskId.value, {
        editing: state.editing,
        editStatement: state.editStatement,
      });
    };

    const afterTaskIdChange = (id: TaskId) => {
      // Try to restore the saved temp edit state after switching task.
      const storedState = tempEditStateMap.get(id);
      if (storedState) {
        // If found the stored temp edit state, restore it.
        Object.assign(state, storedState);
      } else {
        // Restore to the task's default state otherwise.
        state.editing = false;
        state.editStatement = selectedStatement.value;
      }
    };

    // Save the temp editing state before switching tasks
    const stopWatchBeforeChange = watch(
      [() => state.editing, () => state.editStatement],
      handleEditChange,
      { immediate: true }
    );
    const stopWatchAfterChange = watch(
      selectedTaskId,
      (id) => {
        afterTaskIdChange(id);
      },
      { flush: "post" } // Listen to the event AFTER selectedTaskId changed
    );

    return () => {
      tempEditStateMap.clear();
      stopWatchBeforeChange();
      stopWatchAfterChange();
    };
  };

  watch(
    create,
    () => {
      if (!create.value) {
        // If we are opening an existed issue, we should listen and store the
        // temp editing states.
        stopWatching = startWatching();
      } else {
        // If we are creating an issue, we don't need the temp editing state
        // feature since all tasks are still in editing mode.
        stopWatching && stopWatching();
      }
    },
    { immediate: true }
  );
};

useTempEditState(state);

const dialect = computed((): SQLDialect => {
  const db = selectedDatabase.value;
  if (db?.instance.engine === "POSTGRES") {
    return "postgresql";
  }
  // fallback to mysql dialect anyway
  return "mysql";
});

const formatOnSave = computed({
  get: () => uiStateStore.issueFormatStatementOnSave,
  set: (value: boolean) => uiStateStore.setIssueFormatStatementOnSave(value),
});

const allowUploadSheetForTask = computed(() => {
  const task = selectedTask.value;
  return TaskTypeWithSheetId.includes(task.type);
});

const isTaskHasSheetId = computed(() => {
  if (!allowUploadSheetForTask.value) {
    return false;
  }

  const task = selectedTask.value;
  if (create.value) {
    return !isUndefined((task as TaskCreate).sheetId);
  }
  return !isUndefined(sheetIdOfTask(task as Task));
});

const shouldShowStatementEditButton = computed(() => {
  if (create.value) {
    return false;
  }
  // For those task with sheet, it's readonly.
  if (isTaskHasSheetId.value) {
    return false;
  }
  if (state.editing) {
    return false;
  }

  return allowEditStatement;
});

onMounted(() => {
  if (create.value) {
    state.editing = true;
  }
});

// Reset the edit state after creating the issue.
watch(create, (curNew, prevNew) => {
  if (!curNew && prevNew) {
    if (formatOnSave.value) {
      editorRef.value?.formatEditorContent();
    }
    state.editing = false;
    updateEditorHeight();
  }
});

watch(statement, (cur) => {
  state.editStatement = cur;
});

const beginEdit = () => {
  state.editStatement = statement.value;
  state.editing = true;
};

const saveEdit = () => {
  if (formatOnSave.value) {
    editorRef.value?.formatEditorContent();
  }
  updateStatement(state.editStatement, () => {
    state.editing = false;
  });
};

const cancelEdit = () => {
  state.editStatement = statement.value;
  state.editing = false;
};

const allowSaveSQL = computed((): boolean => {
  if (state.editStatement === "") {
    // Not allowed if the statement is empty.
    return false;
  }
  if (state.editStatement === statement.value) {
    // Not allowed if the statement is not modified.
    return false;
  }

  // Allowed to save otherwise
  return true;
});

const handleUploadFileEvent = (
  event: Event,
  maxFileSizeMB: number
): Promise<{
  filename: string;
  content: string;
}> => {
  return new Promise((resolve) => {
    const target = event.target as HTMLInputElement;
    const file = (target.files || [])[0];
    const cleanup = () => {
      // Note that once selected a file, selecting the same file again will not
      // trigger <input type="file">'s change event.
      // So we need to do some cleanup stuff here.
      target.files = null;
      target.value = "";
    };

    if (!file) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "File not found",
      });
      cleanup();
      return;
    }
    if (file.size > maxFileSizeMB * 1024 * 1024) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("issue.upload-sql-file-max-size-exceeded", {
          size: `${maxFileSizeMB}MB`,
        }),
      });
      cleanup();
      return;
    }

    const fr = new FileReader();
    fr.onload = async () => {
      const content = fr.result as string;
      resolve({
        filename: file.name,
        content: content,
      });
    };
    fr.onerror = () => {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: "Read file error",
        description: String(fr.error),
      });
    };
    fr.readAsText(file);
    cleanup();
  });
};

const handleUploadLocalFile = async (event: Event) => {
  const { content: statement } = await handleUploadFileEvent(event, 1);
  if (state.editStatement) {
    // Show a confirm dialog before replacing if the editing statement
    // is not empty
    overrideSQLDialog.create({
      positiveText: t("common.confirm"),
      negativeText: t("common.cancel"),
      title: t("issue.override-current-statement"),
      onNegativeClick: () => {
        // nothing to do
      },
      onPositiveClick: () => {
        updateStatement(statement);
      },
    });
  } else {
    updateStatement(statement);
  }
};

const handleUploadLocalFileAsSheet = async (event: Event) => {
  if (!allowUploadSheetForTask.value || !selectedDatabase.value) {
    return;
  }

  state.isUploadingFile = true;
  const projectId = selectedDatabase.value.projectId;
  const { filename, content: statement } = await handleUploadFileEvent(
    event,
    100
  );

  const uploadStatementAsSheet = async (statement: string) => {
    const sheet = await sheetStore.createSheet({
      projectId: projectId,
      name: filename,
      statement: statement,
      visibility: "PRIVATE",
      payload: {},
    });

    updateSheetId(sheet.id);
    updateStatement(sheet.statement);
    state.isUploadingFile = false;
    if (selectedTask.value) updateEditorHeight();
  };

  if (state.editStatement) {
    // Show a confirm dialog before replacing if the editing statement
    // is not empty
    overrideSQLDialog.create({
      positiveText: t("common.confirm"),
      negativeText: t("common.cancel"),
      title: t("issue.override-current-statement"),
      onNegativeClick: () => {
        // nothing to do
      },
      onPositiveClick: () => {
        uploadStatementAsSheet(statement);
      },
    });
  } else {
    uploadStatementAsSheet(statement);
  }
};

const goToVCS = () => {
  const issueEntity = issue.value as Issue;
  useRepositoryStore()
    .fetchRepositoryByProjectId(issueEntity.project.id)
    .then((repository: Repository) => {
      window.open(
        baseDirectoryWebUrl(repository, {
          DB_NAME: selectedDatabase.value?.name,
          ENV_NAME: selectedDatabase.value?.instance.environment.name,
        }),
        "_blank"
      );

      state.showVCSGuideModal = false;
    });
};

const onStatementChange = (value: string) => {
  state.editStatement = value;
  if (create.value) {
    // If we are creating an issue, emit the event immediately when every
    // time the user types.
    updateStatement(state.editStatement);
  }

  if (selectedTask.value) updateEditorHeight();
};

// Handle and update monaco editor auto completion context.
const useDatabaseAndTableList = () => {
  const { selectedDatabase } = useIssueLogic();
  const dbSchemaStore = useDBSchemaStore();

  const databaseList = computed(() => {
    if (selectedDatabase.value) return [selectedDatabase.value];
    return [];
  });

  watch(
    databaseList,
    (list) => {
      list.forEach((db) => dbSchemaStore.getOrFetchDatabaseMetadataById(db.id));
    },
    { immediate: true }
  );

  const tableList = computed(() => {
    return databaseList.value
      .map((item) => dbSchemaStore.getTableListByDatabaseId(item.id))
      .flat();
  });

  return { databaseList, tableList };
};

const { databaseList, tableList } = useDatabaseAndTableList();

const handleUpdateEditorAutoCompletionContext = async () => {
  const databaseMap: Map<Database, TableMetadata[]> = new Map();
  for (const database of databaseList.value) {
    const tableList = await dbSchemaStore.getOrFetchTableListByDatabaseId(
      database.id
    );
    databaseMap.set(database, tableList);
  }
  editorRef.value?.setEditorAutoCompletionContext(databaseMap);
};

const updateEditorHeight = () => {
  const contentHeight =
    editorRef.value?.editorInstance?.getContentHeight() as number;
  let actualHeight = contentHeight;
  if (state.editing && actualHeight < EDITOR_MIN_HEIGHT.EDITABLE) {
    actualHeight = EDITOR_MIN_HEIGHT.EDITABLE;
  }
  editorRef.value?.setEditorContentHeight(actualHeight);
};

const handleMonacoEditorReady = () => {
  handleUpdateEditorAutoCompletionContext();
  updateEditorHeight();
};

watch([databaseList, tableList], () => {
  handleUpdateEditorAutoCompletionContext();
});
</script>
