<template>
  <SQLCheckButton
    v-if="database"
    :database="database"
    :statement="statement"
    :errors="errors"
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
import { asyncComputed } from "@vueuse/core";
import { cloneDeep, debounce } from "lodash-es";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { SQLCheckButton, SQLCheckSummary } from "@/components/SQLCheck";
import {
  mergeSchemaEditToMetadata,
  validateDatabaseMetadata,
} from "@/components/SchemaEditorV1/utils";
import { schemaDesignServiceClient } from "@/grpcweb";
import { useDatabaseV1Store, useSchemaEditorV1Store } from "@/store";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
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

const sourceMetadata = asyncComputed(async () => {
  const branch = props.schemaDesign;

  return await fetchBaselineMetadataOfBranch(branch);
}, undefined);

const editingMetadata = asyncComputed(async () => {
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
}, undefined);

const targetMetadata = ref<DatabaseMetadata>();
const statement = ref("");
const errors = ref<string[]>([]);

const evaluateTargetMetadataAndDiff = async () => {
  const metadata = editingMetadata.value;

  const setState = (
    metadata: DatabaseMetadata | undefined,
    stmt: string,
    errs: string[]
  ) => {
    targetMetadata.value = metadata;
    statement.value = stmt;
    errors.value = errs;
  };

  if (!metadata) {
    setState(undefined, "", [t("schema-editor.message.invalid-schema")]);
    return;
  }
  const validationMessages = validateDatabaseMetadata(metadata);
  if (validationMessages.length > 0) {
    setState(undefined, "", validationMessages);
    return;
  }

  // Prepare to diff
  setState(metadata, "", []);
  const db = database.value;
  const source = sourceMetadata.value;
  const target = targetMetadata.value;
  if (!db) return;
  if (!source) return;
  if (!target) return;

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
    setState(metadata, diff, errs);
  } catch {
    // The grpc error message is too long not readable. So we won't use it here.
    setState(metadata, "", [t("schema-editor.message.invalid-schema")]);
  }
};

watch(
  editingMetadata,
  (metadata) => {
    if (!metadata) {
      errors.value = [t("schema-editor.message.invalid-schema")];
    } else {
      errors.value = validateDatabaseMetadata(metadata);
    }
  },
  {
    immediate: true,
  }
);

const watchKey = computed(() => {
  return [
    database.value?.name,
    JSON.stringify(
      DatabaseMetadata.toJSON(
        sourceMetadata.value ?? DatabaseMetadata.fromPartial({})
      )
    ),
    JSON.stringify(
      DatabaseMetadata.toJSON(
        editingMetadata.value ?? DatabaseMetadata.fromPartial({})
      )
    ),
  ].join("\n");
});

evaluateTargetMetadataAndDiff();
// Won't update too frequently since this costs pretty high.
watch(watchKey, debounce(evaluateTargetMetadataAndDiff, 250));
</script>
