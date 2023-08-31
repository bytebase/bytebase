<template>
  <div class="flex flex-col">
    <div class="sm:col-span-2 sm:col-start-1 border-b mb-7 pb-7">
      <p class="text-lg font-medium leading-7 text-main">
        {{ $t("common.environment") }}
      </p>
      <EnvironmentSelect
        id="environment"
        class="mt-1 max-w-md"
        name="environment"
        :disabled="!allowEdit"
        :selected-id="environment?.uid"
        :default-environment="database.instanceEntity.environment"
        @select-environment-id="handleSelectEnvironmentUID"
      />
    </div>
    <Secrets :database="database" />
  </div>
</template>

<script setup lang="ts">
import { cloneDeep } from "lodash-es";
import { computed } from "vue";
import { useDatabaseV1Store, useEnvironmentV1Store } from "@/store";
import { type ComposedDatabase } from "@/types";
import Secrets from "./components/Secrets.vue";

const props = defineProps<{
  database: ComposedDatabase;
  allowEdit: boolean;
}>();

const databaseStore = useDatabaseV1Store();
const envStore = useEnvironmentV1Store();

const environment = computed(() => {
  return envStore.getEnvironmentByName(props.database.effectiveEnvironment);
});

const handleSelectEnvironmentUID = async (uid: number | string) => {
  const environment = envStore.getEnvironmentByUID(String(uid));
  if (environment.name === props.database.effectiveEnvironment) {
    return;
  }
  const databasePatch = cloneDeep(props.database);
  databasePatch.environment = environment.name;
  await databaseStore.updateDatabase({
    database: databasePatch,
    updateMask: ["environment"],
  });
};
</script>
