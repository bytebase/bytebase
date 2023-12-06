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
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { SQLCheckButton, SQLCheckSummary } from "@/components/SQLCheck";
import { validateDatabaseMetadata } from "@/components/SchemaEditorLite/utils";
import { branchServiceClient } from "@/grpcweb";
import { useDatabaseV1Store } from "@/store";
import { Branch } from "@/types/proto/v1/branch_service";

const props = defineProps<{
  branch: Branch;
}>();
const { t } = useI18n();
const databaseStore = useDatabaseV1Store();

const database = computed(() => {
  return databaseStore.getDatabaseByName(props.branch.baselineDatabase);
});

const getStatement = async () => {
  const source = props.branch.baselineSchemaMetadata;
  const editing = props.branch.schemaMetadata;
  if (!editing) {
    return {
      statement: "",
      errors: [t("schema-editor.message.invalid-schema")],
    };
  }
  const validationMessages = validateDatabaseMetadata(editing);
  if (validationMessages.length > 0) {
    return {
      statement: "",
      errors: validationMessages,
    };
  }

  // Prepare to diff
  const db = database.value;

  try {
    const diffResponse = await branchServiceClient.diffMetadata(
      {
        sourceMetadata: source,
        targetMetadata: editing,
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
  } catch (ex) {
    // The grpc error message is too long not readable. So we won't use it here.
    return {
      statement: "",
      errors: [t("schema-editor.message.invalid-schema")],
    };
  }
};
</script>
