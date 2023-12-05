<template>
  <div class="w-full mx-auto flex flex-col justify-start items-start gap-y-3">
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center shrink-0 text-sm">
        {{ $t("common.database") }}
      </span>
      <EnvironmentSelect
        class="!w-60 mr-4 shrink-0"
        name="environment"
        :disabled="props.readonly"
        :selected-id="state.environmentId"
        :select-default="false"
        :environment="state.environmentId"
        @update:environment="handleEnvironmentSelect"
      />
      <DatabaseSelect
        class="!w-128"
        :placeholder="$t('schema-designer.select-database-placeholder')"
        :disabled="props.readonly"
        :allowed-engine-type-list="allowedEngineTypeList"
        :environment="state.environmentId"
        :project="projectId"
        :database="state.databaseId"
        :fallback-option="false"
        @update:database="handleDatabaseSelect"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { debounce, isNull, isUndefined } from "lodash-es";
import { computed, reactive, watch } from "vue";
import { EnvironmentSelect, DatabaseSelect } from "@/components/v2";
import {
  useDBSchemaV1Store,
  useDatabaseV1Store,
  useEnvironmentV1Store,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  DatabaseMetadata,
  DatabaseMetadataView,
} from "@/types/proto/v1/database_service";

const props = defineProps<{
  projectId?: string;
  databaseId?: string;
  databaseMetadata?: DatabaseMetadata;
  readonly?: boolean;
}>();

const emit = defineEmits<{
  (event: "update", state: LocalState): void;
}>();

interface LocalState {
  environmentId?: string;
  databaseId?: string;
  databaseMetadata?: DatabaseMetadata;
}

const state = reactive<LocalState>({});
const databaseStore = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const environmentStore = useEnvironmentV1Store();

const database = computed(() => {
  const databaseId = state.databaseId;
  if (!isValidId(databaseId)) {
    return;
  }
  return databaseStore.getDatabaseByUID(databaseId);
});

watch(
  () => props.databaseId,
  async (databaseId) => {
    state.databaseId = databaseId ?? "";
    if (databaseId) {
      try {
        const database = await databaseStore.getOrFetchDatabaseByUID(
          databaseId
        );
        state.databaseId = database.uid;
        state.environmentId = database.effectiveEnvironmentEntity.uid;
        state.databaseMetadata = props.databaseMetadata;
      } catch (error) {
        // do nothing.
      }
    } else {
      state.databaseMetadata = undefined;
    }
  }
);

watch(
  () => props.projectId,
  () => {
    if (!database.value || !props.projectId) {
      return;
    }
    if (database.value.projectEntity.uid !== props.projectId) {
      state.environmentId = "";
      state.databaseId = "";
    }
  }
);

watch(
  () => state.databaseId,
  async (databaseId) => {
    if (!database.value || !databaseId) {
      state.databaseMetadata = undefined;
      return;
    }

    const databaseMetadata = await dbSchemaStore.getOrFetchDatabaseMetadata({
      database: database.value.name,
      skipCache: true,
      view: DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL,
    });
    state.databaseMetadata = databaseMetadata;
  }
);

const allowedEngineTypeList: Engine[] = [
  Engine.MYSQL,
  Engine.TIDB,
  Engine.POSTGRES,
];

const isValidId = (id: any): id is string => {
  if (isNull(id) || isUndefined(id) || String(id) === String(UNKNOWN_ID)) {
    return false;
  }
  return true;
};

const handleEnvironmentSelect = (environmentId?: string) => {
  if (environmentId !== state.environmentId) {
    state.databaseId = "";
  }
  state.environmentId = environmentId ?? "";
};

const handleDatabaseSelect = (databaseId?: string) => {
  if (isValidId(databaseId)) {
    const database = databaseStore.getDatabaseByUID(databaseId);
    if (!database) {
      return;
    }

    const environment = environmentStore.getEnvironmentByName(
      database.effectiveEnvironment
    );
    state.environmentId = environment?.uid ?? "";
    state.databaseId = databaseId;
  }
};

const debouncedUpdate = debounce((state: LocalState) => {
  emit("update", { ...state });
}, 200);

watch(() => state, debouncedUpdate, { deep: true });
</script>
<style lang="postcss">
.bb-baseline-schema-select-option .n-base-select-option__content {
  @apply w-full;
}
</style>
