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
import { reactive, watch } from "vue";
import { EnvironmentSelect, DatabaseSelect } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { UNKNOWN_DATABASE_NAME, isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto/v1/common";

const props = defineProps<{
  projectName?: string;
  databaseName?: string;
  readonly?: boolean;
  loading?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:database-name", databaseName: string | undefined): void;
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
    const database = isValidDatabaseName(state.databaseName)
      ? databaseStore.getDatabaseByName(state.databaseName)
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
  async (name) => {
    const database = isValidDatabaseName(name)
      ? databaseStore.getDatabaseByName(name)
      : undefined;
    try {
      if (database) {
        state.databaseName = database.name;
        state.environmentName = database.effectiveEnvironment;
      }
    } catch {
      // do nothing.
    }
    emit("update:database-name", database?.name);
  }
);

const allowedEngineTypeList: Engine[] = [
  Engine.MYSQL,
  Engine.TIDB,
  Engine.POSTGRES,
  Engine.ORACLE,
];

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
