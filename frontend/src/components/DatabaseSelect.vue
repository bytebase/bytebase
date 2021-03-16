<template>
  <select
    class="btn-select w-full"
    @change="
      (e) => {
        $emit('select-database-id', e.target.value);
      }
    "
  >
    <option
      v-for="(database, index) in databaseList"
      :key="index"
      :value="database.id"
      :selected="database.id == state.selectedId"
    >
      {{ database.name }}
    </option>
  </select>
</template>

<script lang="ts">
import { computed, reactive } from "vue";
import { useStore } from "vuex";
import { ALL_DATABASE_ID } from "../types";

interface LocalState {
  selectedId?: string;
}

export default {
  name: "DatabaseSelect",
  emits: ["select-database-id"],
  components: {},
  props: {
    selectedId: {
      type: String,
    },
    selectDefault: {
      default: true,
      type: Boolean,
    },
    instanceId: {
      required: true,
      type: String,
    },
  },
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
    });

    const databaseList = computed(() => {
      const list = store.getters["database/databaseListByInstanceId"](
        props.instanceId
      );
      const fullList = [
        {
          id: ALL_DATABASE_ID,
          name: "* (All databases)",
        },
      ];
      fullList.push(...list);
      return fullList;
    });

    if (!props.selectedId && props.selectDefault) {
      state.selectedId = ALL_DATABASE_ID;
      emit("select-database-id", state.selectedId);
    }

    return {
      state,
      databaseList,
    };
  },
};
</script>
