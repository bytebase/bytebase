<template>
  <div class="w-full space-y-4 text-sm">
    <FeatureAttention
      feature="bb.feature.schema-template"
      custom-class="my-4"
    />
    <div class="space-y-4">
      <div class="flex items-center justify-between gap-x-6">
        <div class="flex-1 textinfolabel !leading-8">
          {{ $t("schema-template.column-type-restriction.description") }}
        </div>
      </div>
    </div>
    <div class="w-full max-w-lg flex flex-col justify-start items-start gap-2">
      <div class="w-full flex flex-row justify-between items-center">
        <div class="flex flex-row justify-start items-center">
          <EngineIcon :engine="Engine.MYSQL" :custom-class="'mr-1'" />
          MySQL
        </div>
        <NRadioGroup
          v-model:value="columnTypeTemplateForMySQL.enabled"
          class="gap-x-2"
          :disabled="!hasPermission || !hasFeature"
          @change="handleMySQLEnabledChange"
        >
          <NRadio
            :value="false"
            :label="$t('schema-template.column-type-restriction.allow-all')"
          />
          <NRadio
            :value="true"
            :label="
              $t('schema-template.column-type-restriction.allow-limited-types')
            "
          />
        </NRadioGroup>
      </div>
      <NSelect
        ref="typesSelectorRefForMySQL"
        v-model:value="columnTypeTemplateForMySQL.types"
        filterable
        multiple
        tag
        :disabled="
          !columnTypeTemplateForMySQL.enabled || !hasPermission || !hasFeature
        "
        placeholder="Input, press enter to create type"
        :show-arrow="false"
        :show="false"
        @blur="handleMySQLTypesChange"
        @update:value="handleMySQLTypesChange"
      />
    </div>
    <div class="w-full max-w-lg flex flex-col justify-start items-start gap-2">
      <div class="w-full flex flex-row justify-between items-center">
        <div class="flex flex-row justify-start items-center">
          <EngineIcon :engine="Engine.POSTGRES" :custom-class="'mr-1'" />
          PostgreSQL
        </div>
        <NRadioGroup
          v-model:value="columnTypeTemplateForPostgreSQL.enabled"
          class="gap-x-2"
          :disabled="!hasPermission || !hasFeature"
          @change="handlePostgreSQLEnabledChange"
        >
          <NRadio
            :value="false"
            :label="$t('schema-template.column-type-restriction.allow-all')"
          />
          <NRadio
            :value="true"
            :label="
              $t('schema-template.column-type-restriction.allow-limited-types')
            "
          />
        </NRadioGroup>
      </div>
      <NSelect
        ref="typesSelectorRefForPostgreSQL"
        v-model:value="columnTypeTemplateForPostgreSQL.types"
        filterable
        multiple
        tag
        :disabled="
          !columnTypeTemplateForPostgreSQL.enabled ||
          !hasPermission ||
          !hasFeature
        "
        placeholder="Input, press enter to create type"
        :show-arrow="false"
        :show="false"
        @blur="handlePostgreSQLTypesChange"
        @update:value="handlePostgreSQLTypesChange"
      />
    </div>
  </div>

  <FeatureModal
    feature="bb.feature.schema-template"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />

  <ColumnTypesUpdateFailedModal
    v-if="unmatchedFieldTemplates.length > 0"
    :field-templates="unmatchedFieldTemplates"
    @close="unmatchedFieldTemplates = []"
    @save-all="handleSaveAllUnmatchedFieldTemplates"
  />
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual, uniq, uniqBy } from "lodash-es";
import { NSelect, NRadioGroup, NRadio } from "naive-ui";
import { onMounted, ref } from "vue";
import { Engine } from "@/types/proto/v1/common";
import { featureToRef, pushNotification, useSettingV1Store } from "@/store";
import {
  SchemaTemplateSetting,
  SchemaTemplateSetting_ColumnType,
  SchemaTemplateSetting_FieldTemplate,
} from "@/types/proto/v1/setting_service";
import EngineIcon from "@/components/Icon/EngineIcon.vue";
import ColumnTypesUpdateFailedModal from "./ColumnTypesUpdateFailedModal.vue";
import { useWorkspacePermissionV1 } from "@/utils";
import { useDebounceFn } from "@vueuse/core";

