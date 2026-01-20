<template>
  <div class="divide-y">
    <div class="flex flex-col gap-y-4 pb-7">
      <div>
        <p class="text-lg font-medium leading-7 text-main">
          {{ $t("common.environment") }}
        </p>
        <EnvironmentSelect
          class="mt-1 max-w-md"
          :value="`${environmentNamePrefix}${environment.id}`"
          :disabled="!allowUpdateDatabase"
          :clearable="!getInstanceResource(database).environment"
          :render-suffix="
            (env: Environment) =>
              getInstanceResource(database).environment === env.name
                ? `(${$t('common.default')})`
                : ''
          "
          @update:value="handleSelectEnvironment($event as (string | undefined))"
        />
      </div>
    </div>
    <Labels
      :database="database"
      :allow-edit="allowUpdateDatabase"
      class="pt-7"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { cloneDeep } from "lodash-es";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { EnvironmentSelect } from "@/components/v2";
import {
  environmentNamePrefix,
  pushNotification,
  useDatabaseV1Store,
  useEnvironmentV1Store,
} from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { UpdateDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import type { Environment } from "@/types/v1/environment";
import {
  getDatabaseProject,
  getInstanceResource,
  hasProjectPermissionV2,
} from "@/utils";
import Labels from "./components/Labels.vue";

const props = defineProps<{
  database: Database;
}>();

const databaseStore = useDatabaseV1Store();
const envStore = useEnvironmentV1Store();
const { t } = useI18n();

const environment = computed(() => {
  return envStore.getEnvironmentByName(
    props.database.effectiveEnvironment ?? ""
  );
});

const allowUpdateDatabase = computed(() =>
  hasProjectPermissionV2(
    getDatabaseProject(props.database),
    "bb.databases.update"
  )
);

const handleSelectEnvironment = async (name: string | undefined) => {
  if (name === props.database.effectiveEnvironment) {
    return;
  }
  const databasePatch = cloneDeep(props.database);
  databasePatch.environment = name ?? "";

  await databaseStore.updateDatabase(
    create(UpdateDatabaseRequestSchema, {
      database: databasePatch,
      updateMask: create(FieldMaskSchema, { paths: ["environment"] }),
    })
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
</script>
