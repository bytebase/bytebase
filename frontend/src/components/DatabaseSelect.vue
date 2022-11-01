<template>
  <BBSelect
    :selected-item="state.selectedDatabase"
    :item-list="databaseList"
    :disabled="disabled"
    :placeholder="$t('database.select')"
    :show-prefix-item="true"
    @select-item="(database: Database) => $emit('select-database-id', database.id)"
  >
    <template #menuItem="{ item: database }">
      <slot
        v-if="customizeItem"
        name="customizeItem"
        :database="database"
      ></slot>
      <div v-else class="flex items-center">
        <span>{{ database.name }}</span>
      </div>
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { isNullOrUndefined } from "@/plugins/demo/utils";
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
  EngineType,
  DatabaseSyncStatus,
} from "../types";

interface LocalState {
  selectedId?: number;
  selectedDatabase?: Database;
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
      type: Number as PropType<EnvironmentId>,
      default: UNKNOWN_ID,
    },
    instanceId: {
      type: Number as PropType<InstanceId>,
      default: UNKNOWN_ID,
    },
    projectId: {
      type: Number as PropType<ProjectId>,
      default: UNKNOWN_ID,
    },
    engineType: {
      type: String as PropType<EngineType>,
      default: undefined,
    },
    syncStatus: {
      type: String as PropType<DatabaseSyncStatus>,
      default: undefined,
    },
    disabled: {
      type: Boolean,
      default: false,
    },
    customizeItem: {
      type: Boolean,
      default: false,
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

      if (!isNullOrUndefined(props.engineType)) {
        list = list.filter((database: Database) => {
          return database.instance.engine === props.engineType;
        });
      }

      if (!isNullOrUndefined(props.syncStatus)) {
        list = list.filter((database: Database) => {
          return database.syncStatus === props.syncStatus;
        });
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
      (selectedId) => {
        invalidateSelectionIfNeeded();
        state.selectedId = selectedId;
        state.selectedDatabase = databaseList.value.find(
          (database) => database.id === selectedId
        );
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
