<template>
  <Drawer v-model:show="show" placement="right" :resizable="true">
    <DrawerContent
      :title="$t('schema-editor.self')"
      class="w-[70vw] min-w-4xl max-w-[100vw]"
    >
      <div class="flex flex-col gap-y-4 h-full">
        <!-- Database selector for multi-database scenarios -->
        <div v-if="databaseNames.length > 1" class="flex items-center gap-x-2">
          <span class="text-sm text-control-light">
            {{ $t("schema-editor.template-database") }}:
          </span>
          <NSelect
            v-model:show="showSelect"
            :value="state.selectedDatabaseName"
            :options="databaseOptions"
            :disabled="state.isPreparingMetadata"
            class="w-64"
            @update:value="handleDatabaseChange"
          >
          <template #action>
            <NButton
              v-if="DEFAULT_VISIBLE_TARGETS * (state.databaseSelectorPageIndex + 1) < databaseNames.length"
              class="w-full!"
              quaternary
              :loading="state.isLoadingNextPage"
              @click="onNextPage">
              {{ $t("common.load-more") }}
            </NButton>
          </template>
          </NSelect>
        </div>

        <SchemaEditorLite
          ref="schemaEditorRef"
          :project="project"
          :targets="editTargets"
          :loading="state.isPreparingMetadata"
          class="flex-1"
        />
      </div>

      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="handleCancel">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            type="primary"
            :disabled="!hasPendingChanges"
            @click="handleInsertSQL"
          >
            {{ $t("schema-editor.insert-sql") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { watchDebounced } from "@vueuse/core";
import { cloneDeep } from "lodash-es";
import { NButton, NSelect } from "naive-ui";
import { computed, nextTick, reactive, ref, watch } from "vue";
import SchemaEditorLite, {
  type EditTarget,
  generateDiffDDL,
} from "@/components/SchemaEditorLite";
import { Drawer, DrawerContent } from "@/components/v2";
import { useDatabaseV1Store, useDBSchemaV1Store } from "@/store";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { engineSupportsSchemaEditor } from "@/utils/schemaEditor";
import { DEFAULT_VISIBLE_TARGETS } from "../SpecDetailView/context";

const props = defineProps<{
  show: boolean;
  databaseNames: string[];
  project: Project;
}>();

const emit = defineEmits<{
  (event: "update:show", show: boolean): void;
  (event: "insert", sql: string): void;
}>();

const show = computed({
  get: () => props.show,
  set: (value) => emit("update:show", value),
});

const schemaEditorRef = ref<InstanceType<typeof SchemaEditorLite>>();
const dbSchemaStore = useDBSchemaV1Store();
const databaseStore = useDatabaseV1Store();
const showSelect = ref(false);

const state = reactive({
  isPreparingMetadata: false,
  isLoadingNextPage: false,
  targets: [] as EditTarget[],
  selectedDatabaseName: "",
  databaseSelectorPageIndex: 0,
});

const onNextPage = () => {
  state.isLoadingNextPage = true;
  state.databaseSelectorPageIndex++;
  nextTick(() => (showSelect.value = true));
};

watchDebounced(
  () => state.databaseSelectorPageIndex,
  async (index) => {
    try {
      await databaseStore.batchGetOrFetchDatabases(
        props.databaseNames.slice(
          DEFAULT_VISIBLE_TARGETS * index,
          DEFAULT_VISIBLE_TARGETS * (index + 1)
        )
      );
    } finally {
      state.isLoadingNextPage = false;
    }
  },
  { immediate: true, debounce: DEBOUNCE_SEARCH_DELAY }
);

const databaseOptions = computed(() => {
  const pageIndex = Math.max(
    0,
    state.isLoadingNextPage
      ? state.databaseSelectorPageIndex - 1
      : state.databaseSelectorPageIndex
  );
  return props.databaseNames
    .slice(0, DEFAULT_VISIBLE_TARGETS * (pageIndex + 1))
    .map((dbName) => {
      const db = databaseStore.getDatabaseByName(dbName);
      return {
        label: `${db.databaseName} (${db.instanceResource.title})`,
        value: db.name,
        disabled: !engineSupportsSchemaEditor(db.instanceResource.engine),
      };
    });
});

const editTargets = computed(() => state.targets);

const hasPendingChanges = computed(() => {
  return schemaEditorRef.value?.isDirty ?? false;
});

const prepareDatabaseMetadata = async (databaseName: string) => {
  if (!databaseName) return;

  state.isPreparingMetadata = true;
  state.targets = [];

  const [metadata, database] = await Promise.all([
    dbSchemaStore.getOrFetchDatabaseMetadata({
      database: databaseName,
      skipCache: true,
      limit: 200,
    }),
    databaseStore.getOrFetchDatabaseByName(databaseName),
  ]);

  state.targets = [
    {
      database: database,
      metadata: cloneDeep(metadata),
      baselineMetadata: metadata,
    },
  ];

  state.isPreparingMetadata = false;
};

const handleDatabaseChange = async (selectedDatabaseName: string) => {
  if (selectedDatabaseName) {
    await prepareDatabaseMetadata(selectedDatabaseName);
  }
  state.selectedDatabaseName = selectedDatabaseName;
};

watch(
  () => props.show,
  (show) => {
    if (show && props.databaseNames.length > 0) {
      // Initialize with first database
      handleDatabaseChange(props.databaseNames[0]);
    }
  },
  { immediate: true }
);

const handleCancel = () => {
  show.value = false;
};

const handleInsertSQL = async () => {
  const applyMetadataEdit = schemaEditorRef.value?.applyMetadataEdit;
  if (typeof applyMetadataEdit !== "function") {
    return;
  }

  const target = state.targets[0];
  if (!target) return;

  const { database, baselineMetadata } = target;
  const { metadata } = applyMetadataEdit(database, target.metadata);

  const result = await generateDiffDDL({
    database,
    sourceMetadata: baselineMetadata,
    targetMetadata: metadata,
  });

  if (result.statement) {
    emit("insert", result.statement);
    show.value = false;
  }
};
</script>
