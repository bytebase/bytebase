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
      <NInput
        v-if="columnTypeTemplateForMySQL.enabled"
        v-model:value="columnTypesForMySQL"
        :disabled="!hasPermission || !hasFeature"
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
      <div class="w-full flex flex-row justify-end items-center">
        <NButton
          type="primary"
          :disabled="
            !hasPermission ||
            !hasFeature ||
            !allowToUpdateColumnTypeTemplateForMySQL
          "
          @click="handleMySQLTypesChange"
        >
          {{ $t("common.update") }}
        </NButton>
      </div>
    </div>
    <NDivider class="w-full max-w-lg" />
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
      <NInput
        v-if="hasFeature && columnTypeTemplateForPostgreSQL.enabled"
        v-model:value="columnTypesForPostgreSQL"
        :disabled="!hasPermission || !hasFeature"
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
      <div class="w-full flex flex-row justify-end items-center">
        <NButton
          type="primary"
          :disabled="
            !hasPermission ||
            !hasFeature ||
            !allowToUpdateColumnTypeTemplateForPostgreSQL
          "
          @click="handlePostgreSQLTypesChange"
          >{{ $t("common.update") }}</NButton
        >
      </div>
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
import { NButton, NDivider, NInput, NRadioGroup, NRadio } from "naive-ui";
import { computed, onMounted, ref } from "vue";
import EngineIcon from "@/components/Icon/EngineIcon.vue";
import { featureToRef, pushNotification, useSettingV1Store } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import {
  SchemaTemplateSetting,
  SchemaTemplateSetting_ColumnType,
  SchemaTemplateSetting_FieldTemplate,
} from "@/types/proto/v1/setting_service";
import { getDataTypeSuggestionList, useWorkspacePermissionV1 } from "@/utils";
import ColumnTypesUpdateFailedModal from "./ColumnTypesUpdateFailedModal.vue";

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
const columnTypesForMySQL = ref<string>("");
const columnTypeTemplateForPostgreSQL = ref(
  SchemaTemplateSetting_ColumnType.fromPartial({
    engine: Engine.POSTGRES,
  })
);
const columnTypesForPostgreSQL = ref<string>("");
const unmatchedFieldTemplates = ref<SchemaTemplateSetting_FieldTemplate[]>([]);

const allowToUpdateColumnTypeTemplateForMySQL = computed(() => {
  if (!hasFeature.value || !hasPermission.value) {
    return false;
  }
  const setting = settingStore.getSettingByName("bb.workspace.schema-template");
  const columnTypes =
    setting?.value?.schemaTemplateSettingValue?.columnTypes || [];
  const originTemplate = SchemaTemplateSetting_ColumnType.fromPartial({
    engine: Engine.MYSQL,
    ...columnTypes.find((item) => item.engine === Engine.MYSQL),
  });
  const newTemplate = SchemaTemplateSetting_ColumnType.fromPartial({
    ...columnTypeTemplateForMySQL.value,
    engine: Engine.MYSQL,
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
  if (!hasFeature.value || !hasPermission.value) {
    return false;
  }
  const setting = settingStore.getSettingByName("bb.workspace.schema-template");
  const columnTypes =
    setting?.value?.schemaTemplateSettingValue?.columnTypes || [];
  const originTemplate = SchemaTemplateSetting_ColumnType.fromPartial({
    engine: Engine.POSTGRES,
    ...columnTypes.find((item) => item.engine === Engine.POSTGRES),
  });
  const newTemplate = SchemaTemplateSetting_ColumnType.fromPartial({
    ...columnTypeTemplateForPostgreSQL.value,
    engine: Engine.POSTGRES,
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
      fieldTemplates.map((item) => item.column?.type || "").join("\n");
    handleMySQLTypesChange();
  } else if (engine === Engine.POSTGRES) {
    columnTypesForPostgreSQL.value =
      columnTypesForPostgreSQL.value +
      "\n" +
      fieldTemplates.map((item) => item.column?.type || "").join("\n");
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
    .toUpperCase()
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
      .map((item) => (item.column?.type || "") as string)
      .filter(Boolean)
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
      (item) => !columnTypeTemplateForMySQL.value.types.includes(item)
    );
    if (uncoveredTypes.length > 0) {
      unmatchedFieldTemplates.value = mysqlFieldTemplates.filter((item) =>
        uncoveredTypes.includes(item.column?.type || "")
      );
      return;
    }
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
  if (!hasFeature.value) {
    state.value.showFeatureModal = true;
    return;
  }

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
    .toUpperCase()
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
      .map((item) => (item.column?.type || "") as string)
      .filter(Boolean)
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
      (item) => !columnTypeTemplateForPostgreSQL.value.types.includes(item)
    );
    if (uncoveredTypes.length > 0) {
      unmatchedFieldTemplates.value = postgresFieldTemplates.filter((item) =>
        uncoveredTypes.includes(item.column?.type || "")
      );
      return;
    }
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