interface LocalState {
  showFeatureModal: boolean;
}

const settingStore = useSettingV1Store();
const state = ref<LocalState>({
  showFeatureModal: false,
});
const hasFeature = featureToRef("bb.feature.schema-template");
const hasPermission = useWorkspacePermissionV1(
  "bb.permission.workspace.manage-general"
);
const columnTypeTemplateForMySQL = ref(
  SchemaTemplateSetting_ColumnType.fromPartial({
    engine: Engine.MYSQL,
  })
);
const columnTypeTemplateForPostgreSQL = ref(
  SchemaTemplateSetting_ColumnType.fromPartial({
    engine: Engine.POSTGRES,
  })
);
// The ref of the naive-ui select component.
const typesSelectorRefForMySQL = ref<InstanceType<typeof NSelect>>();
const typesSelectorRefForPostgreSQL = ref<InstanceType<typeof NSelect>>();
const unmatchedFieldTemplates = ref<SchemaTemplateSetting_FieldTemplate[]>([]);

const getOrFetchSchemaTemplate = async () => {
  const setting = await settingStore.getOrFetchSettingByName(
    "bb.workspace.schema-template"
  );
  const columnTypes =
    setting.value?.schemaTemplateSettingValue?.columnTypes || [];
  const mysqlColumnTypes = columnTypes.find(
    (item) => item.engine === Engine.MYSQL
  );
  const postgresqlColumnTypes = columnTypes.find(
    (item) => item.engine === Engine.POSTGRES
  );
  return {
    fieldTemplates:
      setting.value?.schemaTemplateSettingValue?.fieldTemplates || [],
    mysqlColumnTypes,
    postgresqlColumnTypes,
  };
};

onMounted(async () => {
  const { mysqlColumnTypes, postgresqlColumnTypes } =
    await getOrFetchSchemaTemplate();
  if (mysqlColumnTypes) {
    columnTypeTemplateForMySQL.value = cloneDeep(mysqlColumnTypes);
  }
  if (postgresqlColumnTypes) {
    columnTypeTemplateForPostgreSQL.value = cloneDeep(postgresqlColumnTypes);
  }
});

const handleSaveAllUnmatchedFieldTemplates = (
  fieldTemplates: SchemaTemplateSetting_FieldTemplate[]
) => {
  if (fieldTemplates.length === 0) {
    return;
  }

  const engine = fieldTemplates[0].engine;
  if (engine === Engine.MYSQL) {
    columnTypeTemplateForMySQL.value.types = uniqBy(
      [
        ...columnTypeTemplateForMySQL.value.types,
        ...fieldTemplates.map((item) => item.column?.type || ""),
      ],
      (item) => item
    );
    typesSelectorRefForMySQL.value?.focus();
  } else if (engine === Engine.POSTGRES) {
    columnTypeTemplateForPostgreSQL.value.types = uniqBy(
      [
        ...columnTypeTemplateForPostgreSQL.value.types,
        ...fieldTemplates.map((item) => item.column?.type || ""),
      ],
      (item) => item
    );
    typesSelectorRefForPostgreSQL.value?.focus();
  }
  unmatchedFieldTemplates.value = [];
};

const handleMySQLEnabledChange = (event: InputEvent) => {
  const value = (event.target as HTMLInputElement).value === "true";
  columnTypeTemplateForMySQL.value.enabled = value;
  if (value) {
    typesSelectorRefForMySQL.value?.focus();
    typesSelectorRefForMySQL.value?.triggerRef?.focusInput();
  } else {
    handleMySQLTypesChange();
  }
};

