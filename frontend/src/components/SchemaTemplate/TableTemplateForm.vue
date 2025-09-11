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
              <RequiredStar />
            </label>
            <NInput
              :value="editing.tableMetadata.name"
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
              v-model:value="editing.tableMetadata.userComment"
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
                  <PlusIcon class="w-4 h-4 text-control-placeholder" />
                </template>
                {{ $t("schema-editor.actions.add-from-template") }}
              </NButton>
            </div>

            <TableColumnEditor
              :readonly="!!readonly"
              :show-foreign-key="false"
              :db="editing.db"
              :database="editing.databaseMetadata"
              :schema="editing.schemaMetadata"
              :table="editing.tableMetadata"
              :engine="editing.engine"
              :allow-change-primary-keys="true"
              :allow-reorder-columns="allowReorderColumns"
              :max-body-height="640"
              :show-database-catalog-column="true"
              :show-classification-column="'ALWAYS'"
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
import { create as createProto } from "@bufbuild/protobuf";
import { cloneDeep, isEqual, pull } from "lodash-es";
import { PencilIcon, PlusIcon, XIcon } from "lucide-vue-next";
import type { SelectOption } from "naive-ui";
import { NButton, NInput } from "naive-ui";
import { computed, reactive, ref, toRef } from "vue";
import { useI18n } from "vue-i18n";
import RequiredStar from "@/components/RequiredStar.vue";
import {
  TableColumnEditor,
  provideSchemaEditorContext,
  removeColumnFromAllForeignKeys,
  removeColumnPrimaryKey,
  upsertColumnPrimaryKey,
  type EditStatus,
  type EditTarget,
} from "@/components/SchemaEditorLite";
import {
  Drawer,
  DrawerContent,
  DropdownInput,
  InstanceEngineRadioGrid,
  MiniActionButton,
} from "@/components/v2";
import { pushNotification, useSettingV1Store } from "@/store";
import { unknownProject } from "@/types";
import { TableCatalogSchema } from "@/types/proto-es/v1/database_catalog_service_pb";
import {
  ColumnMetadataSchema,
  type ColumnMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { type ColumnMetadata as OldColumnMetadata } from "@/types/proto-es/v1/database_service_pb";
import {
  SchemaTemplateSettingSchema,
  Setting_SettingName,
  type SchemaTemplateSetting_FieldTemplate,
  type SchemaTemplateSetting_TableTemplate,
  ValueSchema as SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { arraySwap, instanceV1AllowsReorderColumns } from "@/utils";
import FieldTemplates from "@/views/SchemaTemplate/FieldTemplates.vue";
import ClassificationLevelBadge from "./ClassificationLevelBadge.vue";
import SelectClassificationDrawer from "./SelectClassificationDrawer.vue";
import {
  categoryList,
  classificationConfig,
  engineList,
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
    metadata: editing.value.databaseMetadata,
    baselineMetadata: editing.value.databaseMetadata,
    catalog: editing.value.databaseCatalog,
    baselineCatalog: editing.value.databaseCatalog,
  };
  return [target];
});

const context = provideSchemaEditorContext({
  targets,
  project: ref(unknownProject()),
  classificationConfigId: ref(classificationConfig.value?.id),
  readonly: toRef(props, "readonly"),
  selectedRolloutObjects: ref(undefined),
  disableDiffColoring: ref(true),
  hidePreview: ref(false),
});

const state = reactive<LocalState>({
  showClassificationDrawer: false,
  showFieldTemplateDrawer: false,
});
const { t } = useI18n();
const settingStore = useSettingV1Store();

const tableCatalog = computed(() =>
  context.getTableCatalog({
    database: editing.value.databaseMetadata.name,
    schema: editing.value.schemaMetadata.name,
    table: editing.value.tableMetadata.name,
  })
);

const tableClassificationId = computed({
  get() {
    return tableCatalog.value?.classification;
  },
  set(id) {
    context.upsertTableCatalog(
      {
        database: editing.value.databaseCatalog.name,
        schema: editing.value.schemaCatalog.name,
        table: editing.value.tableMetadata.name,
      },
      (catalog) => {
        catalog.classification = id ?? "";
      }
    );
  },
});

const metadataForColumn = (column: ColumnMetadata) => {
  const {
    databaseMetadata: database,
    schemaMetadata: schema,
    tableMetadata: table,
  } = editing.value;
  return {
    database: database,
    schema: schema,
    table: table,
    column: column,
  };
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
  const { tableMetadata, category } = editing.value;
  if (!tableMetadata.name || tableMetadata.columns.length === 0) {
    return true;
  }
  if (tableMetadata.columns.some((col) => !col.name || !col.type)) {
    return true;
  }
  if (
    !props.create &&
    isEqual(props.template.table, tableMetadata) &&
    isEqual(props.template.catalog, tableCatalog.value) &&
    props.template.category === category
  ) {
    return true;
  }
  return false;
});

