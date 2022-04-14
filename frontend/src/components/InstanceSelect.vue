<template>
  <select
    class="btn-select w-full disabled:cursor-not-allowed"
    :disabled="disabled"
    @change="
      (e: any) => {
        $emit('select-instance-id', parseInt(e.target.value));
      }
    "
  >
    <option disabled :selected="undefined === state.selectedId">
      {{ $t("instance.select") }}
    </option>
    <template v-for="(instance, index) in instanceList" :key="index">
      <option
        v-if="instance.rowStatus == 'NORMAL'"
        :key="index"
        :value="instance.id"
        :selected="instance.id == state.selectedId"
      >
        {{ instanceName(instance) }}
      </option>
      <option
        v-else-if="instance.id == state.selectedId"
        :value="instance.id"
        :selected="true"
      >
        {{ instanceName(instance) }}
      </option>
    </template>
  </select>
</template>

<script lang="ts">
import { useInstanceStore } from "@/store";
import { computed, defineComponent, reactive, watch } from "vue";
import { Instance } from "../types";

interface LocalState {
  selectedId?: number;
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
      selectedId: props.selectedId,
    });

    const instanceList = computed(() => {
      if (props.environmentId) {
        return instanceStore.getInstanceListByEnvironmentId(
          props.environmentId,
          ["NORMAL", "ARCHIVED"]
        );
      }
      return instanceStore.getInstanceList(["NORMAL", "ARCHIVED"]);
    });

    watch(
      () => props.selectedId,
      (cur) => {
        state.selectedId = cur;
      }
    );

    // The instance list might change if environmentId changes, and the previous selected id
    // might not exist in the new list. In such case, we need to invalidate the selection
    // and emit the event.
    watch(
      () => instanceList.value,
      (curList) => {
        if (
          state.selectedId &&
          !curList.find((instance: Instance) => instance.id == state.selectedId)
        ) {
          state.selectedId = undefined;
          emit("select-instance-id", undefined);
        }
      }
    );

    return {
      state,
      instanceList,
    };
  },
});
</script>
