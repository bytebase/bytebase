<template>
  <select
    class="btn-select disabled:cursor-not-allowed"
    :disabled="disabled"
    @change="
      (e) => {
        state.selectedId = e.target.value;
        $emit('select-database-id', e.target.value);
      }
    "
  >
    <option disabled :selected="undefined === state.selectedId">
      <template v-if="mode == 'INSTANCE' && !instanceId">
        Select instance first
      </template>
      <template v-else-if="mode == 'ENVIRONMENT' && !environmentId">
        Select environment first
      </template>
      <template v-else> Select database </template>
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
import { computed, reactive, watch, watchEffect, PropType } from "vue";
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
    mode: {
      required: true,
      type: String as PropType<"INSTANCE" | "ENVIRONMENT">,
    },
    environmentId: {
      type: String,
    },
    instanceId: {
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

    const prepareDatabaseList = () => {
      // TODO: need to revisit this, instead of fetching each time,
      // we maybe able to let the outside context to provide the database list
      // and we just do a get here.
      if (props.mode == "ENVIRONMENT" && props.environmentId) {
        store
          .dispatch(
            "database/fetchDatabaseListByEnvironmentId",
            props.environmentId
          )
          .catch((error) => {
            console.log(error);
          });
      } else if (props.mode == "INSTANCE" && props.instanceId) {
        store
          .dispatch("database/fetchDatabaseListByInstanceId", props.instanceId)
          .catch((error) => {
            console.log(error);
          });
      }
    };

    watchEffect(prepareDatabaseList);

    const databaseList = computed(() => {
      let list: Database[] = [];
      if (props.mode == "ENVIRONMENT" && props.environmentId) {
        list = store.getters["database/databaseListByEnvironmentId"](
          props.environmentId
        );
      } else if (props.mode == "INSTANCE" && props.instanceId) {
        list = store.getters["database/databaseListByInstanceId"](
          props.instanceId
        );
      }
      return list;
    });

    const invalidateSelectionIfNeeded = () => {
      if (
        state.selectedId &&
        !databaseList.value.find(
          (database: Database) => database.id == state.selectedId
        )
      ) {
        state.selectedId = undefined;
        emit("select-database-id", state.selectedId);
      }
    };

    // The database list might change if environmentId changes, and the previous selected id
    // might not exist in the new list. In such case, we need to invalidate the selection
    // and emit the event.
    watch(
      () => databaseList.value,
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
      databaseList,
    };
  },
};
</script>
