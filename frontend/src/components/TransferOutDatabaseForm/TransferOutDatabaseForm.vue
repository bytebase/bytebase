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
      <div v-else class="space-y-4">
        <div>
          <span class="text-main">
            {{ $t("database.transfer.select-databases") }}
            <span class="text-red-500">*</span>
          </span>
          <MultipleDatabaseSelector
            v-model:selected-uid-list="selectedUidList"
            :transfer-source="'OTHER'"
            :database-list="[...databaseList]"
          />
        </div>
        <NDivider class="w-full py-2" />
        <div class="space-y-4">
          <span class="text-main">
            {{ $t("database.transfer.select-target-project") }}
            <span class="text-red-500">*</span>
          </span>
          <ProjectSelect
            v-model:project="targetProjectId"
            :allowed-project-role-list="[PresetRoleType.PROJECT_OWNER]"
            :include-default-project="true"
          />
        </div>
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
import { NButton, NTooltip, NDivider } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { ProjectSelect, DrawerContent } from "@/components/v2";
import {
  pushNotification,
  useDatabaseV1Store,
  useProjectV1Store,
} from "@/store";
import { PresetRoleType, ComposedDatabase, UNKNOWN_ID } from "@/types";

const props = defineProps<{
  databaseList: ComposedDatabase[];
  selectedDatabaseUidList?: string[];
}>();

const emit = defineEmits<{
  (e: "dismiss"): void;
}>();

const { t } = useI18n();
const projectStore = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const loading = ref(false);

const selectedUidList = ref<string[]>(props.selectedDatabaseUidList ?? []);

watch(
  () => props.selectedDatabaseUidList ?? [],
  (list) => (selectedUidList.value = list),
  { immediate: true }
);

const selectedDatabaseList = computed(() => {
  return selectedUidList.value.map((uid) => {
    return databaseStore.getDatabaseByUID(uid);
  });
});

const targetProjectId = ref<string>();
const targetProject = computed(() => {
  const id = targetProjectId.value;
  if (!id || id === String(UNKNOWN_ID)) return undefined;
  return projectStore.getProjectByUID(id);
});

const validationErrors = computed(() => {
  const errors: string[] = [];
  if (!targetProject.value) {
    errors.push(t("database.transfer.errors.select-target-project"));
  }
  if (selectedUidList.value.length === 0) {
    errors.push(t("database.transfer.errors.select-at-least-one-database"));
  }
  return errors;
});

const allowTransfer = computed(() => {
  return validationErrors.value.length === 0;
});

const doTransfer = async () => {
  const target = targetProject.value!;
  if (!target) return;

  const databaseList = selectedDatabaseList.value.filter(
    (db) => db.project !== target.name
  );

  try {
    loading.value = true;

    if (databaseList.length > 0) {
      await useDatabaseV1Store().transferDatabases(
        selectedDatabaseList.value,
        target.name
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
    }

    emit("dismiss");
  } finally {
    loading.value = false;
  }
};
</script>
