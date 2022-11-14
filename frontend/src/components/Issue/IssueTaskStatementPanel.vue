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
      <template v-if="create">
        <label class="mt-0.5 mr-2 inline-flex items-center gap-1">
          <input
            v-model="formatOnSave"
            type="checkbox"
            class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
          />
          <span class="textlabel">{{ $t("issue.format-on-save") }}</span>
        </label>
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
              @change="handleUploadFile"
            />
          </label>
        </button>
      </template>
      <template v-else>
        <button
          v-if="allowEditStatement && !state.editing"
          type="button"
          class="btn-icon"
          @click.prevent="beginEdit"
        >
          <!-- Heroicon name: solid/pencil -->
          <!-- Use h-5 to avoid flickering when show/hide icon -->
          <heroicons-solid:pencil class="h-5 w-5" />
        </button>
        <template v-if="state.editing">
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
            v-if="state.editing"
            type="button"
            class="mt-0.5 px-3 border border-control-border rounded-sm text-control bg-control-bg hover:bg-control-bg-hover text-sm leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2 relative"
          >
            {{ $t("issue.upload-sql") }}
            <input
              type="file"
              accept=".sql,.txt,application/sql,text/plain"
              class="absolute inset-0 z-1 opacity-0"
              @change="handleUploadFile"
            />
          </button>
          <button
            v-if="state.editing"
            type="button"
            class="mt-0.5 px-3 border border-control-border rounded-sm text-control bg-control-bg hover:bg-control-bg-hover disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed text-sm leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
            :disabled="!allowSaveSQL"
            @click.prevent="saveEdit"
          >
            {{ $t("common.save") }}
          </button>
          <button
            v-if="state.editing"
            type="button"
            class="mt-0.5 px-3 rounded-sm text-control hover:bg-control-bg-hover disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed text-sm leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
            @click.prevent="cancelEdit"
          >
            {{ $t("common.cancel") }}
          </button>
        </template>
        <template v-if="issue.project.workflowType === 'VCS'">
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
      </template>
    </div>
  </div>
  <label class="sr-only">{{ $t("common.sql-statement") }}</label>
  <div
    class="whitespace-pre-wrap mt-2 w-full overflow-hidden"
    :class="state.editing ? 'border-t border-x' : 'border-t border-x'"
  >
    <MonacoEditor
      ref="editorRef"
      class="w-full h-auto max-h-[360px]"
      data-label="bb-issue-sql-editor"
      :value="state.editStatement"
      :readonly="!state.editing"
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

<script lang="ts">
import {
  onMounted,
  reactive,
  watch,
  defineComponent,
  computed,
  ref,
  Ref,
} from "vue";
import { useDialog } from "naive-ui";
import {
  pushNotification,
  useRepositoryStore,
  useTableStore,
  useUIStateStore,
} from "@/store";
import { useIssueLogic } from "./logic";
import MonacoEditor from "../MonacoEditor/MonacoEditor.vue";
import type { Issue, Repository, SQLDialect, Task, TaskId } from "@/types";
import { baseDirectoryWebUrl, UNKNOWN_ID } from "@/types";
import { useI18n } from "vue-i18n";

interface LocalState {
  editing: boolean;
  editStatement: string;
  showVCSGuideModal: boolean;
}

const EDITOR_MIN_HEIGHT = {
  READONLY: 0, // not limited to keep the UI compact
  EDITABLE: 120, // ~= 6 lines, a reasonable size to start writing SQL
};

const MAX_UPLOAD_FILE_SIZE_MB = 1;

