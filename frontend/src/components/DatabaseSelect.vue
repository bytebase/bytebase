<template>
  <BBSelect
    :selected-item="selectedDatabase"
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
  InstanceId,
  EngineType,
  DatabaseSyncStatus,
  DEFAULT_PROJECT_ID,
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
      type: String as PropType<"ALL" | "INSTANCE" | "ENVIRONMENT" | "USER">,
    },
    environmentId: {
      type: String,
      default: String(UNKNOWN_ID),
    },
    instanceId: {
      type: Number as PropType<InstanceId>,
      default: UNKNOWN_ID,
    },
    projectId: {
      type: String,
      default: String(UNKNOWN_ID),
    },
    engineTypeList: {
      type: Array as PropType<EngineType[]>,
      default: undefined,
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
      if (props.mode === "ALL") {
        databaseStore.fetchDatabaseList();
      } else if (
        props.mode == "ENVIRONMENT" &&
        props.environmentId !== String(UNKNOWN_ID)
      ) {
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
      if (props.mode === "ALL") {
        list = databaseStore.getDatabaseList();
      } else if (
        props.mode == "ENVIRONMENT" &&
        props.environmentId !== String(UNKNOWN_ID)
      ) {
        list = databaseStore.getDatabaseListByEnvironmentId(
          props.environmentId
        );
      } else if (props.mode == "INSTANCE" && props.instanceId != UNKNOWN_ID) {
        list = databaseStore.getDatabaseListByInstanceId(props.instanceId);
      } else if (props.mode == "USER") {
        list = databaseStore.getDatabaseListByPrincipalId(currentUser.value.id);
      }

      if (!isNullOrUndefined(props.engineTypeList)) {
        list = list.filter((database: Database) => {
          return props.engineTypeList?.includes(database.instance.engine);
        });
      }

      if (!isNullOrUndefined(props.syncStatus)) {
        list = list.filter((database: Database) => {
          return database.syncStatus === props.syncStatus;
        });
      }

      if (
        props.environmentId !== String(UNKNOWN_ID) ||
        props.projectId !== String(UNKNOWN_ID)
      ) {
        list = list.filter((database: Database) => {
          return (
            (props.environmentId === String(UNKNOWN_ID) ||
              database.instance.environment.id == props.environmentId) &&
            (props.projectId === String(UNKNOWN_ID) ||
              database.project.id == props.projectId)
          );
        });
      }

      list = list.filter((database: Database) => {
        return database.project.id !== DEFAULT_PROJECT_ID;
      });

      return list;
    });

    const selectedDatabase = computed(() => {
      return databaseList.value.find(
        (database: Database) => database.id == state.selectedId
      );
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
      }
    );

    return {
      UNKNOWN_ID,
      state,
      databaseList,
      selectedDatabase,
      onSelectChange,
    };
  },
});
</script>
