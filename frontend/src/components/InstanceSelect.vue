<template>
  <select
    class="btn-select w-full disabled:cursor-not-allowed"
    :disabled="disabled"
    @change="
      (e) => {
        $emit('select-instance-id', e.target.value);
      }
    "
  >
    <option disabled :selected="undefined === state.selectedId">
      Select instance
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
import { computed, reactive, watch } from "vue";
import { useStore } from "vuex";
import { Instance } from "../types";

interface LocalState {
  selectedId?: string;
}

export default {
  name: "InstanceSelect",
  emits: ["select-instance-id"],
  components: {},
  props: {
    selectedId: {
      type: String,
    },
    environmentId: {
      type: String,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
    });

    const instanceList = computed(() => {
      if (props.environmentId) {
        return store.getters[
          "instance/instanceListByEnvironmentId"
        ](props.environmentId, ["NORMAL", "ARCHIVED"]);
      }
      return store.getters["instance/instanceList"](["NORMAL", "ARCHIVED"]);
    });

    watch(
      () => props.selectedId,
      (cur, _) => {
        state.selectedId = cur;
      }
    );

    // The instance list might change if environmentId changes, and the previous selected id
    // might not exist in the new list. In such case, we need to invalidate the selection
    // and emit the event.
    watch(
      () => instanceList.value,
      (curList, _) => {
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
};
</script>
