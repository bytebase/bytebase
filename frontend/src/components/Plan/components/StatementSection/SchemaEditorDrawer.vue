<template>
  <Drawer v-model:show="show" placement="right" :resizable="true">
    <DrawerContent
      :title="$t('schema-editor.self')"
      class="w-[70vw] min-w-4xl max-w-[100vw]"
    >
      <div class="flex flex-col gap-y-4 h-full">
        <!-- Database selector for multi-database scenarios -->
        <div v-if="databases.length > 1" class="flex items-center gap-x-2">
          <span class="text-sm text-control-light">
            {{ $t("schema-editor.template-database") }}:
          </span>
          <NSelect
            v-model:value="state.selectedDatabaseName"
            :options="databaseOptions"
            class="w-64"
            @update:value="handleDatabaseChange"
          />
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
import { cloneDeep } from "lodash-es";
import { NButton, NSelect } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import SchemaEditorLite, {
  type EditTarget,
  generateDiffDDL,
} from "@/components/SchemaEditorLite";
import { Drawer, DrawerContent } from "@/components/v2";
import { useDatabaseCatalogV1Store, useDBSchemaV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

const props = defineProps<{
  show: boolean;
  databases: ComposedDatabase[];
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
  selectedDatabaseName: "",
});

const databaseOptions = computed(() => {
  return props.databases.map((db) => ({
    label: `${db.databaseName} (${db.instanceResource.title})`,
    value: db.name,
  }));
});

const selectedDatabase = computed(() => {
  return (
    props.databases.find((db) => db.name === state.selectedDatabaseName) ||
    props.databases[0]
  );
});

const editTargets = computed(() => state.targets);

const hasPendingChanges = computed(() => {
  return schemaEditorRef.value?.isDirty ?? false;
});

const prepareDatabaseMetadata = async (database: ComposedDatabase) => {
  if (!database) return;

  state.isPreparingMetadata = true;
  state.targets = [];

  const [metadata, catalog] = await Promise.all([
    dbSchemaStore.getOrFetchDatabaseMetadata({
      database: database.name,
      skipCache: true,
      limit: 200,
    }),
    dbCatalogStore.getOrFetchDatabaseCatalog({
      database: database.name,
      skipCache: true,
    }),
  ]);

  state.targets = [
    {
      database: database,
      metadata: cloneDeep(metadata),
      baselineMetadata: metadata,
      catalog: cloneDeep(catalog),
      baselineCatalog: catalog,
    },
  ];

  state.isPreparingMetadata = false;
};

const handleDatabaseChange = async () => {
  if (selectedDatabase.value) {
    await prepareDatabaseMetadata(selectedDatabase.value);
  }
};

watch(
  () => props.show,
  (show) => {
    if (show && props.databases.length > 0) {
      // Initialize with first database
      state.selectedDatabaseName = props.databases[0].name;
      prepareDatabaseMetadata(props.databases[0]);
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
  });

  if (result.statement) {
    emit("insert", result.statement);
    show.value = false;
  }
};
</script>
