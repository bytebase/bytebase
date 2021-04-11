<template>
  <select
    class="btn-select w-full disabled:cursor-not-allowed"
    :disabled="disabled"
    @change="
      (e) => {
        $emit('select-environment-id', e.target.value);
      }
    "
  >
    <option disabled :selected="undefined === state.selectedId">
      Select environment
    </option>
    <template v-for="(environment, index) in environmentList" :key="index">
      <option
        v-if="environment.rowStatus == 'NORMAL'"
        :value="environment.id"
        :selected="environment.id == state.selectedId"
      >
        {{ environment.name }}
      </option>
      <option
        v-else-if="environment.id == state.selectedId"
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
import { Environment, EnvironmentId } from "../types";

interface LocalState {
  selectedId?: string;
}

export default {
  name: "EnvironmentSelect",
  emits: ["select-environment-id"],
  components: {},
  props: {
    selectedId: {
      type: String,
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
      selectedId: props.selectedId,
    });

    const environmentList = computed(() => {
      return cloneDeep(
        store.getters["environment/environmentList"]()
      ).reverse();
    });

    if (environmentList.value && environmentList.value.length > 0) {
      if (
        !props.selectedId ||
        !environmentList.value.find(
          (item: Environment) => item.id == props.selectedId
        )
      ) {
        if (props.selectDefault) {
          state.selectedId = environmentList.value[0].id;
          emit("select-environment-id", state.selectedId);
        }
      }
    }

    const invalidateSelectionIfNeeded = () => {
      if (
        state.selectedId &&
        !environmentList.value.find(
          (item: Environment) => item.id == state.selectedId
        )
      ) {
        state.selectedId = undefined;
        emit("select-environment-id", state.selectedId);
      }
    };

    watch(
      () => environmentList.value,
      () => {
        invalidateSelectionIfNeeded();
      }
    );

    watch(
      () => props.selectedId,
      (cur, _) => {
        state.selectedId = cur;
      }
    );

    return {
      state,
      environmentList,
    };
  },
};
</script>
