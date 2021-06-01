<template>
  <div class="flex justify-between">
    <div class="textlabel">{{ rollback ? "Rollback SQL" : "SQL" }}</div>
  </div>
  <label class="sr-only">SQL statement</label>
  <template v-if="state.editing">
    <textarea
      ref="editSqlTextArea"
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
      placeholder="Add SQL statement..."
      v-model="state.editSql"
      @input="
        (e) => {
          sizeToFit(e.target);
          // When creating the issue, we will emit the event on keystroke to update the in-memory state.
          if (create) {
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
import { Issue } from "../types";
import { sizeToFit } from "../utils";

interface LocalState {
  editing: boolean;
  editSql: string;
}

export default {
  name: "IssueSqlPanel",
  emits: ["update-sql"],
  props: {
    issue: {
      required: true,
      type: Object as PropType<Issue>,
    },
    create: {
      required: true,
      type: Boolean,
    },
    rollback: {
      required: true,
      type: Boolean,
    },
  },
  components: {},
  setup(props, { emit }) {
    const editSqlTextArea = ref();

    const effectiveSql = (issue: Issue): string => {
      return (props.rollback ? issue.rollbackSql : issue.sql) || "";
    };

    const state = reactive<LocalState>({
      editing: false,
      editSql: effectiveSql(props.issue),
    });

    const resizeTextAreaHandler = () => {
      if (state.editing) {
        sizeToFit(editSqlTextArea.value);
      }
    };

    onMounted(() => {
      window.addEventListener("resize", resizeTextAreaHandler);
      if (props.create) {
        state.editing = true;
        nextTick(() => {
          sizeToFit(editSqlTextArea.value);
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

    return {
      editSqlTextArea,
      state,
    };
  },
};
</script>
