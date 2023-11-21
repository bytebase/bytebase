<template>
  <DrawerContent :title="$t('schema-template.table-template.self')">
    <div
      class="space-y-6 divide-y divide-block-border w-[calc(100vw-256px)] !max-w-[60rem]"
    >
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
              v-model:value="state.category"
              :options="categoryOptions"
              :placeholder="$t('schema-template.form.unclassified')"
              :disabled="!allowEdit"
              :consistent-menu-width="true"
            />
          </div>
        </div>

        <div class="w-full mb-6 space-y-1">
          <label for="engine" class="textlabel">
            {{ $t("database.engine") }}
          </label>
          <div class="grid grid-cols-4 gap-2">
            <NButton
              v-for="engine in engineList"
              :key="engine"
              size="large"
              ghost
              class="table-template-engine-button"
              :type="state.engine === engine ? 'primary' : 'default'"
              @click="changeEngine(engine)"
            >
              <NRadio
                :checked="state.engine === engine"
                size="large"
                class="btn mr-2 pointer-events-none"
              />
              <EngineIcon
                :engine="engine"
                class="w-5 h-auto max-h-[20px] object-contain mr-1 ml-0"
              />
              <RichEngineName
                :engine="engine"
                tag="p"
                class="text-center text-sm !text-main"
              />
            </NButton>
          </div>
        </div>

        <div v-if="classificationConfig" class="sm:col-span-2 sm:col-start-1">
          <label for="column-name" class="textlabel">
            {{ $t("schema-template.classification.self") }}
          </label>
          <div class="flex items-center gap-x-2 mt-1">
            <ClassificationLevelBadge
              :classification="state.table?.classification"
              :classification-config="classificationConfig"
            />
            <div v-if="allowEdit" class="flex">
              <MiniActionButton
                v-if="state.table?.classification"
                @click.prevent="state.table!.classification = ''"
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
              v-model:value="state.table!.name"
              placeholder="table name"
              :disabled="!allowEdit"
            />
          </div>

          <!-- comment -->
          <div class="sm:col-span-4 sm:col-start-1">
            <label for="comment" class="textlabel">
              {{ $t("schema-template.form.comment") }}
            </label>
            <NInput
              v-model:value="state.table!.userComment"
              type="textarea"
              :autosize="{ minRows: 3, maxRows: 3 }"
              :disabled="!allowEdit"
            />
          </div>

          <div class="col-span-4">
            <div
              v-if="allowEdit"
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
              :readonly="!allowEdit"
              :show-foreign-key="false"
              :table="state.table"
              :engine="state.engine"
              :classification-config-id="classificationConfig?.id"
              :allow-reorder-columns="allowReorderColumns"
              :max-body-height="640"
              @drop="onColumnDrop"
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
            v-if="allowEdit"
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
          :engine="state.engine"
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
    @apply="onClassificationSelect"
  />
</template>

<script lang="ts" setup>
import { isEqual, cloneDeep } from "lodash-es";
import { PlusIcon, XIcon, PencilIcon } from "lucide-vue-next";
import { NButton, NInput, NRadio, SelectOption } from "naive-ui";
import { computed, nextTick, reactive } from "vue";
import { useI18n } from "vue-i18n";
import FeatureBadge from "@/components/FeatureGuard/FeatureBadge.vue";
import { EngineIcon } from "@/components/Icon";
import TableColumnEditor from "@/components/SchemaEditorV1/Panels/TableColumnEditor";
import { transformTableEditToMetadata } from "@/components/SchemaEditorV1/utils";
import { Drawer, DrawerContent, RichEngineName } from "@/components/v2";
import { useSettingV1Store, useNotificationStore } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import { ColumnMetadata, TableConfig } from "@/types/proto/v1/database_service";
import {
  SchemaTemplateSetting,
  SchemaTemplateSetting_FieldTemplate,
  SchemaTemplateSetting_TableTemplate,
} from "@/types/proto/v1/setting_service";
import { convertTableMetadataToTable } from "@/types/v1/schemaEditor";
import {
  Column,
  Table,
  convertColumnMetadataToColumn,
} from "@/types/v1/schemaEditor";
import {
  arraySwap,
  instanceV1AllowsReorderColumns,
  useWorkspacePermissionV1,
} from "@/utils";
import FieldTemplates from "@/views/SchemaTemplate/FieldTemplates.vue";
import { engineList, categoryList, classificationConfig } from "./utils";

