<template>
  <select
    class="btn-select disabled:cursor-not-allowed"
    :disabled="disabled"
    @change="
      (e) => {
        state.selectedID = e.target.value;
        $emit('select-database-id', parseInt(e.target.value));
      }
    "
  >
    <option disabled :selected="UNKNOWN_ID === state.selectedID">
      <template v-if="mode == 'INSTANCE' && instanceID == UNKNOWN_ID">
        Select instance first
      </template>
      <template
        v-else-if="mode == 'ENVIRONMENT' && environmentID == UNKNOWN_ID"
      >
        Select environment first
      </template>
      <template v-else> Select database </template>
    </option>
    <option
      v-for="(database, index) in databaseList"
      :key="index"
      :value="database.id"
      :selected="database.id == state.selectedID"
    >
      {{ database.name }}
    </option>
  </select>
</template>

<script lang="ts">
import { computed, reactive, watch, watchEffect, PropType } from "vue";
import { useStore } from "vuex";
import {
  UNKNOWN_ID,
  Database,
  Principal,
  ProjectID,
  InstanceID,
  EnvironmentID,
} from "../types";

interface LocalState {
  selectedID?: number;
}

export default {
  name: "DatabaseSelect",
  props: {
    selectedID: {
      required: true,
      type: Number,
    },
    mode: {
      required: true,
      type: String as PropType<"INSTANCE" | "ENVIRONMENT" | "USER">,
    },
    environmentID: {
      default: UNKNOWN_ID,
      type: Number as PropType<EnvironmentID>,
    },
    instanceID: {
      default: UNKNOWN_ID,
      type: Number as PropType<InstanceID>,
    },
    projectID: {
      default: UNKNOWN_ID,
      type: Number as PropType<ProjectID>,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  emits: ["select-database-id"],
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      selectedID: props.selectedID,
    });

    const currentUser = computed(
      (): Principal => store.getters["auth/currentUser"]()
    );

    const prepareDatabaseList = () => {
      // TODO(tianzhou): Instead of fetching each time, we maybe able to let the outside context
      // to provide the database list and we just do a get here.
      if (props.mode == "ENVIRONMENT" && props.environmentID != UNKNOWN_ID) {
        store.dispatch(
          "database/fetchDatabaseListByEnvironmentID",
          props.environmentID
        );
      } else if (props.mode == "INSTANCE" && props.instanceID != UNKNOWN_ID) {
        store.dispatch(
          "database/fetchDatabaseListByInstanceID",
          props.instanceID
        );
      } else if (props.mode == "USER") {
        // We assume the database list for the current user should have already been fetched, so we won't do a fetch here.
      }
    };

    watchEffect(prepareDatabaseList);

    const databaseList = computed(() => {
      let list: Database[] = [];
      if (props.mode == "ENVIRONMENT" && props.environmentID != UNKNOWN_ID) {
        list = store.getters["database/databaseListByEnvironmentID"](
          props.environmentID
        );
      } else if (props.mode == "INSTANCE" && props.instanceID != UNKNOWN_ID) {
        list = store.getters["database/databaseListByInstanceID"](
          props.instanceID
        );
      } else if (props.mode == "USER") {
        list = store.getters["database/databaseListByPrincipalID"](
          currentUser.value.id
        );
        if (
          props.environmentID != UNKNOWN_ID ||
          props.projectID != UNKNOWN_ID
        ) {
          list = list.filter((database: Database) => {
            return (
              (props.environmentID == UNKNOWN_ID ||
                database.instance.environment.id == props.environmentID) &&
              (props.projectID == UNKNOWN_ID ||
                database.project.id == props.projectID)
            );
          });
        }
      }
      return list;
    });

    const invalidateSelectionIfNeeded = () => {
      if (
        state.selectedID != UNKNOWN_ID &&
        !databaseList.value.find(
          (database: Database) => database.id == state.selectedID
        )
      ) {
        state.selectedID = UNKNOWN_ID;
        emit("select-database-id", state.selectedID);
      }
    };

    // The database list might change if environmentID changes, and the previous selected id
    // might not exist in the new list. In such case, we need to invalidate the selection
    // and emit the event.
    watch(
      () => databaseList.value,
      () => {
        invalidateSelectionIfNeeded();
      }
    );

    watch(
      () => props.selectedID,
      (cur) => {
        state.selectedID = cur;
      }
    );

    return {
      UNKNOWN_ID,
      state,
      databaseList,
    };
  },
};
</script>
