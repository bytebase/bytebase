<template>
  <div class="contents">
    <EnvironmentSelect
      name="environment"
      :disabled="readonly || props.loading"
      :environment-name="state.environmentName"
      @update:environment-name="handleEnvironmentSelect"
    />
    <DatabaseSelect
      style="width: 100%"
      :placeholder="$t('schema-designer.select-database-placeholder')"
      :disabled="readonly || props.loading"
      :allowed-engine-type-list="allowedEngineTypeList"
      :environment-name="state.environmentName"
      :project-name="props.projectName"
      :database-name="state.databaseName ?? UNKNOWN_DATABASE_NAME"
      :fallback-option="false"
      @update:database-name="handleDatabaseSelect"
    />
  </div>
</template>

<script lang="ts" setup>
import { isNull, isUndefined } from "lodash-es";
import { reactive, watch } from "vue";
import { EnvironmentSelect, DatabaseSelect } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import {
  UNKNOWN_DATABASE_NAME,
  UNKNOWN_ID,
  isValidDatabaseName,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";

const props = defineProps<{
  projectName?: string;
  databaseId?: string;
  readonly?: boolean;
  loading?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:database-id", databaseId: string | undefined): void;
}>();

interface LocalState {
  environmentName?: string;
  databaseName?: string;
}

const state = reactive<LocalState>({});
const databaseStore = useDatabaseV1Store();

watch(
  () => props.projectName,
  () => {
    const database = isValidId(state.databaseName)
      ? databaseStore.getDatabaseByUID(state.databaseName)
      : undefined;
    if (!database || !props.projectName) {
      return;
    }
    if (database.project !== props.projectName) {
      state.environmentName = undefined;
      state.databaseName = undefined;
    }
  }
);

watch(
  () => state.databaseName,
  async (databaseId) => {
    const database = isValidId(state.databaseName)
      ? databaseStore.getDatabaseByUID(state.databaseName)
      : undefined;
    try {
      if (database) {
        state.databaseName = database.uid;
        state.environmentName = database.effectiveEnvironment;
      }
    } catch (error) {
      // do nothing.
    }
    emit("update:database-id", databaseId);
  }
);

const allowedEngineTypeList: Engine[] = [
  Engine.MYSQL,
  Engine.TIDB,
  Engine.POSTGRES,
  Engine.ORACLE,
];

const isValidId = (id: any): id is string => {
  if (isNull(id) || isUndefined(id) || String(id) === String(UNKNOWN_ID)) {
    return false;
  }
  return true;
};

const handleEnvironmentSelect = (name?: string) => {
  if (name !== state.environmentName) {
    state.environmentName = name;
    state.databaseName = undefined;
  }
};

const handleDatabaseSelect = (name?: string) => {
  if (isValidDatabaseName(name)) {
    const database = databaseStore.getDatabaseByName(name);
    if (!database) {
      return;
    }
    state.environmentName = database.effectiveEnvironment;
    state.databaseName = name;
  }
};
</script>

<style lang="postcss">
.bb-baseline-schema-select-option .n-base-select-option__content {
  @apply w-full;
}
</style>
