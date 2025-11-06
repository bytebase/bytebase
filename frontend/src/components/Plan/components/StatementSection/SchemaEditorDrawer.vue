<template>
  <Drawer
    v-model:show="show"
    placement="right"
    :resizable="true"
    :default-width="'50%'"
    :width="undefined"
    class="max-w-[90vw]!"
  >
    <DrawerContent :title="$t('schema-editor.self')">
      <SchemaEditorLite
        ref="schemaEditorRef"
        :project="project"
        :targets="editTargets"
        :loading="isPreparingMetadata"
      />

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
import { cloneDeep } from "lodash-es";
import { NButton } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import SchemaEditorLite, {
  generateDiffDDL,
  type EditTarget,
} from "@/components/SchemaEditorLite";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  useDatabaseV1Store,
  useDBSchemaV1Store,
  useDatabaseCatalogV1Store,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

const props = defineProps<{
  show: boolean;
  database: ComposedDatabase;
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
const dbCatalogStore = useDatabaseCatalogV1Store();

const state = reactive({
  isPreparingMetadata: false,
  targets: [] as EditTarget[],
});

const editTargets = computed(() => state.targets);

const hasPendingChanges = computed(() => {
  return schemaEditorRef.value?.isDirty ?? false;
});

const prepareDatabaseMetadata = async () => {
  if (!props.database) return;

  state.isPreparingMetadata = true;
  state.targets = [];

  const [metadata, catalog] = await Promise.all([
    dbSchemaStore.getOrFetchDatabaseMetadata({
      database: props.database.name,
      skipCache: true,
      limit: 200,
    }),
    dbCatalogStore.getOrFetchDatabaseCatalog({
      database: props.database.name,
      skipCache: true,
    }),
  ]);

  state.targets = [
    {
      database: props.database,
      metadata: cloneDeep(metadata),
      baselineMetadata: metadata,
      catalog: cloneDeep(catalog),
      baselineCatalog: catalog,
    },
  ];

  state.isPreparingMetadata = false;
};

watch(
  () => props.show,
  (show) => {
    if (show) {
      prepareDatabaseMetadata();
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

  const { database, baselineMetadata, baselineCatalog } = target;
  const { metadata, catalog } = applyMetadataEdit(
    database,
    target.metadata,
    target.catalog
  );

  const result = await generateDiffDDL({
    database,
    sourceMetadata: baselineMetadata,
    targetMetadata: metadata,
    sourceCatalog: baselineCatalog,
    targetCatalog: catalog,
    allowEmptyDiffDDLWithConfigChange: false,
  });

  if (result.statement) {
    emit("insert", result.statement);
    show.value = false;
  }
};
</script>
