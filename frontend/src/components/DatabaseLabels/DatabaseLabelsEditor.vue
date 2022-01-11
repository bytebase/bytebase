<template>
  <div class="database-labels-editor flex items-center">
    <DatabaseLabels
      :labels="state.labels"
      :editable="allowEdit && state.mode === 'EDIT'"
    />
    <div
      v-if="state.mode === 'VIEW' && state.labels.length === 0"
      class="text-sm text-control-placeholder"
    >
      {{ $t("label.no-label") }}
    </div>
    <div
      v-if="allowEdit"
      class="buttons flex items-center gap-1 ml-1 text-control"
    >
      <template v-if="state.mode === 'VIEW'">
        <button class="icon-btn lite" @click="state.mode = 'EDIT'">
          <heroicons-outline:pencil class="w-4 h-4" />
        </button>
      </template>
      <template v-else>
        <button class="icon-btn text-error" @click="cancel">
          <heroicons-solid:x class="w-4 h-4" />
        </button>

        <NPopover trigger="hover" :disabled="!state.error">
          <template #trigger>
            <button
              class="icon-btn text-success"
              :class="{ disabled: !!state.error }"
              @click="save"
            >
              <heroicons-solid:check class="w-4 h-4" />
            </button>
          </template>

          <div class="text-red-600 whitespace-nowrap">
            {{ state.error }}
          </div>
        </NPopover>
      </template>
    </div>
  </div>
</template>

<script lang="ts">
import { cloneDeep } from "lodash-es";
import { defineComponent, PropType, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { DatabaseLabel } from "../../types";
import { validateLabels } from "../../utils";
import { NPopover } from "naive-ui";

type LocalState = {
  mode: "VIEW" | "EDIT";
  labels: DatabaseLabel[];
  error: string | undefined;
};

export default defineComponent({
  name: "DatabaseLabelsEditor",
  components: { NPopover },
  props: {
    labels: {
      type: Array as PropType<DatabaseLabel[]>,
      default: () => [],
    },
    allowEdit: {
      type: Boolean,
      default: true,
    },
  },
  emits: ["save"],
  setup(props, { emit }) {
    const { t } = useI18n();

    const state = reactive<LocalState>({
      mode: "VIEW",
      labels: cloneDeep(props.labels),
      error: undefined,
    });

    watch(
      () => props.labels,
      (labels) => {
        // state.labels are a local deep-copy of props.labels
        // <DatabaseLabels /> will mutate state.labels directly
        // when save button clicked, we emit a event to notify the parent
        //   component to dispatch a real save action
        state.labels = cloneDeep(labels);
        state.error = undefined;
      }
    );

    watch(
      () => state.labels,
      (labels) => {
        const error = validateLabels(labels);
        if (error) {
          state.error = t(error);
        } else {
          state.error = undefined;
        }
      },
      { deep: true }
    );

    const cancel = () => {
      state.mode = "VIEW";
      state.labels = cloneDeep(props.labels);
      state.error = undefined;
    };
    const save = () => {
      if (state.error) return;
      emit("save", state.labels);
      state.mode = "VIEW";
    };

    return {
      state,
      cancel,
      save,
    };
  },
});
</script>

<style scoped lang="postcss">
.icon-btn {
  @apply h-6 px-1 py-1 inline-flex items-center
    rounded bg-white border border-control-border
    hover:bg-control-bg-hover
    cursor-pointer;
}
.icon-btn.disabled {
  @apply cursor-not-allowed bg-control-bg;
}
.icon-btn.lite {
  @apply px-1 border-none hover:bg-control-bg-hover;
}
</style>
