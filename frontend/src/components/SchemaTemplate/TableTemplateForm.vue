<template>
  <DrawerContent :title="$t('schema-template.table-template.self')">
    <div class="space-y-6 divide-y divide-block-border w-[calc(100vw-256px)]">
      <div class="space-y-6">
        <!-- category -->
        <div class="sm:col-span-2 sm:col-start-1">
          <label for="category" class="textlabel">
            {{ $t("schema-template.form.category") }}
          </label>
          <p class="text-sm text-gray-500 mb-2">
            {{ $t("schema-template.form.category-desc") }}
          </p>
          <div class="relative flex flex-row justify-between items-center mt-1">
            <DropdownInput
              v-model:value="editing.category"
              :options="categoryOptions"
              :placeholder="$t('schema-template.form.unclassified')"
              :disabled="readonly"
              :consistent-menu-width="true"
              :allow-filter="true"
            />
          </div>
        </div>

        <div class="w-full mb-6 space-y-1">
          <label for="engine" class="textlabel">
            {{ $t("database.engine") }}
          </label>
          <InstanceEngineRadioGrid
            v-model:engine="editing.engine"
            :engine-list="engineList"
            :disabled="readonly"
            class="grid-cols-4 gap-2"
          />
        </div>

        <div v-if="classificationConfig" class="sm:col-span-2 sm:col-start-1">
          <label for="column-name" class="textlabel">
            {{ $t("schema-template.classification.self") }}
          </label>
          <div class="flex items-center gap-x-2 mt-1">
            <ClassificationLevelBadge
              :classification="tableClassificationId"
              :classification-config="classificationConfig"
            />
            <div v-if="!readonly" class="flex">
              <MiniActionButton
                v-if="tableClassificationId"
                @click.prevent="tableClassificationId = ''"
              >
                <XIcon class="w-4 h-4" />
              </MiniActionButton>
              <MiniActionButton
                @click.prevent="state.showClassificationDrawer = true"
              >
                <PencilIcon class="w-4 h-4" />
              </MiniActionButton>
            </div>
          </div>
        </div>
      </div>
      <div class="space-y-6 pt-6">
        <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4">
          <!-- table name -->
          <div class="sm:col-span-2 sm:col-start-1">
            <label for="table-name" class="textlabel">
              {{ $t("schema-template.form.table-name") }}
              <span class="text-red-600 mr-2">*</span>
            </label>
            <NInput
              :value="editing.table.name"
              :disabled="readonly"
              placeholder="table name"
              @update:value="handleUpdateTableName"
            />
          </div>

          <!-- comment -->
          <div class="sm:col-span-4 sm:col-start-1">
            <label for="comment" class="textlabel">
              {{ $t("schema-template.form.comment") }}
            </label>
            <NInput
              v-model:value="editing.table.userComment"
              type="textarea"
              :autosize="{ minRows: 3, maxRows: 3 }"
              :disabled="readonly"
            />
          </div>

          <div class="col-span-4">
            <div
              v-if="!readonly"
              class="w-full py-2 flex items-center space-x-2"
            >
              <NButton size="small" :disabled="false" @click="onColumnAdd">
                <template #icon>
                  <PlusIcon class="w-4 h-4 text-control-placeholder" />
                </template>
                {{ $t("schema-editor.actions.add-column") }}
              </NButton>
              <NButton
                size="small"
                :disabled="false"
                @click="state.showFieldTemplateDrawer = true"
              >
                <template #icon>
                  <FeatureBadge feature="bb.feature.schema-template" />
                  <PlusIcon class="w-4 h-4 text-control-placeholder" />
                </template>
                {{ $t("schema-editor.actions.add-from-template") }}
              </NButton>
            </div>
            <TableColumnEditor
              :readonly="!!readonly"
              :show-foreign-key="false"
              :db="editing.db"
              :database="editing.database"
              :schema="editing.schema"
              :table="editing.table"
              :engine="editing.engine"
              :classification-config-id="classificationConfig?.id"
              :allow-change-primary-keys="true"
              :allow-reorder-columns="allowReorderColumns"
              :max-body-height="640"
              @drop="handleDropColumn"
              @reorder="handleReorderColumn"
              @primary-key-set="setColumnPrimaryKey"
            />
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="w-full flex justify-between items-center">
        <div class="w-full flex justify-end items-center gap-x-3">
          <NButton @click.prevent="$emit('dismiss')">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            v-if="!readonly"
            :disabled="submitDisabled"
            type="primary"
            @click.prevent="onSubmit"
          >
            {{ create ? $t("common.create") : $t("common.update") }}
          </NButton>
        </div>
      </div>
    </template>
  </DrawerContent>

  <Drawer
    :show="state.showFieldTemplateDrawer"
    @close="state.showFieldTemplateDrawer = false"
  >
    <DrawerContent :title="$t('schema-template.field-template.self')">
      <div class="w-[calc(100vw-36rem)] min-w-[64rem] max-w-[calc(100vw-8rem)]">
        <FieldTemplates
          :engine="editing.engine"
          :readonly="false"
          @apply="handleApplyColumnTemplate"
        />
      </div>
    </DrawerContent>
  </Drawer>

  <SelectClassificationDrawer
    v-if="classificationConfig"
    :show="state.showClassificationDrawer"
    :classification-config="classificationConfig"
    @dismiss="state.showClassificationDrawer = false"
    @apply="(id) => (tableClassificationId = id)"
  />
