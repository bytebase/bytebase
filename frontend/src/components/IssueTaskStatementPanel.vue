<template>
  <div class="flex justify-between">
    <div class="flex space-x-4">
      <div
        class="text-sm font-medium"
        :class="
          !rollback && isEmpty(state.editStatement)
            ? 'text-red-600'
            : 'text-control'
        "
      >
        {{ rollback ? "Rollback SQL" : "SQL" }}
        <span v-if="create && !rollback" class="text-red-600">*</span>
        <span class="text-accent" v-if="sqlHint && !rollback">{{
          `(${sqlHint})`
        }}</span>
      </div>
      <button
        v-if="showApplyStatement"
        :disabled="isEmpty(state.editStatement)"
        type="button"
        class="btn-small"
        @click.prevent="
          $emit('apply-statement-to-other-stages', state.editStatement)
        "
      >
        Apply to other stages
      </button>
    </div>
    <div v-if="!create" class="space-x-2">
      <button
        v-if="allowEdit && !state.editing"
        type="button"
        class="btn-icon"
        @click.prevent="beginEdit"
      >
        <!-- Heroicon name: solid/pencil -->
        <!-- Use h-5 to avoid flickering when show/hide icon -->
        <svg
          class="h-5 w-5"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
        >
          <path
            d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z"
          />
        </svg>
      </button>
      <!-- mt-0.5 is to prevent jiggling betweening switching edit/none-edit -->
      <button
        v-if="state.editing"
        type="button"
        class="
          mt-0.5
          px-3
          rounded-sm
          text-control
          hover:bg-control-bg-hover
          disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed
          text-sm
          leading-5
          font-normal
          focus:ring-control focus:outline-none
          focus-visible:ring-2
          focus:ring-offset-2
        "
        @click.prevent="cancelEdit"
      >
        Cancel
      </button>
      <button
        v-if="state.editing"
        type="button"
        class="
          mt-0.5
          px-3
          border border-control-border
          rounded-sm
          text-control
          bg-control-bg
          hover:bg-control-bg-hover
          disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed
          text-sm
          leading-5
          font-normal
          focus:ring-control focus:outline-none
          focus-visible:ring-2
          focus:ring-offset-2
        "
        :disabled="state.editStatement == statement"
        @click.prevent="saveEdit"
      >
        Save
      </button>
    </div>
  </div>
  <label class="sr-only">SQL statement</label>
  <template v-if="state.editing">
    <textarea
      ref="editStatementTextArea"
      class="
        whitespace-pre-wrap
        mt-2
        w-full
        resize-none
        border-white
        focus:border-white
        outline-none
      "
      :class="state.editing ? 'focus:ring-control focus-visible:ring-2' : ''"
      :placeholder="
        create && rollback
          ? 'Add SQL statement...'
          : '(Required) Add SQL statement...'
      "
      v-model="state.editStatement"
      @input="
        (e) => {
          sizeToFit(e.target);
          // When creating the issue, we will emit the event on keystroke to update the in-memory state.
          if (create) {
            $emit('update-statement', state.editStatement);
          }
        }
      "
      @focus="
        (e) => {
          sizeToFit(e.target);
        }
      "
    ></textarea>
  </template>
  <!-- Margin value is to prevent flickering when switching between edit/non-edit mode -->
  <div v-else style="margin-left: 5px; margin-top: 8.5px; margin-bottom: 31px">
    <div v-if="state.editStatement" v-highlight class="whitespace-pre-wrap">
      {{ state.editStatement }}
    </div>
    <div v-else-if="state.create" class="ml-2 text-control-light">
      {{ rollback ? "Add rollback SQL statement..." : "Add SQL statement..." }}
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
  PropType,
} from "vue";
import { sizeToFit } from "../utils";
import { MigrationType, Task, TaskCreate } from "../types";

interface LocalState {
  editing: boolean;
  editStatement: string;
}

export default {
  name: "IssueTaskStatementPanel",
  emits: ["update-statement", "apply-statement-to-other-stages"],
  props: {
    statement: {
      required: true,
      type: String,
    },
    create: {
      required: true,
      type: Boolean,
    },
    allowEdit: {
      required: true,
      type: Boolean,
    },
    rollback: {
      required: true,
      type: Boolean,
    },
    showApplyStatement: {
      required: true,
      type: Boolean,
    },
    sqlHint: {
      required: false,
      type: String,
    },
  },
  components: {},
  setup(props, { emit }) {
    const editStatementTextArea = ref();

    const state = reactive<LocalState>({
      editing: false,
      editStatement: props.statement,
    });

    const resizeTextAreaHandler = () => {
      if (state.editing) {
        sizeToFit(editStatementTextArea.value);
      }
    };

    onMounted(() => {
      window.addEventListener("resize", resizeTextAreaHandler);
      if (props.create) {
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
    watch(
      () => props.create,
      (curNew, prevNew) => {
        if (!curNew && prevNew) {
          state.editing = false;
        }
      }
    );

    watch(
      () => props.statement,
      (cur, _) => {
        state.editStatement = cur;
      }
    );

    const beginEdit = () => {
      state.editStatement = props.statement;
      state.editing = true;
      nextTick(() => {
        editStatementTextArea.value.focus();
      });
    };

    const saveEdit = () => {
      emit("update-statement", state.editStatement, () => {
        state.editing = false;
      });
    };

    const cancelEdit = () => {
      state.editStatement = props.statement;
      state.editing = false;
    };

    return {
      editStatementTextArea,
      state,
      beginEdit,
      saveEdit,
      cancelEdit,
    };
  },
};
</script>
