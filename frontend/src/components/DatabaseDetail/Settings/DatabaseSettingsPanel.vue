<template>
  <div class="flex flex-col divide-y gap-y-7">
    <div class="space-y-4">
      <div>
        <p class="text-lg font-medium leading-7 text-main">
          {{ $t("common.environment") }}
        </p>
        <EnvironmentSelect
          class="mt-1 max-w-md"
          :environment-name="`${environmentNamePrefix}${environment.id}`"
          :disabled="!allowUpdateDatabase"
          :render-suffix="
            (env: string) =>
              database.instanceResource.environment === env
                ? `(${$t('common.default')})`
                : ''
          "
          @update:environment-name="handleSelectEnvironment"
        />
      </div>
    </div>
    <Labels
      :database="database"
      :allow-edit="allowUpdateDatabase"
      class="pt-5"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { cloneDeep } from "lodash-es";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useDatabaseDetailContext } from "@/components/Database/context";
import { EnvironmentSelect } from "@/components/v2";
import {
  useDatabaseV1Store,
  useEnvironmentV1Store,
  pushNotification,
  environmentNamePrefix,
} from "@/store";
import { type ComposedDatabase } from "@/types";
import { UpdateDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import Labels from "./components/Labels.vue";

const props = defineProps<{
  database: ComposedDatabase;
}>();

const databaseStore = useDatabaseV1Store();
const envStore = useEnvironmentV1Store();
const { t } = useI18n();

const environment = computed(() => {
  return envStore.getEnvironmentByName(
    props.database.effectiveEnvironment ?? ""
  );
});

const { allowUpdateDatabase } = useDatabaseDetailContext();

const handleSelectEnvironment = async (name: string | undefined) => {
  if (!name || name === props.database.effectiveEnvironment) {
    return;
  }
  const databasePatch = cloneDeep(props.database);
  databasePatch.environment = name;

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
