<template>
  <div class="w-full space-y-4 text-sm">
    <div class="space-y-4">
      <div class="flex items-center justify-between gap-x-6">
        <div class="flex-1 textinfolabel">
          {{ $t("schema-template.column-type-restriction.description") }}
        </div>
      </div>
    </div>
    <div class="w-full flex flex-col justify-start items-start gap-2">
      <div class="w-full flex flex-row justify-between items-center">
        <div class="flex flex-row justify-start items-center">
          <EngineIcon :engine="Engine.MYSQL" :custom-class="'mr-1'" />
          MySQL
        </div>
        <NRadioGroup
          v-model:value="columnTypeTemplateForMySQL.enabled"
          class="gap-x-2"
          :disabled="readonly"
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
      <NInput
        v-if="columnTypeTemplateForMySQL.enabled"
        v-model:value="columnTypesForMySQL"
        :disabled="readonly"
        type="textarea"
        :placeholder="
          $t(
            'schema-template.column-type-restriction.messages.one-allowed-type-per-line'
          )
        "
        :autosize="{
          minRows: 5,
          maxRows: 10,
        }"
      />
      <div class="w-full flex flex-row justify-end items-center mt-2">
        <NButton
          type="primary"
          :disabled="!allowToUpdateColumnTypeTemplateForMySQL"
          @click="handleMySQLTypesChange"
        >
          {{ $t("common.update") }}
        </NButton>
      </div>
    </div>
    <NDivider class="w-full py-2" />
    <div class="w-full flex flex-col justify-start items-start gap-2">
      <div class="w-full flex flex-row justify-between items-center">
        <div class="flex flex-row justify-start items-center">
          <EngineIcon :engine="Engine.POSTGRES" :custom-class="'mr-1'" />
          PostgreSQL
        </div>
        <NRadioGroup
          v-model:value="columnTypeTemplateForPostgreSQL.enabled"
          class="gap-x-2"
          :disabled="readonly"
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
      <NInput
        v-if="columnTypeTemplateForPostgreSQL.enabled"
        v-model:value="columnTypesForPostgreSQL"
        :disabled="readonly"
        type="textarea"
        :placeholder="
          $t(
            'schema-template.column-type-restriction.messages.one-allowed-type-per-line'
          )
        "
        :autosize="{
          minRows: 5,
          maxRows: 10,
        }"
      />
      <div class="w-full flex flex-row justify-end items-center mt-2">
        <NButton
          type="primary"
          :disabled="!allowToUpdateColumnTypeTemplateForPostgreSQL"
          @click="handlePostgreSQLTypesChange"
          >{{ $t("common.update") }}</NButton
        >
      </div>
    </div>
  </div>

  <ColumnTypesUpdateFailedModal
    v-if="unmatchedFieldTemplates.length > 0"
    :field-templates="unmatchedFieldTemplates"
    @close="unmatchedFieldTemplates = []"
    @save-all="handleSaveAllUnmatchedFieldTemplates"
  />
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { cloneDeep, isEqual, uniq, uniqBy } from "lodash-es";
import { NButton, NDivider, NInput, NRadioGroup, NRadio } from "naive-ui";
import { computed, onMounted, ref } from "vue";
import EngineIcon from "@/components/Icon/EngineIcon.vue";
import { pushNotification, useSettingV1Store } from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  SchemaTemplateSetting_FieldTemplate,
  SchemaTemplateSetting_ColumnType,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  SchemaTemplateSetting_ColumnTypeSchema,
  SchemaTemplateSettingSchema,
  Setting_SettingName,
  ValueSchema as SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { getDataTypeSuggestionList } from "@/utils";
import ColumnTypesUpdateFailedModal from "./ColumnTypesUpdateFailedModal.vue";

const props = defineProps<{
  readonly?: boolean;
}>();

const settingStore = useSettingV1Store();
const columnTypeTemplateForMySQL = ref(
  create(SchemaTemplateSetting_ColumnTypeSchema, {
    engine: Engine.MYSQL,
  })
);
const columnTypesForMySQL = ref<string>("");
const columnTypeTemplateForPostgreSQL = ref(
  create(SchemaTemplateSetting_ColumnTypeSchema, {
    engine: Engine.POSTGRES,
  })
);
const columnTypesForPostgreSQL = ref<string>("");
const unmatchedFieldTemplates = ref<SchemaTemplateSetting_FieldTemplate[]>([]);