</template>

<script lang="ts" setup>
import { isEqual, cloneDeep, pull } from "lodash-es";
import { PlusIcon, XIcon, PencilIcon } from "lucide-vue-next";
import type { SelectOption } from "naive-ui";
import { NButton, NInput } from "naive-ui";
import { computed, reactive, ref, toRef } from "vue";
import { useI18n } from "vue-i18n";
import FeatureBadge from "@/components/FeatureGuard/FeatureBadge.vue";
import {
  TableColumnEditor,
  provideSchemaEditorContext,
  upsertColumnPrimaryKey,
  removeColumnPrimaryKey,
  removeColumnFromAllForeignKeys,
  type EditTarget,
  type EditStatus,
} from "@/components/SchemaEditorLite";
import {
  Drawer,
  DrawerContent,
  DropdownInput,
  InstanceEngineRadioGrid,
  MiniActionButton,
} from "@/components/v2";
import {
  useSettingV1Store,
  useDatabaseCatalog,
  useNotificationStore,
  pushNotification,
} from "@/store";
import { unknownProject } from "@/types";
import {
  ColumnMetadata,
} from "@/types/proto/v1/database_service";
import {
  SchemaCatalog,
  TableCatalog,
} from "@/types/proto/v1/database_catalog_service";
import {
  SchemaTemplateSetting_TableTemplate,
  type SchemaTemplateSetting_FieldTemplate,
} from "@/types/proto/v1/setting_service";
import { SchemaTemplateSetting } from "@/types/proto/v1/setting_service";
import { arraySwap, instanceV1AllowsReorderColumns } from "@/utils";
import FieldTemplates from "@/views/SchemaTemplate/FieldTemplates.vue";
import ClassificationLevelBadge from "./ClassificationLevelBadge.vue";
import SelectClassificationDrawer from "./SelectClassificationDrawer.vue";
import {
  engineList,
  categoryList,
  classificationConfig,
  mockMetadataFromTableTemplate,
  rebuildTableTemplateFromMetadata,
} from "./utils";

const props = defineProps<{
  create: boolean;
  readonly?: boolean;
  template: SchemaTemplateSetting_TableTemplate;
}>();

const emit = defineEmits(["dismiss"]);

interface LocalState {
  showClassificationDrawer: boolean;
  showFieldTemplateDrawer: boolean;
}

const editing = computed(() => {
  return mockMetadataFromTableTemplate(props.template);
});

const targets = computed(() => {
  const target: EditTarget = {
    database: editing.value.db,
    metadata: editing.value.database,
    baselineMetadata: editing.value.database,
  };
  return [target];
});

const context = provideSchemaEditorContext({
  targets,
  project: ref(unknownProject()),
  resourceType: ref("branch"),
  readonly: toRef(props, "readonly"),
  selectedRolloutObjects: ref(undefined),
  showLastUpdater: ref(false),
  disableDiffColoring: ref(true),
  hidePreview: ref(false),
});

const state = reactive<LocalState>({
  showClassificationDrawer: false,
  showFieldTemplateDrawer: false,
});
const { t } = useI18n();
const settingStore = useSettingV1Store();
const databaseCatalog = useDatabaseCatalog(editing.value.database.name, false);
const tableCatalog = computed(() => {
  const { schema, table } = editing.value;
  const schemaCatalog = databaseCatalog.value.schemas.find(
    (sc) => sc.name === schema.name
  );
  if (!schemaCatalog) return undefined;
  return schemaCatalog.tables.find((tc) => tc.name === table.name);
});

const tableClassificationId = computed({
  get() {
    return tableCatalog.value?.classificationId;
  },
  set(id) {
    const { schema, table } = editing.value;
    let schemaCatalog = databaseCatalog.value.schemas.find(
      (sc) => sc.name === schema.name
    );
    if (!schemaCatalog) {
      schemaCatalog = SchemaCatalog.fromPartial({
        name: schema.name,
        tables: [],
      });
    }
    if (!schemaCatalog.tables) {
      schemaCatalog.tables = [];
    }
    let tableCatalog = schemaCatalog.tables.find(
      (tc) => tc.name === table.name
    );
    if (!tableCatalog) {
      tableCatalog = TableCatalog.fromPartial({
        name: table.name,
        classificationId: id,
      });
      schemaCatalog.tables.push(tableCatalog);
    }
    tableCatalog.classificationId = id ?? "";
  },
});