const onSubmit = async () => {
  const template = rebuildTableTemplateFromMetadata({
    ...editing.value,
    tableCatalog: tableCatalog.value
      ? tableCatalog.value
      : createProto(TableCatalogSchema, {}),
  });
  const setting = await settingStore.fetchSettingByName(
    Setting_SettingName.SCHEMA_TEMPLATE
  );

  const existingValue =
    setting?.value?.value?.case === "schemaTemplateSettingValue"
      ? setting.value.value.value
      : undefined;
  const settingValue = createProto(SchemaTemplateSettingSchema, {
    columnTypes: existingValue?.columnTypes ?? [],
    fieldTemplates: existingValue?.fieldTemplates ?? [],
    tableTemplates: existingValue?.tableTemplates ?? [],
  });

  const index = settingValue.tableTemplates.findIndex(
    (t) => t.id === template.id
  );
  if (index >= 0) {
    settingValue.tableTemplates[index] = template;
  } else {
    settingValue.tableTemplates.push(template);
  }

  await settingStore.upsertSetting({
    name: Setting_SettingName.SCHEMA_TEMPLATE,
    value: createProto(SettingValueSchema, {
      value: {
        case: "schemaTemplateSettingValue",
        value: settingValue,
      },
    }),
  });

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });

  emit("dismiss");
};

const onColumnAdd = () => {
  const {
    db,
    databaseMetadata: database,
    schemaMetadata: schema,
    tableMetadata: table,
  } = editing.value;
  const column = createProto(ColumnMetadataSchema, {});
  table.columns.push(column);
  markColumnStatus(column, "created");

  context.queuePendingScrollToColumn({
    db,
    metadata: {
      database: database,
      schema: schema,
      table: table,
      column: column,
    },
  });
};

const handleDropColumn = (column: ColumnMetadata) => {
  const { tableMetadata } = editing.value;
  pull(tableMetadata.columns, column);
  tableMetadata.columns = tableMetadata.columns.filter((col) => col !== column);

  removeColumnPrimaryKey(tableMetadata, column.name);
  removeColumnFromAllForeignKeys(tableMetadata, column.name);
  // Convert back to update proto-es tableMetadata
  context.removeColumnCatalog({
    database: editing.value.databaseCatalog.name,
    schema: editing.value.schemaCatalog.name,
    table: editing.value.tableMetadata.name,
    column: column.name,
  });
};

const setColumnPrimaryKey = (column: ColumnMetadata, isPrimaryKey: boolean) => {
  if (isPrimaryKey) {
    column.nullable = false;
    upsertColumnPrimaryKey(
      editing.value.engine,
      editing.value.tableMetadata,
      column.name
    );
  } else {
    removeColumnPrimaryKey(editing.value.tableMetadata, column.name);
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
  const { db, tableMetadata, engine } = editing.value;
  if (template.engine !== engine) {
    return;
  }
  const column = cloneDeep(template.column);
  tableMetadata.columns.push(column);
  if (template.catalog) {
    context.upsertColumnCatalog(
      {
        database: editing.value.databaseCatalog.name,
        schema: editing.value.schemaCatalog.name,
        table: editing.value.tableCatalog.name,
        column: template.column.name,
      },
      (config) => {
        Object.assign(config, template.catalog);
      }
    );
  }
  markColumnStatus(column, "created");
  context.queuePendingScrollToColumn({
    db: db,
    metadata: metadataForColumn(column),
  });
};

const handleReorderColumn = (
  column: OldColumnMetadata,
  index: number,
  delta: -1 | 1
) => {
  const target = index + delta;
  const { columns } = editing.value.tableMetadata;
  if (target < 0) return;
  if (target >= columns.length) return;
  arraySwap(columns, index, target);
};

const handleUpdateTableName = (name: string) => {
  context.upsertTableCatalog(
    {
      database: editing.value.databaseCatalog.name,
      schema: editing.value.schemaCatalog.name,
      table: editing.value.tableMetadata.name,
    },
    (catalog) => {
      catalog.name = name;
    }
  );
  context.removeTableCatalog({
    database: editing.value.databaseCatalog.name,
    schema: editing.value.schemaCatalog.name,
    table: editing.value.tableMetadata.name,
  });
  editing.value.tableMetadata.name = name;
};
</script>