export default defineComponent({
  name: "IssueTaskStatementPanel",
  components: {
    MonacoEditor,
  },
  props: {
    sqlHint: {
      required: false,
      type: String,
      default: undefined,
    },
  },
  setup(props, { emit }) {
    const {
      issue,
      create,
      allowEditStatement,
      selectedDatabase,
      selectedStatement: statement,
      updateStatement,
      allowApplyStatementToOtherTasks,
      applyStatementToOtherTasks,
    } = useIssueLogic();

    const uiStateStore = useUIStateStore();
    const { t } = useI18n();

    const state = reactive<LocalState>({
      editing: false,
      editStatement: statement.value,
      showVCSGuideModal: false,
    });
    useTempEditState(state);

    const editorRef = ref<InstanceType<typeof MonacoEditor>>();
    const overrideSQLDialog = useDialog();

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
      set: (value: boolean) =>
        uiStateStore.setIssueFormatStatementOnSave(value),
    });

    const { databaseList, tableList } = useDatabaseAndTableList();

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

    const handleMonacoEditorReady = () => {
      editorRef.value?.setEditorAutoCompletionContext(
        databaseList.value,
        tableList.value
      );

      updateEditorHeight();
    };

    watch([databaseList, tableList], () => {
      editorRef.value?.setEditorAutoCompletionContext(
        databaseList.value,
        tableList.value
      );
    });

    const updateEditorHeight = () => {
      const contentHeight =
        editorRef.value?.editorInstance?.getContentHeight() as number;
      let actualHeight = contentHeight;
      if (state.editing && actualHeight < EDITOR_MIN_HEIGHT.EDITABLE) {
        actualHeight = EDITOR_MIN_HEIGHT.EDITABLE;
      }
      editorRef.value?.setEditorContentHeight(actualHeight);
    };

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
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("issue.upload-sql-file-max-size-exceeded", {
            size: `${MAX_UPLOAD_FILE_SIZE_MB}MB`,
          }),
        });
        return cleanup();
      }
      const fr = new FileReader();
      fr.onload = () => {
        const sql = fr.result as string;
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
              onStatementChange(sql);
            },
          });
        } else {
          onStatementChange(sql);
        }
      };
      fr.onerror = (e) => {
        pushNotification({
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

    const onStatementChange = (value: string) => {
      state.editStatement = value;
      if (create.value) {
        // If we are creating an issue, emit the event immediately when every
        // time the user types.
        updateStatement(state.editStatement);
      }

      updateEditorHeight();
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

    return {
      issue: issue as Ref<Issue>,
      create,
      allowEditStatement,
      statement,
      allowApplyStatementToOtherTasks,
      dialect,
      formatOnSave,
      state,
      editorRef,
      updateStatement,
      applyStatementToOtherTasks,
      beginEdit,
      saveEdit,
      cancelEdit,
      allowSaveSQL,
      handleUploadFile,
      onStatementChange,
      goToVCS,
      handleMonacoEditorReady,
    };
  },
});

const useDatabaseAndTableList = () => {
  const { selectedDatabase } = useIssueLogic();
  const tableStore = useTableStore();

  const databaseList = computed(() => {
    if (selectedDatabase.value) return [selectedDatabase.value];
    return [];
  });

  watch(
    databaseList,
    (list) => {
      list.forEach((db) => tableStore.getOrFetchTableListByDatabaseId(db.id));
    },
    { immediate: true }
  );

  const tableList = computed(() => {
    return databaseList.value
      .map((item) => tableStore.getTableListByDatabaseId(item.id))
      .flat();
  });

  return { databaseList, tableList };
};

type LocalEditState = Pick<LocalState, "editing" | "editStatement">;

const useTempEditState = (state: LocalState) => {
  const { create, selectedTask, selectedStatement } = useIssueLogic();

  let stopWatching: (() => void) | null = null;

  const startWatching = () => {
    const tempEditStateMap = new Map<TaskId, LocalEditState>();

    // The issue page is polling the issue entity, making the reference obj
    // of `selectedTask` changes every time.
    // So we need to watch the id instead of the object ref.
    const selectedTaskId = computed(() => {
      if (create.value) return UNKNOWN_ID;
      return (selectedTask.value as Task).id;
    });

    const beforeTaskIdChange = (id: TaskId) => {
      // Save the temp edit state before switching task.
      tempEditStateMap.set(id, {
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

    const stopWatchBeforeChange = watch(
      selectedTaskId,
      (_, id) => {
        beforeTaskIdChange(id);
      },
      { flush: "pre" } // Listen to the event BEFORE selectedTaskId changes
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
</script>
