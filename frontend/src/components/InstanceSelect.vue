<template>
  <select
    class="btn-select w-full"
    @change="
      (e) => {
        $emit('select-instance-id', e.target.value);
      }
    "
  >
    <option disabled :selected="undefined === state.selectedId">
      Select instance
    </option>
    <option
      v-for="(instance, index) in instanceList"
      :key="index"
      :value="instance.id"
      :selected="instance.id == state.selectedId"
    >
      {{ instance.name }}
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
  name: "InstanceSelect",
  emits: ["select-instance-id"],
  components: {},
  props: {
    selectedId: {
      type: String,
    },
  },
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
    });

    const instanceList = computed(() => {
      return store.getters["instance/instanceList"]();
    });

    return {
      state,
      instanceList,
    };
  },
};
</script>