const allowToUpdateColumnTypeTemplateForMySQL = computed(() => {
  if (props.readonly) {
    return false;
  }
  const setting = settingStore.getSettingByName(
    Setting_SettingName.SCHEMA_TEMPLATE
  );
  const columnTypes =
    setting?.value?.value?.case === "schemaTemplateSettingValue"
      ? setting.value.value.value.columnTypes || []
      : [];
  const existingTemplate = columnTypes.find(
    (item) => item.engine === Engine.MYSQL
  );
  const originTemplate = existingTemplate
    ? create(SchemaTemplateSetting_ColumnTypeSchema, {
        engine: existingTemplate.engine,
        enabled: existingTemplate.enabled,
        types: existingTemplate.types || [],
      })
    : create(SchemaTemplateSetting_ColumnTypeSchema, {
        engine: Engine.MYSQL,
        enabled: false,
        types: [],
      });
  const newTemplate = create(SchemaTemplateSetting_ColumnTypeSchema, {
    engine: Engine.MYSQL,
    enabled: columnTypeTemplateForMySQL.value.enabled,
    types: columnTypesForMySQL.value
      .split("\n")
      .map((item) => item.trim())
      .filter(Boolean),
  });
  if (isEqual(originTemplate.enabled, newTemplate.enabled)) {
    if (!newTemplate.enabled) {
      return false;
    }
    if (isEqual(originTemplate.types, newTemplate.types)) {
      return false;
    }
  }
  return true;
});

const allowToUpdateColumnTypeTemplateForPostgreSQL = computed(() => {
  if (props.readonly) {
    return false;
  }
  const setting = settingStore.getSettingByName(
    Setting_SettingName.SCHEMA_TEMPLATE
  );
  const columnTypes =
    setting?.value?.value?.case === "schemaTemplateSettingValue"
      ? setting.value.value.value.columnTypes || []
      : [];
  const existingTemplate = columnTypes.find(
    (item) => item.engine === Engine.POSTGRES
  );
  const originTemplate = existingTemplate
    ? create(SchemaTemplateSetting_ColumnTypeSchema, {
        engine: existingTemplate.engine,
        enabled: existingTemplate.enabled,
        types: existingTemplate.types || [],
      })
    : create(SchemaTemplateSetting_ColumnTypeSchema, {
        engine: Engine.POSTGRES,
        enabled: false,
        types: [],
      });
  const newTemplate = create(SchemaTemplateSetting_ColumnTypeSchema, {
    engine: Engine.POSTGRES,
    enabled: columnTypeTemplateForPostgreSQL.value.enabled,
    types: columnTypesForPostgreSQL.value
      .split("\n")
      .map((item) => item.trim())
      .filter(Boolean),
  });
  if (isEqual(originTemplate.enabled, newTemplate.enabled)) {
    if (!newTemplate.enabled) {
      return false;
    }
    if (isEqual(originTemplate.types, newTemplate.types)) {
      return false;
    }
  }
  return true;
});

const getOrFetchSchemaTemplate = async () => {
  const setting = await settingStore.getOrFetchSettingByName(
    Setting_SettingName.SCHEMA_TEMPLATE
  );
  const columnTypes =
    setting?.value?.value?.case === "schemaTemplateSettingValue"
      ? setting.value.value.value.columnTypes || []
      : [];
  const mysqlColumnTypes = columnTypes.find(
    (item) => item.engine === Engine.MYSQL
  );
  const postgresqlColumnTypes = columnTypes.find(
    (item) => item.engine === Engine.POSTGRES
  );
  return {
    fieldTemplates:
      setting?.value?.value?.case === "schemaTemplateSettingValue"
        ? setting.value.value.value.fieldTemplates || []
        : [],
    mysqlColumnTypes,
    postgresqlColumnTypes,
  };
};

onMounted(async () => {
  const { mysqlColumnTypes, postgresqlColumnTypes } =
    await getOrFetchSchemaTemplate();
  if (mysqlColumnTypes) {
    columnTypeTemplateForMySQL.value = cloneDeep(mysqlColumnTypes);
    columnTypesForMySQL.value =
      columnTypeTemplateForMySQL.value.types.join("\n");
  }
  if (postgresqlColumnTypes) {
    columnTypeTemplateForPostgreSQL.value = cloneDeep(postgresqlColumnTypes);
    columnTypesForPostgreSQL.value =
      columnTypeTemplateForPostgreSQL.value.types.join("\n");
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
    columnTypesForMySQL.value =
      columnTypesForMySQL.value +
      "\n" +
      uniq(fieldTemplates.map((item) => item.column?.type || "")).join("\n");
    handleMySQLTypesChange();
  } else if (engine === Engine.POSTGRES) {
    columnTypesForPostgreSQL.value =
      columnTypesForPostgreSQL.value +
      "\n" +
      uniq(fieldTemplates.map((item) => item.column?.type || "")).join("\n");
    handlePostgreSQLTypesChange();
  }
  unmatchedFieldTemplates.value = [];
};

const handleMySQLEnabledChange = (event: InputEvent) => {
  const enabled = (event.target as HTMLInputElement).value === "true";
  columnTypeTemplateForMySQL.value.enabled = enabled;
  if (enabled) {
    if (columnTypeTemplateForMySQL.value.types.filter(Boolean).length === 0) {
      columnTypesForMySQL.value = getDataTypeSuggestionList(Engine.MYSQL).join(
        "\n"
      );
    }
  }
};

const handleMySQLTypesChange = async () => {
  columnTypesForMySQL.value = columnTypesForMySQL.value
    .toLowerCase()
    .split("\n")
    .map((item) => item.trim())
    .filter(Boolean)
    .join("\n");
  columnTypeTemplateForMySQL.value.types =
    columnTypesForMySQL.value.split("\n");

  const { fieldTemplates } = await getOrFetchSchemaTemplate();
  const mysqlFieldTemplates = uniq(
    fieldTemplates.filter(
      (item) => item.engine === Engine.MYSQL && item.column?.type
    )
  );
  const fieldTemplateTypesOfMySQL = uniq(
    mysqlFieldTemplates
      .map((item) => item.column?.type || "")
      .filter(Boolean)
      .map((item) => item.toLowerCase())
  );
  if (columnTypeTemplateForMySQL.value.enabled) {
    // Check if there is any field template that is not covered by the column types.
    if (columnTypeTemplateForMySQL.value.types.length === 0) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Column types cannot be empty when enabled",
      });
      return;
    }

    // Check if there is any field template that is not covered by the column types.
    const uncoveredTypes = fieldTemplateTypesOfMySQL.filter(
      (item) =>
        !columnTypeTemplateForMySQL.value.types.includes(item.toLowerCase())
    );
    if (uncoveredTypes.length > 0) {
      unmatchedFieldTemplates.value = mysqlFieldTemplates.filter((item) =>
        uncoveredTypes.includes((item.column?.type || "").toLowerCase())
      );
      return;
    }
  }

  await upsertSchemaTemplateSetting(columnTypeTemplateForMySQL.value);

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: "Success to update column types",
  });
};