const props = defineProps<{
  create: boolean;
  readonly?: boolean;
  template: SchemaTemplateSetting_TableTemplate;
}>();

const emit = defineEmits(["dismiss"]);

interface LocalState {
  id: string;
  engine: Engine;
  category: string;
  table: Table;
  showClassificationDrawer: boolean;
  showFieldTemplateDrawer: boolean;
}

const state = reactive<LocalState>({
  id: props.template.id,
  engine: props.template.engine,
  category: props.template.category,
  table: convertTableMetadataToTable(
    Object.assign({}, props.template.table),
    "normal",
    props.template.config
  ),
  showClassificationDrawer: false,
  showFieldTemplateDrawer: false,
});
const { t } = useI18n();
const settingStore = useSettingV1Store();
const allowEdit = computed(() => {
  return (
    useWorkspacePermissionV1("bb.permission.workspace.manage-general").value &&
    !props.readonly
  );
});

const categoryOptions = computed(() => {
  return categoryList.value.map<SelectOption>((category) => ({
    label: category,
    value: category,
  }));
});

const allowReorderColumns = computed(() => {
  return instanceV1AllowsReorderColumns(state.engine);
});

const changeEngine = (engine: Engine) => {
  if (allowEdit.value) {
    state.engine = engine;
  }
};

const submitDisabled = computed(() => {
  if (!state.table.name || state.table.columnList.length === 0) {
    return true;
  }
  if (state.table.columnList.some((col) => !col.name || !col.type)) {
    return true;
  }
  if (
    !props.create &&
    isEqual(
      convertTableMetadataToTable(
        props.template.table!,
        "normal",
        props.template.config
      ),
      state.table
    )
  ) {
    return true;
  }
  return false;
});

const onSubmit = async () => {
  const template: SchemaTemplateSetting_TableTemplate = {
    id: state.id,
    engine: state.engine,
    category: state.category,
    table: transformTableEditToMetadata(state.table),
    config: TableConfig.fromPartial({
      name: state.table.name,
      columnConfigs: state.table.columnList.map((col) => col.config),
    }),
  };
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
  const column = convertColumnMetadataToColumn(
    ColumnMetadata.fromPartial({}),
    "created"
  );
  state.table.columnList.push(column);
  nextTick(() => {
    const container = document.querySelector("#table-editor-container");
    (
      container?.querySelector(
        `.column-${column.id} .column-name-input`
      ) as HTMLInputElement
    )?.focus();
  });
};

const onColumnDrop = (column: Column) => {
  state.table.columnList = state.table.columnList.filter(
    (item) => item.id !== column.id
  );
  state.table.primaryKey.columnIdList =
    state.table.primaryKey.columnIdList.filter(
      (columnId) => columnId !== column.id
    );
};

const setColumnPrimaryKey = (column: Column, isPrimaryKey: boolean) => {
  if (isPrimaryKey) {
    column.nullable = false;
    state.table.primaryKey.columnIdList.push(column.id);
  } else {
    state.table.primaryKey.columnIdList =
      state.table.primaryKey.columnIdList.filter(
        (columnId) => columnId !== column.id
      );
  }
};

const handleApplyColumnTemplate = (
  template: SchemaTemplateSetting_FieldTemplate
) => {
  state.showFieldTemplateDrawer = false;

  if (template.engine !== state.engine || !template.column) {
    return;
  }
  const column = convertColumnMetadataToColumn(
    template.column,
    "created",
    template.config
  );
  state.table.columnList.push(column);
};

const onClassificationSelect = (id: string) => {
  state.table.classification = id;
};

const handleReorderColumn = (column: Column, index: number, delta: -1 | 1) => {
  const target = index + delta;
  const { columnList } = state.table;
  if (target < 0) return;
  if (target >= columnList.length) return;

  arraySwap(columnList, index, target);
};
</script>

<style lang="postcss" scoped>
.table-template-engine-button :deep(.n-button__content) {
  @apply w-full justify-start;
}
</style>
