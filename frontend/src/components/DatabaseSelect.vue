<template>
  <select
    class="btn-select w-full"
    @change="
      (e) => {
        $emit('select-database-id', e.target.value);
      }
    "
  >
    <option disabled :selected="undefined === state.selectedId">
      Not selected
    </option>
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
import {
  ALL_DATABASE_NAME,
  ALL_DATABASE_PLACEHOLDER_ID,
  Database,
} from "../types";

interface LocalState {
  selectedId?: string;
}

export default {
  name: "DatabaseSelect",
  emits: ["select-database-id"],
  components: {},
  props: {
    selectedId: {
      default: ALL_DATABASE_PLACEHOLDER_ID,
      type: String,
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
      return list;
    });

    if (
      props.selectedId == ALL_DATABASE_PLACEHOLDER_ID &&
      databaseList.value &&
      databaseList.value.length > 0
    ) {
      const allDatabase = databaseList.value.find(
        (item: Database) => item.name == ALL_DATABASE_NAME
      );
      state.selectedId = allDatabase.id;
      emit("select-database-id", state.selectedId);
    }

    return {
      state,
      databaseList,
    };
  },
};
</script>
