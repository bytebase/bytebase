<template>
  <SQLCheckButton
    v-if="database"
    :database="database"
    :get-statement="getStatement"
    class="justify-end"
    :button-style="{
      height: '28px',
    }"
  >
    <template #result="{ advices, isRunning }">
      <SQLCheckSummary
        v-if="advices !== undefined && !isRunning"
        :database="database"
        :advices="advices"
      />
    </template>
  </SQLCheckButton>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { SQLCheckButton, SQLCheckSummary } from "@/components/SQLCheck";
import {
  mergeSchemaEditToMetadata,
  validateDatabaseMetadata,
} from "@/components/SchemaEditorV1/utils";
import { schemaDesignServiceClient } from "@/grpcweb";
import { useDatabaseV1Store, useSchemaEditorV1Store } from "@/store";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import { fetchBaselineMetadataOfBranch } from "../SchemaEditorV1/utils/branch";

const props = defineProps<{
  schemaDesign: SchemaDesign;
}>();
const { t } = useI18n();
const databaseStore = useDatabaseV1Store();
const schemaEditorV1Store = useSchemaEditorV1Store();

const database = computed(() => {
  return databaseStore.getDatabaseByName(props.schemaDesign.baselineDatabase);
});

const getSourceMetadata = async () => {
  const branch = props.schemaDesign;

  return await fetchBaselineMetadataOfBranch(branch);
};
const getEditingMetadata = async () => {
  const branchSchema = schemaEditorV1Store.resourceMap["branch"].get(
    props.schemaDesign.name
  );
  if (!branchSchema) {
    return undefined;
  }
  const baselineMetadata = await fetchBaselineMetadataOfBranch(
    branchSchema.branch
  );
  const metadata = mergeSchemaEditToMetadata(
    branchSchema.schemaList,
    cloneDeep(baselineMetadata)
  );

  return metadata;
};

const getStatement = async () => {
  const editingMetadata = await getEditingMetadata();

  if (!editingMetadata) {
    return {
      statement: "",
      errors: [t("schema-editor.message.invalid-schema")],
    };
  }
  const validationMessages = validateDatabaseMetadata(editingMetadata);
  if (validationMessages.length > 0) {
    return {
      statement: "",
      errors: validationMessages,
    };
  }

  // Prepare to diff
  const db = database.value;
  const source = await getSourceMetadata();
  const target = editingMetadata;

  try {
    const diffResponse = await schemaDesignServiceClient.diffMetadata(
      {
        sourceMetadata: source,
        targetMetadata: target,
        engine: db.instanceEntity.engine,
      },
      {
        silent: true,
      }
    );
    const diff = diffResponse.diff;
    const errs = diff.length === 0 ? [t("schema-editor.nothing-changed")] : [];
    return {
      statement: diff,
      errors: errs,
    };
  } catch {
    // The grpc error message is too long not readable. So we won't use it here.
    return {
      statement: "",
      errors: [t("schema-editor.message.invalid-schema")],
    };
  }
};
</script>
