<template>
  <select
    class="btn-select w-full disabled:cursor-not-allowed"
    :disabled="disabled"
    @change="
      (e) => {
        $emit('select-instance-id', parseInt(e.target.value));
      }
    "
  >
    <option disabled :selected="undefined === state.selectedID">
      Select instance
    </option>
    <template v-for="(instance, index) in instanceList" :key="index">
      <option
        v-if="instance.rowStatus == 'NORMAL'"
        :key="index"
        :value="instance.id"
        :selected="instance.id == state.selectedID"
      >
        {{ instanceName(instance) }}
      </option>
      <option
        v-else-if="instance.id == state.selectedID"
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
  selectedID?: number;
}

export default {
  name: "InstanceSelect",
  components: {},
  props: {
    selectedID: {
      type: Number,
    },
    environmentID: {
      type: Number,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  emits: ["select-instance-id"],
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      selectedID: props.selectedID,
    });

    const instanceList = computed(() => {
      if (props.environmentID) {
        return store.getters["instance/instanceListByEnvironmentID"](
          props.environmentID,
          ["NORMAL", "ARCHIVED"]
        );
      }
      return store.getters["instance/instanceList"](["NORMAL", "ARCHIVED"]);
    });

    watch(
      () => props.selectedID,
      (cur) => {
        state.selectedID = cur;
      }
    );

    // The instance list might change if environmentID changes, and the previous selected id
    // might not exist in the new list. In such case, we need to invalidate the selection
    // and emit the event.
    watch(
      () => instanceList.value,
      (curList) => {
        if (
          state.selectedID &&
          !curList.find((instance: Instance) => instance.id == state.selectedID)
        ) {
          state.selectedID = undefined;
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