// Update the column types for MySQL in the following cases:
// 1. When the column types are deleted and the selector is not focused.
// 2. When selector is blurred.
const handleMySQLTypesChange = useDebounceFn(async () => {
  if (typesSelectorRefForMySQL.value?.focused) {
    return;
  }

  const { fieldTemplates, mysqlColumnTypes } = await getOrFetchSchemaTemplate();
  if (isEqual(columnTypeTemplateForMySQL.value, mysqlColumnTypes)) {
    return;
  }

  if (
    columnTypeTemplateForMySQL.value.enabled &&
    columnTypeTemplateForMySQL.value.types.length === 0
  ) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Column types cannot be empty when enabled",
    });
    return;
  }

  const mysqlFieldTemplates = uniq(
    fieldTemplates.filter(
      (item) => item.engine === Engine.MYSQL && item.column?.type
    )
  );
  const fieldTemplateTypesOfMySQL = uniq(
    mysqlFieldTemplates
      .map((item) => (item.column?.type || "") as string)
      .filter(Boolean)
  );
  const uncoveredTypes = fieldTemplateTypesOfMySQL.filter(
    (item) => !columnTypeTemplateForMySQL.value.types.includes(item)
  );
  if (uncoveredTypes.length > 0) {
    unmatchedFieldTemplates.value = mysqlFieldTemplates.filter((item) =>
      uncoveredTypes.includes(item.column?.type || "")
    );
    return;
  }

  const setting = await settingStore.getOrFetchSettingByName(
    "bb.workspace.schema-template"
  );
  setting.value!.schemaTemplateSettingValue = SchemaTemplateSetting.fromPartial(
    {
      ...setting.value?.schemaTemplateSettingValue,
      columnTypes: uniqBy(
        [
          columnTypeTemplateForMySQL.value,
          ...(setting.value?.schemaTemplateSettingValue?.columnTypes || []),
        ],
        "engine"
      ),
    }
  );
  await settingStore.upsertSetting({
    name: "bb.workspace.schema-template",
    value: {
      schemaTemplateSettingValue: setting.value?.schemaTemplateSettingValue,
    },
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: "Success to update column types",
  });
}, 1000);

const handlePostgreSQLEnabledChange = (event: InputEvent) => {
  if (!hasFeature.value) {
    state.value.showFeatureModal = true;
    return;
  }

  const value = (event.target as HTMLInputElement).value === "true";
  columnTypeTemplateForPostgreSQL.value.enabled = value;
  if (value) {
    typesSelectorRefForPostgreSQL.value?.focus();
    typesSelectorRefForPostgreSQL.value?.triggerRef?.focusInput();
  } else {
    handlePostgreSQLTypesChange();
  }
};

const handlePostgreSQLTypesChange = useDebounceFn(async () => {
  if (typesSelectorRefForPostgreSQL.value?.focused) {
    return;
  }

  const { fieldTemplates, postgresqlColumnTypes } =
    await getOrFetchSchemaTemplate();
  if (isEqual(columnTypeTemplateForPostgreSQL.value, postgresqlColumnTypes)) {
    return;
  }

  if (
    columnTypeTemplateForPostgreSQL.value.enabled &&
    columnTypeTemplateForPostgreSQL.value.types.length === 0
  ) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Column types cannot be empty when enabled",
    });
    return;
  }

  const postgresFieldTemplates = uniq(
    fieldTemplates.filter(
      (item) => item.engine === Engine.POSTGRES && item.column?.type
    )
  );
  const fieldTemplateTypesOfPostgreSQL = uniq(
    postgresFieldTemplates
      .map((item) => (item.column?.type || "") as string)
      .filter(Boolean)
  );
  const uncoveredTypes = fieldTemplateTypesOfPostgreSQL.filter(
    (item) => !columnTypeTemplateForPostgreSQL.value.types.includes(item)
  );
  if (uncoveredTypes.length > 0) {
    unmatchedFieldTemplates.value = postgresFieldTemplates.filter((item) =>
      uncoveredTypes.includes(item.column?.type || "")
    );
    return;
  }

  const setting = await settingStore.getOrFetchSettingByName(
    "bb.workspace.schema-template"
  );
  setting.value!.schemaTemplateSettingValue = SchemaTemplateSetting.fromPartial(
    {
      ...setting.value?.schemaTemplateSettingValue,
      columnTypes: uniqBy(
        [
          columnTypeTemplateForPostgreSQL.value,
          ...(setting.value?.schemaTemplateSettingValue?.columnTypes || []),
        ],
        "engine"
      ),
    }
  );
  await settingStore.upsertSetting({
    name: "bb.workspace.schema-template",
    value: {
      schemaTemplateSettingValue: setting.value?.schemaTemplateSettingValue,
    },
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: "Success to update column types",
  });
}, 1000);
</script>
