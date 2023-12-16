<template>
  <div
    class="flex flex-col gap-y-4 w-full h-full overflow-x-auto relative p-1"
    v-bind="$attrs"
  >
    <div class="flex flex-col gap-y-2">
      <div class="textlabel">
        {{ $t("branch.rollout.select-target-database") }}
      </div>
      <div class="flex flex-row items-center gap-x-4">
        <div class="flex flex-row items-center gap-x-2">
          <div class="textlabel">{{ $t("common.environment") }}</div>
          <EnvironmentSelect
            :environment="environment?.uid"
            style="width: 8rem"
            @update:environment="handleSelectEnvironment"
          />
        </div>
        <div class="flex flex-row items-center gap-x-2">
          <div class="textlabel">{{ $t("common.database") }}</div>
          <DatabaseSelect
            :database="database?.uid"
            :project="project.uid"
            :environment="environment?.uid"
            style="width: 16rem"
            @update:database="handleSelectDatabase"
          />
        </div>
      </div>
    </div>
    <div class="flex-1 overflow-hidden relative">
      <SchemaEditorLite
        v-if="database"
        :key="virtualBranch.name"
        ref="schemaEditorRef"
        v-model:selected-rollout-objects="selectedRolloutObjects"
        :project="project"
        :readonly="true"
        :resource-type="'branch'"
        :branch="virtualBranch"
        :loading="isLoadingVirtualBranch"
        :diff-when-ready="true"
      />

      <!-- used as a placeholder -->
      <template v-else>
        <SchemaEditorLite
          :project="project"
          :readonly="true"
          :resource-type="'branch'"
          :branch="emptyBranch"
        />
        <div
          class="absolute inset-0 bg-white/75 text-sm flex flex-col items-center justify-center"
        >
          {{ $t("branch.rollout.select-target-database") }}
        </div>
      </template>
    </div>
    <div class="flex flex-row items-center justify-between">
      <div class="flex flex-row items-center justify-start"></div>
      <div class="flex flex-row items-center justify-end">
        <NButton
          type="primary"
          :disabled="!allowPreviewIssue"
          @click="handlePreviewIssue"
        >
          {{ $t("issue.preview") }}
        </NButton>
      </div>
    </div>

    <MaskSpinner
      v-if="virtualBranchReady && isGeneratingDDL"
      class="!bg-white/75"
    >
      <div class="text-sm">Generating DDL</div>
    </MaskSpinner>
  </div>

  <div
    class="text-xs font-mono fixed bottom-0 left-0 bg-white/50 border p-1 max-w-[40rem]"
  >
    <div>ready: {{ virtualBranchReady }}</div>
    <div>
      selectedRolloutObjects.length: {{ selectedRolloutObjects.length }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { cloneDeep } from "lodash-es";
import { NButton } from "naive-ui";
import { computed, ref, toRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import SchemaEditorLite, {
  RolloutObject,
  generateDiffDDL,
} from "@/components/SchemaEditorLite";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { DatabaseSelect, EnvironmentSelect } from "@/components/v2";
import {
  pushNotification,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useSheetV1Store,
} from "@/store";
import { ComposedDatabase, ComposedProject, UNKNOWN_ID } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { Environment } from "@/types/proto/v1/environment_service";
import {
  Sheet,
  SheetPayload_Type,
  Sheet_Source,
  Sheet_Type,
  Sheet_Visibility,
} from "@/types/proto/v1/sheet_service";
import { extractSheetUID, setSheetStatement } from "@/utils";
import { useVirtualBranch } from "./useVirtualBranch";

const props = defineProps<{
  project: ComposedProject;
  branch: Branch;
}>();

const { t } = useI18n();
const router = useRouter();
const schemaEditorRef = ref<InstanceType<typeof SchemaEditorLite>>();
const environment = ref<Environment>();
const database = ref<ComposedDatabase>();
const {
  isLoading: isLoadingVirtualBranch,
  ready: virtualBranchReady,
  branch: virtualBranch,
} = useVirtualBranch(toRef(props, "project"), toRef(props, "branch"), database);
const selectedRolloutObjects = ref<RolloutObject[]>([]);
const emptyBranch = Branch.fromJSON({});
const isGeneratingDDL = ref(false);

const handleSelectEnvironment = (uid: string | undefined) => {
  if (!uid || uid === String(UNKNOWN_ID)) {
    environment.value = undefined;
    return;
  }
  environment.value = useEnvironmentV1Store().getEnvironmentByUID(uid);
  if (database.value) {
    if (database.value.effectiveEnvironment !== environment.value.name) {
      // de-select database since environment changed
      handleSelectDatabase(undefined);
    }
  }
};
const handleSelectDatabase = (uid: string | undefined) => {
  if (!uid || uid === String(UNKNOWN_ID)) {
    database.value = undefined;
    return;
  }
  database.value = useDatabaseV1Store().getDatabaseByUID(uid);
  if (!environment.value) {
    // Auto select environment if it's not selected
    environment.value = database.value.effectiveEnvironmentEntity;
  }
};

const allowPreviewIssue = computed(() => {
  if (!database.value) return false;
  if (!virtualBranchReady.value) return false;
  return selectedRolloutObjects.value.length > 0;
});

const handlePreviewIssue = async () => {
  const cleanup = (errors: string[], fatal: boolean) => {
    if (errors.length > 0) {
      pushNotification({
        module: "bytebase",
        style: fatal ? "CRITICAL" : "WARN",
        title: t("common.error"),
        description: errors.join("\n"),
      });
    }
    isGeneratingDDL.value = false;
  };

  const source = cloneDeep(virtualBranch.value.baselineSchemaMetadata);
  const target = cloneDeep(virtualBranch.value.schemaMetadata);
  const db = database.value;
  const editor = schemaEditorRef.value;
  if (!source) return;
  if (!target) return;
  if (!db) return;
  if (!editor) return;
  editor.applySelectedMetadataEdit(
    db,
    source,
    target,
    selectedRolloutObjects.value
  );

  isGeneratingDDL.value = true;
  const { statement, errors, fatal } = await generateDiffDDL(
    db,
    source,
    target,
    /* !allowEmptyDiffDDLWithConfigChange */ false
  );
  if (errors.length > 0) {
    return cleanup(errors, fatal);
  }
  const sheet = Sheet.fromPartial({
    database: db.name,
    visibility: Sheet_Visibility.VISIBILITY_PROJECT,
    type: Sheet_Type.TYPE_SQL,
    source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
    payload: {
      type: SheetPayload_Type.SCHEMA_DESIGN,
      baselineDatabaseConfig: {
        schemaConfigs: source.schemaConfigs,
      },
      databaseConfig: {
        schemaConfigs: target.schemaConfigs,
      },
    },
  });
  setSheetStatement(sheet, statement);
  const createdSheet = await useSheetV1Store().createSheet(
    props.project.name,
    sheet
  );
  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    project: props.project.uid,
    databaseList: db.uid,
    sheetId: extractSheetUID(createdSheet.name),
    name: generateIssueName(db.databaseName),
  };
  const routeInfo = {
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  };
  router.push(routeInfo);
};

const generateIssueName = (databaseName: string) => {
  const issueNameParts: string[] = [];
  issueNameParts.push(`Apply branch`);
  issueNameParts.push(`"${props.branch.name}"`);
  issueNameParts.push("to database");
  issueNameParts.push(`[${databaseName}]`);
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);
  return issueNameParts.join(" ");
};

watch(
  () => props.branch,
  () => {
    // Auto select the branch's baseline database as the target database by default
    const db = useDatabaseV1Store().getDatabaseByName(
      props.branch.baselineDatabase
    );
    handleSelectDatabase(db.uid);
  },
  { immediate: true }
);
</script>
