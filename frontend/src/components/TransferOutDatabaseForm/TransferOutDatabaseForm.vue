<template>
  <DrawerContent :title="$t('database.transfer-database-to')">
    <div
      class="w-[calc(100vw-8rem)] lg:w-240 max-w-[calc(100vw-8rem)] h-full flex flex-col gap-y-2"
    >
      <div
        v-if="loading"
        class="absolute inset-0 z-10 bg-white/70 flex items-center justify-center"
      >
        <BBSpin />
      </div>
      <div v-else class="flex flex-col gap-y-4 pb-4">
        <span class="text-main text-base">
          {{ $t("database.transfer.select-databases") }}
          <span class="text-red-500">*</span>
        </span>
        <DatabaseV1Table
          :database-list="databaseList"
          :show-selection="true"
          v-model:selected-database-names="selectedDatabaseNameList"
        />
        <NDivider class="w-full" />
        <NRadioGroup v-model:value="transfer" class="gap-x-4">
          <NRadio value="project">
            <span class="text-main text-base">
              {{ $t("database.transfer.select-target-project") }}
            </span>
          </NRadio>
          <NRadio v-if="showUnassignOption" value="unassign">
            <span class="text-main text-base">
              {{ $t("database.unassign") }}
            </span>
          </NRadio>
        </NRadioGroup>
        <ProjectSelect
          v-if="transfer === 'project'"
          v-model:value="targetProjectName"
        />
      </div>
    </div>

    <template #footer>
      <div class="flex items-center justify-end gap-x-3">
        <NButton @click="$emit('dismiss')">{{ $t("common.cancel") }}</NButton>
        <NTooltip :disabled="allowTransfer">
          <template #trigger>
            <NButton
              type="primary"
              :disabled="!allowTransfer"
              tag="div"
              @click="doTransfer"
            >
              {{ $t("common.transfer") }}
            </NButton>
          </template>
          <ul>
            <li v-for="(error, i) in validationErrors" :key="i">
              {{ error }}
            </li>
          </ul>
        </NTooltip>
      </div>
    </template>
  </DrawerContent>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { NButton, NDivider, NRadio, NRadioGroup, NTooltip } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import { DrawerContent, ProjectSelect } from "@/components/v2";
import {
  pushNotification,
  useDatabaseV1Store,
  useProjectV1Store,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import { DEFAULT_PROJECT_NAME, isValidProjectName } from "@/types";
import {
  BatchUpdateDatabasesRequestSchema,
  DatabaseSchema$,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { autoProjectRoute } from "@/utils";
import DatabaseV1Table from "../v2/Model/DatabaseV1Table/DatabaseV1Table.vue";

const props = withDefaults(
  defineProps<{
    databaseList: ComposedDatabase[];
    selectedDatabaseNames?: string[];
    onSuccess?: (databases: ComposedDatabase[]) => void;
  }>(),
  {
    selectedDatabaseNames: () => [],
    onSuccess: (_: ComposedDatabase[]) => {},
  }
);

const emit = defineEmits<{
  (e: "dismiss"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const projectStore = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const loading = ref(false);
const transfer = ref<"project" | "unassign">("project");

const selectedDatabaseNameList = ref<string[]>(
  props.selectedDatabaseNames ?? []
);

watch(
  () => props.selectedDatabaseNames ?? [],
  (list) => (selectedDatabaseNameList.value = list),
  { immediate: true }
);

const selectedDatabaseList = computed(() => {
  return selectedDatabaseNameList.value.map((name) => {
    return databaseStore.getDatabaseByName(name);
  });
});

const showUnassignOption = computed(() => {
  return selectedDatabaseList.value.some(
    (db) => db.project !== DEFAULT_PROJECT_NAME
  );
});

const targetProjectName = ref<string>();

watch(
  () => transfer.value,
  (transfer) => {
    if (transfer === "unassign") {
      targetProjectName.value = DEFAULT_PROJECT_NAME;
    } else {
      targetProjectName.value = undefined;
    }
  }
);

const validationErrors = computed(() => {
  const errors: string[] = [];
  if (!targetProjectName.value) {
    errors.push(t("database.transfer.errors.select-target-project"));
  }
  if (selectedDatabaseNameList.value.length === 0) {
    errors.push(t("database.transfer.errors.select-at-least-one-database"));
  }
  return errors;
});

const allowTransfer = computed(() => {
  return validationErrors.value.length === 0;
});

const doTransfer = async () => {
  const name = targetProjectName.value;
  if (!name || !isValidProjectName(name)) {
    return;
  }

  const target = await projectStore.getOrFetchProjectByName(name);
  if (!target) {
    return;
  }

  const databaseList = selectedDatabaseList.value.filter(
    (db) => db.project !== target.name
  );

  try {
    loading.value = true;

    if (databaseList.length > 0) {
      await useDatabaseV1Store().batchUpdateDatabases(
        create(BatchUpdateDatabasesRequestSchema, {
          parent: "-",
          requests: databaseList.map((database) => {
            return create(UpdateDatabaseRequestSchema, {
              database: create(DatabaseSchema$, {
                name: database.name,
                project: target.name,
              }),
              updateMask: create(FieldMaskSchema, { paths: ["project"] }),
            });
          }),
        })
      );

      const displayDatabaseName =
        databaseList.length > 1
          ? `${databaseList.length} databases`
          : `'${databaseList[0].databaseName}'`;

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: `Successfully transferred ${displayDatabaseName} to project '${target.title}'.`,
      });

      router.push({
        ...autoProjectRoute(router, target),
      });
    }

    props.onSuccess(databaseList);
    emit("dismiss");
  } finally {
    loading.value = false;
  }
};
</script>
