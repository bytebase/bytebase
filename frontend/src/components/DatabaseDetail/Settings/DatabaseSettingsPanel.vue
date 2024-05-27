<template>
  <div class="flex flex-col divide-y gap-y-7">
    <div class="space-y-4">
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
      <div
        v-if="
          supportClassificationFromCommentFeature(
            database.instanceEntity.engine
          )
        "
      >
        <p class="text-lg font-medium leading-7 text-main">
          {{ $t("database.classification.sync-from-comment") }}
        </p>
        <i18n-t
          class="textinfolabel"
          tag="div"
          keypath="database.classification.sync-from-comment-tip"
        >
          <template #format>
            <span class="font-semibold">{calssification id}-{comment}</span>
          </template>
        </i18n-t>
        <NSwitch
          class="mt-2"
          :value="!databaseMetadata.classificationFromConfig"
          :disabled="!allowUpdateDatabase"
          @update:value="onClassificationConfigChange"
        />
      </div>
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
import { NSwitch, useDialog } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { supportClassificationFromCommentFeature } from "@/components/ColumnDataTable/utils";
import { useDatabaseDetailContext } from "@/components/Database/context";
import { EnvironmentSelect } from "@/components/v2";
import {
  useDBSchemaV1Store,
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
const dbSchemaV1Store = useDBSchemaV1Store();
const { t } = useI18n();
const $dialog = useDialog();

const environment = computed(() => {
  return envStore.getEnvironmentByName(props.database.effectiveEnvironment);
});

const databaseMetadata = computed(() =>
  dbSchemaV1Store.getDatabaseMetadata(props.database.name)
);

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
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const onClassificationConfigChange = (on: boolean) => {
  const classificationFromConfig = !on;

  $dialog.warning({
    title: t("common.warning"),
    content: on
      ? t("database.classification.sync-from-comment-enable-warning")
      : t("database.classification.sync-from-comment-disable-warning"),
    style: "z-index: 100000",
    negativeText: t("common.cancel"),
    positiveText: t("common.confirm"),
    onPositiveClick: async () => {
      const pendingUpdateDatabaseConfig = cloneDeep(databaseMetadata.value);
      pendingUpdateDatabaseConfig.classificationFromConfig =
        classificationFromConfig;
      await dbSchemaV1Store.updateDatabaseSchemaConfigs(
        pendingUpdateDatabaseConfig
      );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    },
  });
};
</script>
