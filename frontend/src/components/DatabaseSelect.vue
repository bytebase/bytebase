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
import { computed, reactive, watchEffect } from "vue";
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
      type: String,
    },
    environmentId: {
      type: String,
    },
    instanceId: {
      type: String,
    },
  },
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
    });

    const selectDefaultIfNeeded = () => {
      if (
        !props.selectedId &&
        databaseList.value &&
        databaseList.value.length > 0
      ) {
        state.selectedId = databaseList.value[0].id;
        emit("select-database-id", state.selectedId);
      }
    };

    const prepareDatabaseListByEnvironment = () => {
      if (props.environmentId) {
        store
          .dispatch(
            "database/fetchDatabaseListByEnvironmentId",
            props.environmentId
          )
          .then(() => {
            selectDefaultIfNeeded();
          })
          .catch((error) => {
            console.log(error);
          });
      }
    };

    if (props.environmentId) {
      watchEffect(prepareDatabaseListByEnvironment);
    }

    const databaseList = computed(() => {
      if (props.environmentId) {
        return store.getters["database/databaseListByEnvironmentId"](
          props.environmentId
        );
      } else {
        return store.getters["database/databaseListByInstanceId"](
          props.instanceId
        );
      }
    });

    selectDefaultIfNeeded();

    return {
      state,
      databaseList,
    };
  },
};
</script>
