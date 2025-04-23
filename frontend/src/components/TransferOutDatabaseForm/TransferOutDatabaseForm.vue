<template>
  <DrawerContent :title="$t('database.transfer-database-to')">
    <div
      class="w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)] h-full flex flex-col gap-y-2"
    >
      <div
        v-if="loading"
        class="absolute inset-0 z-10 bg-white/70 flex items-center justify-center"
      >
        <BBSpin />
      </div>
      <div v-else class="space-y-4 pb-4">
        <div class="space-y-4">
          <span class="text-main text-base">
            {{ $t("database.transfer.select-databases") }}
            <span class="text-red-500">*</span>
          </span>
          <DatabaseV1Table
            :database-list="databaseList"
            :show-selection="true"
            v-model:selected-database-names="selectedDatabaseNameList"
          />
        </div>
        <NDivider class="w-full py-2" />
        <NRadioGroup v-model:value="transfer" class="space-x-4">
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
          v-model:project-name="targetProjectName"
          :default-select-first="true"
          :allowed-project-role-list="[PresetRoleType.PROJECT_OWNER]"
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
import { NButton, NTooltip, NDivider, NRadioGroup, NRadio } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import { ProjectSelect, DrawerContent } from "@/components/v2";
import {
  pushNotification,
  useAppFeature,
  useDatabaseV1Store,
  useProjectV1Store,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import {
  PresetRoleType,
  DEFAULT_PROJECT_NAME,
  isValidProjectName,
} from "@/types";
import { UpdateDatabaseRequest } from "@/types/proto/v1/database_service";
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

const disallowNavigateToConsole = useAppFeature(
  "bb.feature.disallow-navigate-to-console"
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
      await useDatabaseV1Store().batchUpdateDatabases({
        parent: "-",
        requests: databaseList.map((database) => {
          return UpdateDatabaseRequest.fromPartial({
            database: {
              name: database.name,
              project: target.name,
            },
            updateMask: ["project"],
          });
        }),
      });

      const displayDatabaseName =
        databaseList.length > 1
          ? `${databaseList.length} databases`
          : `'${databaseList[0].databaseName}'`;

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: `Successfully transferred ${displayDatabaseName} to project '${target.title}'.`,
      });

      if (!disallowNavigateToConsole.value) {
        router.push({
          ...autoProjectRoute(router, target),
        });
      }
    }

    props.onSuccess(databaseList);
    emit("dismiss");
  } finally {
    loading.value = false;
  }
};
</script>
