<template>
  <div class="w-full space-y-4 text-sm">
    <FeatureAttention
      feature="bb.feature.schema-template"
      custom-class="my-4"
    />
    <div class="space-y-4">
      <div class="flex items-center justify-between gap-x-6">
        <div class="flex-1 textinfolabel !leading-8">
          You can restrict the allowed column types for each database engine.
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
          @change="handleMySQLEnabledChange"
        >
          <NRadio :value="false" :label="'Allow all'" />
          <NRadio :value="true" :label="'Allow limited types'" />
        </NRadioGroup>
      </div>
      <NSelect
        ref="typesSelectorRefForMySQL"
        v-model:value="columnTypeTemplateForMySQL.types"
        filterable
        multiple
        tag
        :show-on-focus="true"
        :disabled="!columnTypeTemplateForMySQL.enabled"
        placeholder="Input, press enter to create type"
        :show-arrow="false"
        :show="false"
        @blur="handleMySQLTypesChange"
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
          @change="handlePostgreSQLEnabledChange"
        >
          <NRadio :value="false" :label="'Allow all'" />
          <NRadio :value="true" :label="'Allow limited types'" />
        </NRadioGroup>
      </div>
      <NSelect
        ref="typesSelectorRefForPostgreSQL"
        v-model:value="columnTypeTemplateForPostgreSQL.types"
        filterable
        multiple
        tag
        :disabled="!columnTypeTemplateForPostgreSQL.enabled"
        placeholder="Input, press enter to create type"
        :show-arrow="false"
        :show="false"
        @blur="handlePostgreSQLTypesChange"
      />
    </div>
  </div>

  <ColumnTypesUpdateFailedModal
    v-if="unmatchedFieldTemplates.length > 0"
    :field-templates="unmatchedFieldTemplates"
    @close="unmatchedFieldTemplates = []"
  />
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual, uniq, uniqBy } from "lodash-es";
import { NSelect, NRadioGroup, NRadio } from "naive-ui";
import { onMounted, ref } from "vue";
import { Engine } from "@/types/proto/v1/common";
import { pushNotification, useSettingV1Store } from "@/store";
import {
  SchemaTemplateSetting,
  SchemaTemplateSetting_ColumnType,
  SchemaTemplateSetting_FieldTemplate,
} from "@/types/proto/v1/setting_service";
import EngineIcon from "@/components/Icon/EngineIcon.vue";
import ColumnTypesUpdateFailedModal from "./ColumnTypesUpdateFailedModal.vue";

const settingStore = useSettingV1Store();
const fieldTemplates = ref<SchemaTemplateSetting_FieldTemplate[]>([]);
const originColumnTypeTemplateForMySQL = ref(
  SchemaTemplateSetting_ColumnType.fromPartial({
    engine: Engine.MYSQL,
  })
);
const originColumnTypeTemplateForPostgreSQL = ref(
  SchemaTemplateSetting_ColumnType.fromPartial({
    engine: Engine.POSTGRES,
  })
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

onMounted(async () => {
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
  if (mysqlColumnTypes) {
    originColumnTypeTemplateForMySQL.value = mysqlColumnTypes;
    columnTypeTemplateForMySQL.value = cloneDeep(mysqlColumnTypes);
  }
  if (postgresqlColumnTypes) {
    originColumnTypeTemplateForPostgreSQL.value = postgresqlColumnTypes;
    columnTypeTemplateForPostgreSQL.value = cloneDeep(postgresqlColumnTypes);
  }
  fieldTemplates.value =
    setting.value?.schemaTemplateSettingValue?.fieldTemplates || [];
});

const handleMySQLEnabledChange = (event: InputEvent) => {
  const value = (event.target as HTMLInputElement).value === "true";
  columnTypeTemplateForMySQL.value.enabled = value;
  if (value) {
    typesSelectorRefForMySQL.value?.focus();
  } else {
    handleMySQLTypesChange();
  }
};

const handleMySQLTypesChange = async () => {
  if (
    isEqual(
      columnTypeTemplateForMySQL.value,
      originColumnTypeTemplateForMySQL.value
    )
  ) {
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

  const fieldTemplateTypesOfMySQL = uniq(
    fieldTemplates.value
      .filter((item) => item.engine === Engine.MYSQL && item.column?.type)
      .map((item) => (item.column?.type || "") as string)
      .filter(Boolean)
  );
  const uncoveredTypes = fieldTemplateTypesOfMySQL.filter(
    (item) => !columnTypeTemplateForMySQL.value.types.includes(item)
  );
  if (uncoveredTypes.length > 0) {
    unmatchedFieldTemplates.value = fieldTemplates.value.filter((item) =>
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
};

const handlePostgreSQLEnabledChange = (event: InputEvent) => {
  const value = (event.target as HTMLInputElement).value === "true";
  columnTypeTemplateForPostgreSQL.value.enabled = value;
  if (value) {
    typesSelectorRefForPostgreSQL.value?.focus();
  } else {
    handlePostgreSQLTypesChange();
  }
};

const handlePostgreSQLTypesChange = async () => {
  if (
    isEqual(
      columnTypeTemplateForPostgreSQL.value,
      originColumnTypeTemplateForPostgreSQL.value
    )
  ) {
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

  const fieldTemplateTypesOfPostgreSQL = uniq(
    fieldTemplates.value
      .filter((item) => item.engine === Engine.POSTGRES && item.column?.type)
      .map((item) => (item.column?.type || "") as string)
      .filter(Boolean)
  );
  const uncoveredTypes = fieldTemplateTypesOfPostgreSQL.filter(
    (item) => !columnTypeTemplateForPostgreSQL.value.types.includes(item)
  );
  if (uncoveredTypes.length > 0) {
    unmatchedFieldTemplates.value = fieldTemplates.value.filter((item) =>
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
};
</script>
