<template>
  <BBSelect
    :selected-item="selectedDatabase"
    :item-list="databaseList"
    :disabled="disabled"
    :placeholder="placeholder || $t('database.select')"
    :show-prefix-item="true"
    @select-item="(database: ComposedDatabase) => $emit('select-database-id', database.uid)"
  >
    <template #menuItem="{ item: database }: { item: ComposedDatabase }">
      <slot
        v-if="customizeItem"
        name="customizeItem"
        :database="database"
      ></slot>
      <div v-else class="flex items-center">
        <span>{{ database.databaseName }}</span>
      </div>
    </template>
  </BBSelect>
</template>

<script lang="ts">
import {
  computed,
  reactive,
  watch,
  watchEffect,
  PropType,
  defineComponent,
} from "vue";
import { isNullOrUndefined } from "@/plugins/demo/utils";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
} from "@/store";
import { Engine, State } from "@/types/proto/v1/common";
import {
  UNKNOWN_ID,
  ComposedDatabase,
  DEFAULT_PROJECT_V1_NAME,
} from "../types";

interface LocalState {
  selectedId?: string;
}

export default defineComponent({
  name: "DatabaseSelect",
  props: {
    selectedId: {
      required: true,
      type: String,
    },
    mode: {
      required: true,
      type: String as PropType<"ALL" | "INSTANCE" | "ENVIRONMENT" | "USER">,
    },
    placeholder: {
      type: String,
      default: "",
    },
    environmentId: {
      type: String,
      default: String(UNKNOWN_ID),
    },
    instanceId: {
      type: String,
      default: String(UNKNOWN_ID),
    },
    projectId: {
      type: String,
      default: String(UNKNOWN_ID),
    },
    engineTypeList: {
      type: Array as PropType<Engine[]>,
      default: undefined,
    },
    engineType: {
      type: Number as PropType<Engine>,
      default: undefined,
    },
    syncState: {
      type: Number as PropType<State>,
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
    const databaseStore = useDatabaseV1Store();
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
    });

    const currentUserV1 = useCurrentUserV1();

    const prepareDatabaseList = async () => {
      // TODO(tianzhou): Instead of fetching each time, we maybe able to let the outside context
      // to provide the database list and we just do a get here.
      if (props.mode === "ALL") {
        databaseStore.fetchDatabaseList({
          parent: "instances/-",
        });
      } else if (
        props.mode == "ENVIRONMENT" &&
        props.environmentId !== String(UNKNOWN_ID)
      ) {
        // TODO: we do not support filtering database list by environment in v1
        // API so we need to fetch them all
        databaseStore.fetchDatabaseList({
          parent: "instances/-",
        });
      } else if (
        props.mode == "INSTANCE" &&
        props.instanceId !== String(UNKNOWN_ID)
      ) {
        const instance = await useInstanceV1Store().getOrFetchInstanceByUID(
          props.instanceId
        );
        databaseStore.fetchDatabaseList({
          parent: instance.name,
        });
      } else if (props.mode == "USER") {
        // We assume the database list for the current user should have already been fetched, so we won't do a fetch here.
      }
    };

    watchEffect(prepareDatabaseList);

    const databaseList = computed(() => {
      let list: ComposedDatabase[] = [];
      if (props.mode === "ALL") {
        list = [...databaseStore.databaseList];
      } else if (
        props.mode === "ENVIRONMENT" &&
        props.environmentId !== String(UNKNOWN_ID)
      ) {
        const environment = useEnvironmentV1Store().getEnvironmentByUID(
          props.environmentId
        );
        list = databaseStore.databaseListByEnvironment(environment.name);
      } else if (
        props.mode === "INSTANCE" &&
        props.instanceId !== String(UNKNOWN_ID)
      ) {
        const instance = useInstanceV1Store().getInstanceByUID(
          props.instanceId
        );
        list = databaseStore.databaseListByInstance(instance.name);
      } else if (props.mode == "USER") {
        list = databaseStore.databaseListByUser(currentUserV1.value);
      }

      if (!isNullOrUndefined(props.engineTypeList)) {
        list = list.filter((database) => {
          return props.engineTypeList?.includes(database.instanceEntity.engine);
        });
      }

      if (!isNullOrUndefined(props.syncState)) {
        list = list.filter((database) => {
          return database.syncState === props.syncState;
        });
      }

      if (props.environmentId !== String(UNKNOWN_ID)) {
        list = list.filter(
          (db) =>
            db.instanceEntity.environmentEntity.uid === props.environmentId
        );
      }
      if (props.instanceId !== String(UNKNOWN_ID)) {
        list = list.filter((db) => db.instanceEntity.uid === props.instanceId);
      }
      if (props.projectId !== String(UNKNOWN_ID)) {
        list = list.filter((db) => db.projectEntity.uid === props.projectId);
      }

      list = list.filter((database) => {
        return database.project !== DEFAULT_PROJECT_V1_NAME;
      });

      return list;
    });

    const selectedDatabase = computed(() => {
      return databaseList.value.find(
        (database) => database.uid == state.selectedId
      );
    });

    const invalidateSelectionIfNeeded = () => {
      if (
        state.selectedId !== String(UNKNOWN_ID) &&
        !databaseList.value.find((database) => database.uid == state.selectedId)
      ) {
        state.selectedId = String(UNKNOWN_ID);
        emit("select-database-id", state.selectedId);
      }
    };

    const onSelectChange = (e: Event) => {
      const id = (e.target as HTMLSelectElement).value;
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
