<template>
  <div class="flex justify-between">
    <div class="textlabel">{{ rollback ? "Rollback SQL" : "SQL" }}</div>
    <div v-if="!$props.new" class="space-x-2">
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
        class="mt-0.5 px-3 rounded-sm text-control hover:bg-control-bg-hover disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed text-sm leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
        @click.prevent="cancelEdit"
      >
        Cancel
      </button>
      <button
        v-if="state.editing"
        type="button"
        class="mt-0.5 px-3 border border-control-border rounded-sm text-control bg-control-bg hover:bg-control-bg-hover disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed text-sm leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
        :disabled="!allowSave"
        @click.prevent="saveEdit"
      >
        Save
      </button>
    </div>
  </div>
  <label class="sr-only">SQL statement</label>
  <template v-if="state.editing">
    <textarea
      ref="editSqlTextArea"
      class="whitespace-pre-wrap mt-2 w-full resize-none border-white focus:border-white outline-none"
      :class="state.editing ? 'focus:ring-control focus-visible:ring-2' : ''"
      placeholder="Add SQL statement..."
      v-model="state.editSql"
      @input="
        (e) => {
          sizeToFit(e.target);
          // When creating the task, we will emit the event on keystroke to update the in-memory state.
          if ($props.new) {
            $emit('update-sql', state.editSql);
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
  <!-- TODO: There is still flickering between edit/non-edit mode depending on the line height -->
  <div v-else style="margin-left: 5px; margin-top: 8.5px; margin-bottom: 31px">
    <div v-if="state.editSql" v-highlight class="whitespace-pre-wrap">
      {{ state.editSql }}
    </div>
    <div v-else class="ml-2 text-control-light">Add SQL statement...</div>
  </div>
</template>

<script lang="ts">
import {
  nextTick,
  onMounted,
  onUnmounted,
  PropType,
  ref,
  reactive,
  watch,
  computed,
} from "vue";
import { Task } from "../types";
import { sizeToFit } from "../utils";
import command from "../store/modules/command";

interface LocalState {
  editing: boolean;
  editSql: string;
}

export default {
  name: "TaskSqlPanel",
  emits: ["update-sql"],
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
    new: {
      required: true,
      type: Boolean,
    },
    rollback: {
      required: true,
      type: Boolean,
    },
    allowEdit: {
      required: true,
      type: Boolean,
    },
  },
  components: {},
  setup(props, { emit }) {
    const editSqlTextArea = ref();

    const effectiveSql = (task: Task): string => {
      return (props.rollback ? task.rollbackSql : task.sql) || "";
    };

    const state = reactive<LocalState>({
      editing: false,
      editSql: effectiveSql(props.task),
    });

    const keyboardHandler = (e: KeyboardEvent) => {
      if (state.editing && editSqlTextArea.value === document.activeElement) {
        if (e.code == "Escape") {
          cancelEdit();
        } else if (e.code == "Enter" && e.metaKey) {
          if (props.rollback) {
            if (state.editSql != props.task.rollbackSql) {
              saveEdit();
            }
          } else {
            if (state.editSql != props.task.sql) {
              saveEdit();
            }
          }
        }
      }
    };

    const resizeTextAreaHandler = () => {
      if (state.editing) {
        sizeToFit(editSqlTextArea.value);
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", keyboardHandler);
      window.addEventListener("resize", resizeTextAreaHandler);
      if (props.new) {
        state.editing = true;
        nextTick(() => {
          sizeToFit(editSqlTextArea.value);
        });
      }
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", keyboardHandler);
      window.removeEventListener("resize", resizeTextAreaHandler);
    });

    // Reset the edit state after creating the task.
    watch(
      () => props.new,
      (curNew, prevNew) => {
        if (!curNew && prevNew) {
          state.editing = false;
        }
      }
    );

    const allowSave = computed(() => {
      return props.rollback
        ? state.editSql != props.task.rollbackSql
        : state.editSql != props.task.sql;
    });

    const beginEdit = () => {
      state.editSql = effectiveSql(props.task);
      state.editing = true;
      nextTick(() => {
        editSqlTextArea.value.focus();
      });
    };

    const saveEdit = () => {
      emit("update-sql", state.editSql, (updatedTask: Task) => {
        state.editSql = effectiveSql(updatedTask);
        state.editing = false;
      });
    };

    const cancelEdit = () => {
      state.editSql = effectiveSql(props.task);
      state.editing = false;
    };

    return {
      editSqlTextArea,
      state,
      allowSave,
      beginEdit,
      saveEdit,
      cancelEdit,
    };
  },
};
</script>
