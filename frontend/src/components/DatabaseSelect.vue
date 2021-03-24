<template>
  <select
    class="btn-select w-full"
    @change="
      (e) => {
        state.selectedId = e.target.value;
        console.log('active select', state.selectedId);
        $emit('select-database-id', e.target.value);
      }
    "
  >
    <option disabled :selected="undefined === state.selectedId">
      Select database
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
import { computed, reactive, watch, watchEffect } from "vue";
import { useStore } from "vuex";
import { Database } from "../types";

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
        !state.selectedId &&
        databaseList.value &&
        databaseList.value.length > 0
      ) {
        state.selectedId = databaseList.value[0].id;
        emit("select-database-id", state.selectedId);
      }
    };

    const prepareDatabaseListByEnvironment = () => {
      if (props.environmentId) {
        // TODO: need to revisit this, instead of fetching each time
        // we maybe able to let the outside context to provide the database list
        // and we just do a get here.
        store
          .dispatch(
            "database/fetchDatabaseListByEnvironmentId",
            props.environmentId
          )
          .catch((error) => {
            console.log(error);
          });
      }
    };

    watchEffect(prepareDatabaseListByEnvironment);

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

    // The database list might change if environmentId changes, and the previous selected id
    // might not exist in the new list. In such case, we need to invalidate the selection
    // and emit the event.
    watch(
      () => databaseList.value,
      (curList, _) => {
        if (
          state.selectedId &&
          !curList.find((database: Database) => database.id == state.selectedId)
        ) {
          state.selectedId = undefined;
          selectDefaultIfNeeded();
        }
      }
    );

    selectDefaultIfNeeded();

    return {
      state,
      databaseList,
    };
  },
};
</script>
