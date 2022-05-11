<template>
  <div class="flex justify-between">
    <div class="flex space-x-4">
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
  <template v-if="state.editing">
    <textarea
      ref="editStatementTextArea"
      v-model="state.editStatement"
      class="whitespace-pre-wrap mt-2 w-full resize-none border-white focus:border-white outline-none"
      :class="state.editing ? 'focus:ring-control focus-visible:ring-2' : ''"
      :placeholder="$t('issue.add-sql-statement')"
      @input="
        (e) => {
          sizeToFit(e.target as HTMLTextAreaElement);
          // When creating the issue, we update the in-memory state.
          if (create) {
            updateStatement( state.editStatement);
          }
        }
      "
      @focus="
        (e) => {
          sizeToFit(e.target as HTMLTextAreaElement);
        }
      "
    ></textarea>
  </template>
  <!-- Margin value is to prevent flickering when switching between edit/non-edit mode -->
  <div v-else style="margin-left: 5px; margin-top: 8.5px; margin-bottom: 31px">
    <highlight-code-block
      v-if="statement"
      :code="statement"
      class="whitespace-pre-wrap"
    />
    <div v-else-if="create" class="ml-2 text-control-light">
      {{ $t("issue.add-sql-statement") }}
    </div>
    <div v-else class="ml-2 text-control-light">None</div>
  </div>
</template>

<script lang="ts">
import {
  nextTick,
  onMounted,
  onUnmounted,
  ref,
  reactive,
  watch,
  defineComponent,
  computed,
} from "vue";
import { sizeToFit } from "../../utils";
import { useUIStateStore } from "@/store";
import { useIssueContext } from "./context";

interface LocalState {
  editing: boolean;
  editStatement: string;
}

export default defineComponent({
  name: "IssueTaskStatementPanel",
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
    } = useIssueContext();

    const editStatementTextArea = ref();

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

    const resizeTextAreaHandler = () => {
      if (state.editing) {
        sizeToFit(editStatementTextArea.value);
      }
    };

    onMounted(() => {
      window.addEventListener("resize", resizeTextAreaHandler);
      if (create.value) {
        state.editing = true;
        nextTick(() => {
          sizeToFit(editStatementTextArea.value);
        });
      }
    });

    onUnmounted(() => {
      window.removeEventListener("resize", resizeTextAreaHandler);
    });

    // Reset the edit state after creating the issue.
    watch(create, (curNew, prevNew) => {
      if (!curNew && prevNew) {
        state.editing = false;
      }
    });

    watch(statement, (cur) => {
      state.editStatement = cur;
      nextTick(() => sizeToFit(editStatementTextArea.value));
    });

    const beginEdit = () => {
      state.editStatement = statement.value;
      state.editing = true;
      nextTick(() => {
        sizeToFit(editStatementTextArea.value);
        editStatementTextArea.value.focus();
      });
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

    return {
      create,
      allowEditStatement,
      statement,
      updateStatement,
      allowApplyStatementToOtherStages,
      applyStatementToOtherStages,
      editStatementTextArea,
      formatOnSave,
      state,
      beginEdit,
      saveEdit,
      cancelEdit,
    };
  },
});
</script>
