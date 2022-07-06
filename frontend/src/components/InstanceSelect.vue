<template>
  <BBSelect
    :selected-item="state.selectedInstance"
    :item-list="instanceList"
    :disabled="disabled"
    :placeholder="$t('instance.select')"
    :show-prefix-item="true"
    @select-item="(instance) => $emit('select-instance-id', instance.id)"
  >
    <template #menuItem="{ item: instance }">
      {{ instanceName(instance) }}
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { useInstanceStore } from "@/store";
import { computed, defineComponent, reactive, watch } from "vue";
import { Instance } from "../types";

interface LocalState {
  selectedInstance?: Instance;
}

export default defineComponent({
  name: "InstanceSelect",
  components: {},
  props: {
    selectedId: {
      type: Number,
      default: undefined,
    },
    environmentId: {
      type: Number,
      default: undefined,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  emits: ["select-instance-id"],
  setup(props, { emit }) {
    const instanceStore = useInstanceStore();
    const state = reactive<LocalState>({
      selectedInstance: undefined,
    });

    const rawInstanceList = computed(() => {
      if (props.environmentId) {
        return instanceStore.getInstanceListByEnvironmentId(
          props.environmentId,
          ["NORMAL", "ARCHIVED"]
        );
      }
      return instanceStore.getInstanceList(["NORMAL", "ARCHIVED"]);
    });

    const instanceList = computed(() => {
      return rawInstanceList.value.filter((instance) => {
        if (instance.rowStatus === "NORMAL") {
          return true;
        }
        // instance.rowStatus === "ARCHIVED"
        if (instance.id === state.selectedInstance?.id) {
          return true;
        }
        return false;
      });
    });

    // The instance list might change if environmentId changes, and the previous selected id
    // might not exist in the new list. In such case, we need to invalidate the selection
    // and emit the event.
    const invalidateSelectionIfNeeded = () => {
      if (
        state.selectedInstance &&
        !instanceList.value.find(
          (item) => item.id == state.selectedInstance?.id
        )
      ) {
        state.selectedInstance = undefined;
        emit("select-instance-id", undefined);
      }
    };

    watch(
      [() => props.selectedId, instanceList],
      ([selectedId, list]) => {
        invalidateSelectionIfNeeded();
        state.selectedInstance = list.find(
          (instance) => instance.id === selectedId
        );
      },
      { immediate: true }
    );

    return {
      state,
      instanceList,
    };
  },
});
</script>
