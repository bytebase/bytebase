<template>
  <BBSelect
    :selected-item="state.selectedInstance"
    :item-list="instanceList"
    :disabled="disabled"
    :placeholder="$t('instance.select')"
    :show-prefix-item="true"
    :error="!validate()"
    @select-item="(instance) => $emit('select-instance-id', instance.id)"
  >
    <template #menuItem="{ item: instance }">
      {{ instanceName(instance) }}
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { useInstanceStore } from "@/store";
import { computed, defineComponent, PropType, reactive, watch } from "vue";
import { IdType, Instance, UNKNOWN_ID } from "../types";

interface LocalState {
  selectedInstance?: Instance;
}

export default defineComponent({
  name: "InstanceSelect",
  components: {},
  props: {
    selectedId: {
      type: [String, Number] as PropType<IdType>,
      default: undefined,
    },
    environmentId: {
      type: [String, Number] as PropType<IdType>,
      default: undefined,
    },
    filter: {
      type: Function as PropType<(instance: Instance) => boolean>,
      default: undefined,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
    required: {
      type: Boolean,
      default: false,
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
      const list = rawInstanceList.value.filter((instance) => {
        if (instance.rowStatus === "NORMAL") {
          return true;
        }
        // instance.rowStatus === "ARCHIVED"
        if (instance.id === state.selectedInstance?.id) {
          return true;
        }
        return false;
      });

      if (!props.filter) {
        return list;
      }
      return list.filter(props.filter);
    });

    const validate = (): boolean => {
      if (!props.required) {
        return true;
      }
      return (
        !!state.selectedInstance && state.selectedInstance.id !== UNKNOWN_ID
      );
    };

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
      validate,
    };
  },
});
</script>
