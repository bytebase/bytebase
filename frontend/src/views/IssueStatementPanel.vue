<template>
  <div class="flex justify-between">
    <div class="textlabel">
      {{ rollback ? "Rollback SQL" : "SQL" }}
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
      placeholder="Add SQL statement..."
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
  <!-- TODO: There is still flickering between edit/non-edit mode depending on the line height -->
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
  PropType,
  ref,
  reactive,
  watch,
} from "vue";
import { Issue } from "../types";
import { sizeToFit } from "../utils";

interface LocalState {
  editing: boolean;
  editStatement: string;
}

export default {
  name: "IssueStatementPanel",
  emits: ["update-statement"],
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
    const editStatementTextArea = ref();

    const effectiveStatement = (issue: Issue): string => {
      return (props.rollback ? issue.rollbackStatement : issue.statement) || "";
    };

    const state = reactive<LocalState>({
      editing: false,
      editStatement: effectiveStatement(props.issue),
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

    return {
      editStatementTextArea,
      state,
    };
  },
};
</script>
