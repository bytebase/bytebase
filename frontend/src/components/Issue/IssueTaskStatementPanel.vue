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
        v-if="allowApplyStatementToOtherStages"
        :disabled="isEmpty(state.editStatement)"
        type="button"
        class="btn-small"
        @click.prevent="applyStatementToOtherStages(state.editStatement)"
      >
        {{ $t("issue.apply-to-other-stages") }}
      </button>
    </div>

    <div class="space-x-2 flex items-center">
      <template v-if="create">
        <label class="mt-0.5 inline-flex items-center gap-1">
          <input
            v-model="formatOnSave"
            type="checkbox"
            class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
          />
          <span class="textlabel">{{ $t("issue.format-on-save") }}</span>
        </label>
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
          <label class="mt-0.5 inline-flex items-center gap-1">
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
            class="mt-0.5 px-3 rounded-sm text-control hover:bg-control-bg-hover disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed text-sm leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
            @click.prevent="cancelEdit"
          >
            {{ $t("common.cancel") }}
          </button>
          <button
            v-if="state.editing"
            type="button"
            class="mt-0.5 px-3 border border-control-border rounded-sm text-control bg-control-bg hover:bg-control-bg-hover disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed text-sm leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
            :disabled="state.editStatement == statement"
            @click.prevent="saveEdit"
          >
            {{ $t("common.save") }}
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
    <EmbeddedMonacoEditor
      :value="state.editing ? state.editStatement : statement"
      :database-list="databaseList"
      :table-list="tableList"
      :readonly="!state.editing"
      :min-height="
        state.editing ? EDITOR_MIN_HEIGHT.EDITABLE : EDITOR_MIN_HEIGHT.READONLY
      "
      :max-height="EDITOR_MAX_HEIGHT"
      @change="onStatementChange"
    />
  </div>
</template>

<script lang="ts">
import { onMounted, reactive, watch, defineComponent, computed } from "vue";
import { useTableStore, useUIStateStore } from "@/store";
import { useIssueLogic } from "./logic";
import EmbeddedMonacoEditor from "../MonacoEditor/EmbeddedMonacoEditor.vue";

interface LocalState {
  editing: boolean;
  editStatement: string;
}

const EDITOR_MIN_HEIGHT = {
  READONLY: 0, // not limited to keep the UI compact
  EDITABLE: 120, // ~= 6 lines, a reasonable size to start writing SQL
};
const EDITOR_MAX_HEIGHT = 360; // ~= 20 lines, not too high for most screen size

export default defineComponent({
  name: "IssueTaskStatementPanel",
  components: {
    EmbeddedMonacoEditor,
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
      create,
      allowEditStatement,
      selectedStatement: statement,
      updateStatement,
      allowApplyStatementToOtherStages,
      applyStatementToOtherStages,
    } = useIssueLogic();

    const uiStateStore = useUIStateStore();

    const state = reactive<LocalState>({
      editing: false,
      editStatement: statement.value,
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
        state.editing = false;
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
      updateStatement(state.editStatement, () => {
        state.editing = false;
      });
    };

    const cancelEdit = () => {
      state.editStatement = statement.value;
      state.editing = false;
    };

    const onStatementChange = (value: string) => {
      state.editStatement = value;
      if (create.value) {
        // If we are creating an issue, emit the event immediately when every
        // time the user types.
        updateStatement(state.editStatement);
      }
    };

    return {
      create,
      allowEditStatement,
      statement,
      databaseList,
      tableList,
      updateStatement,
      allowApplyStatementToOtherStages,
      applyStatementToOtherStages,
      formatOnSave,
      state,
      beginEdit,
      saveEdit,
      cancelEdit,
      onStatementChange,
      EDITOR_MIN_HEIGHT,
      EDITOR_MAX_HEIGHT,
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
      list.forEach((db) => tableStore.fetchTableListByDatabaseId(db.id));
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
</script>
