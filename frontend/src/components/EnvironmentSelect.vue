<template>
  <select
    class="btn-select w-full"
    @change="
      (e) => {
        $emit('select-environment-id', e.target.value);
      }
    "
  >
    <option disabled :selected="undefined === state.selectedId">
      Select environment
    </option>
    <option
      v-for="(environment, index) in environmentList"
      :key="index"
      :value="environment.id"
      :selected="environment.id == state.selectedId"
    >
      {{ environment.name }}
    </option>
  </select>
</template>

<script lang="ts">
import { computed, reactive } from "vue";
import { useStore } from "vuex";

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
  },
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
    });

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"]();
    });

    if (
      !props.selectedId &&
      props.selectDefault &&
      environmentList.value &&
      environmentList.value.length > 0
    ) {
      state.selectedId = environmentList.value[0].id;
      emit("select-environment-id", state.selectedId);
    }

    return {
      state,
      environmentList,
    };
  },
};
</script>