const handlePostgreSQLEnabledChange = (event: InputEvent) => {
  const enabled = (event.target as HTMLInputElement).value === "true";
  columnTypeTemplateForPostgreSQL.value.enabled = enabled;
  if (enabled) {
    if (
      columnTypeTemplateForPostgreSQL.value.types.filter(Boolean).length === 0
    ) {
      columnTypesForPostgreSQL.value = getDataTypeSuggestionList(
        Engine.POSTGRES
      ).join("\n");
    }
  }
};

const handlePostgreSQLTypesChange = async () => {
  columnTypesForPostgreSQL.value = columnTypesForPostgreSQL.value
    .toLowerCase()
    .split("\n")
    .map((item) => item.trim())
    .filter(Boolean)
    .join("\n");
  columnTypeTemplateForPostgreSQL.value.types =
    columnTypesForPostgreSQL.value.split("\n");

  const { fieldTemplates, postgresqlColumnTypes } =
    await getOrFetchSchemaTemplate();
  if (isEqual(columnTypeTemplateForPostgreSQL.value, postgresqlColumnTypes)) {
    return;
  }

  const postgresFieldTemplates = uniq(
    fieldTemplates.filter(
      (item) => item.engine === Engine.POSTGRES && item.column?.type
    )
  );
  const fieldTemplateTypesOfPostgreSQL = uniq(
    postgresFieldTemplates
      .map((item) => item.column?.type || "")
      .filter(Boolean)
      .map((item) => item.toLowerCase())
  );
  if (columnTypeTemplateForPostgreSQL.value.enabled) {
    // Check if there is any field template that is not covered by the column types.
    if (columnTypeTemplateForPostgreSQL.value.types.length === 0) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Column types cannot be empty when enabled",
      });
      return;
    }

    // Check if there is any field template that is not covered by the column types.
    const uncoveredTypes = fieldTemplateTypesOfPostgreSQL.filter(
      (item) =>
        !columnTypeTemplateForPostgreSQL.value.types.includes(
          item.toLowerCase()
        )
    );
    if (uncoveredTypes.length > 0) {
      unmatchedFieldTemplates.value = postgresFieldTemplates.filter((item) =>
        uncoveredTypes.includes((item.column?.type || "").toLowerCase())
      );
      return;
    }
  }

  await upsertSchemaTemplateSetting(columnTypeTemplateForPostgreSQL.value);

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: "Success to update column types",
  });
};

const upsertSchemaTemplateSetting = async (
  columnType: SchemaTemplateSetting_ColumnType
) => {
  const setting = await settingStore.getOrFetchSettingByName(
    Setting_SettingName.SCHEMA_TEMPLATE
  );
  const existingValue =
    setting?.value?.value?.case === "schemaTemplateSettingValue"
      ? setting.value.value.value
      : undefined;
  const schemaTemplateSettingValue = create(SchemaTemplateSettingSchema, {
    fieldTemplates: existingValue?.fieldTemplates || [],
    tableTemplates: existingValue?.tableTemplates || [],
    columnTypes: uniqBy(
      [columnType, ...(existingValue?.columnTypes || [])],
      "engine"
    ),
  });
  await settingStore.upsertSetting({
    name: Setting_SettingName.SCHEMA_TEMPLATE,
    value: create(SettingValueSchema, {
      value: {
        case: "schemaTemplateSettingValue",
        value: schemaTemplateSettingValue,
      },
    }),
  });
};
</script>
