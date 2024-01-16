<template>
  <div class="flex flex-col divide-y gap-y-7">
    <div>
      <p class="text-lg font-medium leading-7 text-main">
        {{ $t("common.environment") }}
      </p>
      <EnvironmentSelect
        class="mt-1 max-w-md"
        :environment="environment?.uid"
        :disabled="!allowUpdateDatabase"
        :default-environment-name="database.instanceEntity.environment"
        @update:environment="handleSelectEnvironmentUID"
      />
    </div>
    <Labels
      :database="database"
      :allow-edit="allowUpdateDatabase"
      class="pt-5"
    />
    <Secrets
      v-if="allowListSecrets"
      :database="database"
      :allow-edit="allowUpdateSecrets"
      :allow-delete="allowDeleteSecrets"
      class="pt-5"
    />
  </div>
</template>

<script setup lang="ts">
import { cloneDeep } from "lodash-es";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useDatabaseDetailContext } from "@/components/Database/context";
import { EnvironmentSelect } from "@/components/v2";
import {
  useDatabaseV1Store,
  useEnvironmentV1Store,
  pushNotification,
} from "@/store";
import { type ComposedDatabase } from "@/types";
import Labels from "./components/Labels.vue";
import Secrets from "./components/Secrets.vue";

const props = defineProps<{
  database: ComposedDatabase;
}>();

const databaseStore = useDatabaseV1Store();
const envStore = useEnvironmentV1Store();
const { t } = useI18n();

const environment = computed(() => {
  return envStore.getEnvironmentByName(props.database.effectiveEnvironment);
});

const {
  allowUpdateDatabase,
  allowListSecrets,
  allowUpdateSecrets,
  allowDeleteSecrets,
} = useDatabaseDetailContext();

const handleSelectEnvironmentUID = async (uid?: string) => {
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
  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("common.updated"),
  });
};
</script>
