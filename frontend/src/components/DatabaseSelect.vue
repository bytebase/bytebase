<template>
  <select
    class="btn-select disabled:cursor-not-allowed"
    :disabled="disabled"
    @change="onSelectChange"
  >
    <option disabled :selected="UNKNOWN_ID === state.selectedId">
      <template v-if="mode == 'INSTANCE' && instanceId == UNKNOWN_ID">
        {{ $t("db.select-instance-first") }}
      </template>
      <template
        v-else-if="mode == 'ENVIRONMENT' && environmentId == UNKNOWN_ID"
      >
        {{ $t("db.select-environment-first") }}
      </template>
      <template v-else> {{ $t("db.select") }} </template>
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
import { useCurrentUser, useDatabaseStore } from "@/store";
import {
  computed,
  reactive,
  watch,
  watchEffect,
  PropType,
  defineComponent,
} from "vue";
import {
  UNKNOWN_ID,
  Database,
  ProjectId,
  InstanceId,
  EnvironmentId,
} from "../types";

interface LocalState {
  selectedId?: number;
}

export default defineComponent({
  name: "DatabaseSelect",
  props: {
    selectedId: {
      required: true,
      type: Number,
    },
    mode: {
      required: true,
      type: String as PropType<"INSTANCE" | "ENVIRONMENT" | "USER">,
    },
    environmentId: {
      default: UNKNOWN_ID,
      type: Number as PropType<EnvironmentId>,
    },
    instanceId: {
      default: UNKNOWN_ID,
      type: Number as PropType<InstanceId>,
    },
    projectId: {
      default: UNKNOWN_ID,
      type: Number as PropType<ProjectId>,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  emits: ["select-database-id"],
  setup(props, { emit }) {
    const databaseStore = useDatabaseStore();
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
    });

    const currentUser = useCurrentUser();

    const prepareDatabaseList = () => {
      // TODO(tianzhou): Instead of fetching each time, we maybe able to let the outside context
      // to provide the database list and we just do a get here.
      if (props.mode == "ENVIRONMENT" && props.environmentId != UNKNOWN_ID) {
        databaseStore.fetchDatabaseListByEnvironmentId(props.environmentId);
      } else if (props.mode == "INSTANCE" && props.instanceId != UNKNOWN_ID) {
        databaseStore.fetchDatabaseListByInstanceId(props.instanceId);
      } else if (props.mode == "USER") {
        // We assume the database list for the current user should have already been fetched, so we won't do a fetch here.
      }
    };

    watchEffect(prepareDatabaseList);

    const databaseList = computed(() => {
      let list: Database[] = [];
      if (props.mode == "ENVIRONMENT" && props.environmentId != UNKNOWN_ID) {
        list = databaseStore.getDatabaseListByEnvironmentId(
          props.environmentId
        );
      } else if (props.mode == "INSTANCE" && props.instanceId != UNKNOWN_ID) {
        list = databaseStore.getDatabaseListByInstanceId(props.instanceId);
      } else if (props.mode == "USER") {
        list = databaseStore.getDatabaseListByPrincipalId(currentUser.value.id);
        if (
          props.environmentId != UNKNOWN_ID ||
          props.projectId != UNKNOWN_ID
        ) {
          list = list.filter((database: Database) => {
            return (
              (props.environmentId == UNKNOWN_ID ||
                database.instance.environment.id == props.environmentId) &&
              (props.projectId == UNKNOWN_ID ||
                database.project.id == props.projectId)
            );
          });
        }
      }
      return list;
    });

    const invalidateSelectionIfNeeded = () => {
      if (
        state.selectedId != UNKNOWN_ID &&
        !databaseList.value.find(
          (database: Database) => database.id == state.selectedId
        )
      ) {
        state.selectedId = UNKNOWN_ID;
        emit("select-database-id", state.selectedId);
      }
    };

    const onSelectChange = (e: Event) => {
      const id = parseInt((e.target as HTMLSelectElement).value, 10);
      state.selectedId = id;
      emit("select-database-id", id);
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
      (cur) => {
        state.selectedId = cur;
      }
    );

    return {
      UNKNOWN_ID,
      state,
      databaseList,
      onSelectChange,
    };
  },
});
</script>