const metadataForColumn = (column: ColumnMetadata) => {
  const { database, schema, table } = editing.value;
  return {
    database,
    schema,
    table,
    column,
  };
};

const statusForColumn = (column: ColumnMetadata) => {
  return context.getColumnStatus(editing.value.db, metadataForColumn(column));
};

const markColumnStatus = (column: ColumnMetadata, status: EditStatus) => {
  context.markEditStatus(editing.value.db, metadataForColumn(column), status);
};

const categoryOptions = computed(() => {
  return categoryList.value.map<SelectOption>((category) => ({
    label: category,
    value: category,
  }));
});

const allowReorderColumns = computed(() => {
  return instanceV1AllowsReorderColumns(editing.value.engine);
});

const submitDisabled = computed(() => {
  const { table, category } = editing.value;
  if (!table.name || table.columns.length === 0) {
    return true;
  }
  if (table.columns.some((col) => !col.name || !col.type)) {
    return true;
  }
  if (
    !props.create &&
    isEqual(props.template.table, table) &&
    isEqual(props.template.catalog, tableCatalog.value) &&
    props.template.category === category
  ) {
    return true;
  }
  return false;
});

const onSubmit = async () => {
  const template = rebuildTableTemplateFromMetadata(editing.value);
  const setting = await settingStore.fetchSettingByName(
    "bb.workspace.schema-template"
  );

  const settingValue = SchemaTemplateSetting.fromJSON({});
  if (setting?.value?.schemaTemplateSettingValue) {
    Object.assign(
      settingValue,
      cloneDeep(setting.value.schemaTemplateSettingValue)
    );
  }

  const index = settingValue.tableTemplates.findIndex(
    (t) => t.id === template.id
  );
  if (index >= 0) {
    settingValue.tableTemplates[index] = template;
  } else {
    settingValue.tableTemplates.push(template);
  }

  await settingStore.upsertSetting({
    name: "bb.workspace.schema-template",
    value: {
      schemaTemplateSettingValue: settingValue,
    },
  });

  useNotificationStore().pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });

  emit("dismiss");
};

const onColumnAdd = () => {
  const { db, database, schema, table } = editing.value;
  const column = ColumnMetadata.fromPartial({});
  table.columns.push(column);
  markColumnStatus(column, "created");

  context.queuePendingScrollToColumn({
    db,
    metadata: {
      database,
      schema,
      table,
      column,
    },
  });
};

const handleDropColumn = (column: ColumnMetadata) => {
  const { table } = editing.value;
  // Disallow to drop the last column.
  const nonDroppedColumns = table.columns.filter((column) => {
    return statusForColumn(column) !== "dropped";
  });
  if (nonDroppedColumns.length === 1) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.cannot-drop-the-last-column"),
    });
    return;
  }
  const status = statusForColumn(column);
  if (status === "created") {
    pull(table.columns, column);
    table.columns = table.columns.filter((col) => col !== column);

    removeColumnPrimaryKey(table, column.name);
    removeColumnFromAllForeignKeys(table, column.name);
  } else {
    markColumnStatus(column, "dropped");
  }
};

const setColumnPrimaryKey = (column: ColumnMetadata, isPrimaryKey: boolean) => {
  if (isPrimaryKey) {
    column.nullable = false;
    upsertColumnPrimaryKey(
      editing.value.engine,
      editing.value.table,
      column.name
    );
  } else {
    removeColumnPrimaryKey(editing.value.table, column.name);
  }
  markColumnStatus(column, "updated");
};

const handleApplyColumnTemplate = (
  template: SchemaTemplateSetting_FieldTemplate
) => {
  state.showFieldTemplateDrawer = false;
  if (!template.column) {
    return;
  }
  const { db, table, engine } = editing.value;
  if (template.engine !== engine) {
    return;
  }
  const column = cloneDeep(template.column);
  table.columns.push(column);
  if (template.catalog) {
    context.upsertColumnConfig(db, metadataForColumn(column), (config) => {
      Object.assign(config, template.catalog);
    });
  }
  markColumnStatus(column, "created");
  context.queuePendingScrollToColumn({
    db: db,
    metadata: metadataForColumn(column),
  });
};

const handleReorderColumn = (
  column: ColumnMetadata,
  index: number,
  delta: -1 | 1
) => {
  const target = index + delta;
  const { columns } = editing.value.table;
  if (target < 0) return;
  if (target >= columns.length) return;
  arraySwap(columns, index, target);
};

const handleUpdateTableName = (name: string) => {
  editing.value.table.name = name;
  const tc = tableCatalog.value;
  if (tc) {
    tc.name = name;
  }
};
</script>
