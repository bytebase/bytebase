<template>
  <select
    class="btn-select disabled:cursor-not-allowed"
    :disabled="disabled"
    @change="
      (e) => {
        $emit('select-environment-id', parseInt(e.target.value));
      }
    "
  >
    <option disabled :selected="undefined === state.selectedID">
      Select environment
    </option>
    <template v-for="(environment, index) in environmentList" :key="index">
      <option
        v-if="environment.rowStatus == 'NORMAL'"
        :value="environment.id"
        :selected="environment.id == state.selectedID"
      >
        {{ environmentName(environment) }}
      </option>
      <option
        v-else-if="environment.id == state.selectedID"
        :value="environment.id"
        :selected="true"
      >
        {{ environmentName(environment) }}
      </option>
    </template>
  </select>
</template>

<script lang="ts">
import { computed, reactive, watch } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";
import { Environment, EnvironmentID } from "../types";

interface LocalState {
  selectedID?: Number;
}

export default {
  name: "EnvironmentSelect",
  emits: ["select-environment-id"],
  components: {},
  props: {
    selectedID: {
      type: Number,
    },
    selectDefault: {
      default: true,
      type: Boolean,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      selectedID: props.selectedID,
    });

    const environmentList = computed(() => {
      return cloneDeep(
        store.getters["environment/environmentList"](["NORMAL", "ARCHIVED"])
      ).reverse();
    });

    if (environmentList.value && environmentList.value.length > 0) {
      if (
        !props.selectedID ||
        !environmentList.value.find(
          (item: Environment) => item.id == props.selectedID
        )
      ) {
        if (props.selectDefault) {
          for (const environment of environmentList.value) {
            if (environment.rowStatus == "NORMAL") {
              state.selectedID = environment.id;
              emit("select-environment-id", state.selectedID);
              break;
            }
          }
        }
      }
    }

    const invalidateSelectionIfNeeded = () => {
      if (
        state.selectedID &&
        !environmentList.value.find(
          (item: Environment) => item.id == state.selectedID
        )
      ) {
        state.selectedID = undefined;
        emit("select-environment-id", state.selectedID);
      }
    };

    watch(
      () => environmentList.value,
      () => {
        invalidateSelectionIfNeeded();
      }
    );

    watch(
      () => props.selectedID,
      (cur, _) => {
        state.selectedID = cur;
      }
    );

    return {
      state,
      environmentList,
    };
  },
};
</script>
