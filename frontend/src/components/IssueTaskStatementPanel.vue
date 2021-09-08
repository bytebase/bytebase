<template>
  <div class="flex items-center space-x-4">
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
      <span v-if="!create && migrationType == 'BASELINE'" class="text-accent"
        >(This is a baseline migration and bytebase won't apply the SQL to the
        database, it will only record a baseline history)</span
      >
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
import { MigrationType } from "../types";

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
    rollback: {
      required: true,
      type: Boolean,
    },
    migrationType: {
      required: true,
      type: String as PropType<MigrationType>,
    },
    showApplyStatement: {
      required: true,
      type: Boolean,
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

    return {
      editStatementTextArea,
      state,
    };
  },
};
</script>
