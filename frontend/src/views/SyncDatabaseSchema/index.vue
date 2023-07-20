<template>
  <div class="w-full h-full overflow-hidden flex flex-col">
    <p class="text-sm text-gray-500 px-4">
      {{ $t("database.sync-schema.description") }}
      <LearnMoreLink
        url="https://www.bytebase.com/docs/change-database/synchronize-schema?source=console"
      />
    </p>
    <BBStepTab
      ref="bbStepTabRef"
      class="p-4 flex-1 overflow-hidden flex flex-col"
      :step-item-list="stepTabList"
      :show-cancel="false"
      :allow-next="allowNext"
      :finish-title="$t('database.sync-schema.preview-issue')"
      pane-class="flex-1 overflow-y-auto"
      @cancel="cancelSetup"
      @try-change-step="tryChangeStep"
      @try-finish="tryFinishSetup"
    >
      <template #0>
        <div class="mb-4">
          <NRadioGroup v-model:value="state.sourceSchemaType" class="space-x-4">
            <NRadio :value="'DATABASE_SCHEMA'" :label="'Database Schema'" />
            <NRadio :value="'SCHEMA_DESIGN'" :label="'Schema Design'" />
          </NRadioGroup>
        </div>
        <DatabaseSchemaSelector
          v-if="state.sourceSchemaType === 'DATABASE_SCHEMA'"
          :select-state="changeHistorySourceSchemaState"
          @update="handleChangeHistorySchameVersionChanges"
        />
        <SchemaDesignSelector
          v-if="state.sourceSchemaType === 'SCHEMA_DESIGN'"
          :selected-schema-design="schemaDesignState.selectedSchemaDesign"
          @select="
            (schemaDesign) =>
              (schemaDesignState.selectedSchemaDesign = schemaDesign)
          "
        />
      </template>
      <template #1>
        <SelectTargetDatabasesView
          ref="targetDatabaseViewRef"
          :project-id="projectId!"
          :source-schema-type="state.sourceSchemaType"
          :database-source-schema="(changeHistorySourceSchemaState as any)"
          :schema-design="schemaDesignState.selectedSchemaDesign"
        />
      </template>
    </BBStepTab>
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { isNull, isUndefined } from "lodash-es";
import { NRadioGroup, NRadio, useDialog } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useProjectV1Store } from "@/store";
import { UNKNOWN_ID } from "@/types";
import SelectTargetDatabasesView from "./SelectTargetDatabasesView.vue";
import { ChangeHistorySourceSchema, SourceSchemaType } from "./types";
import DatabaseSchemaSelector from "./DatabaseSchemaSelector.vue";
import SchemaDesignSelector from "./SchemaDesignSelector.vue";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import { getProjectAndSchemaDesignSheetId } from "@/store/modules/v1/common";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import { BBStepTab } from "@/bbkit";

const SELECT_SOURCE_SCHEMA = 0;
const SELECT_TARGET_DATABASE_LIST = 1;

type Step = typeof SELECT_SOURCE_SCHEMA | typeof SELECT_TARGET_DATABASE_LIST;

interface LocalState {
  sourceSchemaType: SourceSchemaType;
  currentStep: Step;
}

const { t } = useI18n();
const router = useRouter();
const dialog = useDialog();
const bbStepTabRef = ref<InstanceType<typeof BBStepTab>>();
const projectStore = useProjectV1Store();
const schemaDesignStore = useSchemaDesignStore();
const targetDatabaseViewRef =
  ref<InstanceType<typeof SelectTargetDatabasesView>>();
const state = reactive<LocalState>({
  sourceSchemaType: "DATABASE_SCHEMA",
  currentStep: SELECT_SOURCE_SCHEMA,
});
const changeHistorySourceSchemaState = reactive<ChangeHistorySourceSchema>({});
const schemaDesignState = reactive<{
  selectedSchemaDesign?: SchemaDesign;
}>({});
const projectId = computed(() => {
  if (state.sourceSchemaType === "DATABASE_SCHEMA") {
    return changeHistorySourceSchemaState.projectId;
  } else {
    if (!schemaDesignState.selectedSchemaDesign) {
      return undefined;
    }
    const [projectName] = getProjectAndSchemaDesignSheetId(
      schemaDesignState.selectedSchemaDesign.name
    );
    const project = projectStore.getProjectByName(`projects/${projectName}`);
    return project.uid;
  }
});

const handleChangeHistorySchameVersionChanges = (
  schemaVersion: ChangeHistorySourceSchema
) => {
  Object.assign(changeHistorySourceSchemaState, schemaVersion);
};

const isValidId = (id: any): id is string => {
  if (isNull(id) || isUndefined(id) || String(id) === String(UNKNOWN_ID)) {
    return false;
  }
  return true;
};

const stepTabList = computed(() => {
  return [
    { title: t("database.sync-schema.select-source-schema") },
    { title: t("database.sync-schema.select-target-databases") },
  ];
});

const allowNext = computed(() => {
  if (state.currentStep === SELECT_SOURCE_SCHEMA) {
    if (state.sourceSchemaType === "DATABASE_SCHEMA") {
      return (
        isValidId(changeHistorySourceSchemaState.environmentId) &&
        isValidId(changeHistorySourceSchemaState.databaseId) &&
        !isUndefined(changeHistorySourceSchemaState.changeHistory)
      );
    } else {
      return !isUndefined(schemaDesignState.selectedSchemaDesign);
    }
  } else {
    if (!targetDatabaseViewRef.value) {
      return false;
    }
    const targetDatabaseList = targetDatabaseViewRef.value?.targetDatabaseList;
    const targetDatabaseDiffList = targetDatabaseList
      .map((db) => {
        const diff = targetDatabaseViewRef.value!.databaseDiffCache[db.uid];
        return {
          id: db.uid,
          diff: diff?.edited || "",
        };
      })
      .filter((item) => item.diff !== "");
    return targetDatabaseDiffList.length > 0;
  }
});

onMounted(async () => {
  const schemaDesignName = router.currentRoute.value.query
    .schemaDesignName as string;
  if (schemaDesignName) {
    try {
      const schemaDesign = await schemaDesignStore.getOrFetchSchemaDesignByName(
        schemaDesignName
      );
      if (schemaDesign) {
        state.sourceSchemaType = "SCHEMA_DESIGN";
        schemaDesignState.selectedSchemaDesign = schemaDesign;
        bbStepTabRef.value?.changeStep(SELECT_TARGET_DATABASE_LIST);
      }
    } catch (error) {
      // do nothing
    }
  }
});

const tryChangeStep = async (
  oldStep: number,
  newStep: number,
  allowChangeCallback: () => void
) => {
  if (oldStep === 1 && newStep === 0) {
    const targetDatabaseList =
      targetDatabaseViewRef.value?.targetDatabaseList || [];
    if (targetDatabaseList.length > 0) {
      dialog.create({
        positiveText: t("common.confirm"),
        negativeText: t("common.cancel"),
        title: t("deployment-config.confirm-to-revert"),
        autoFocus: false,
        closable: false,
        maskClosable: false,
        closeOnEsc: false,
        onNegativeClick: () => {
          // nothing to do
        },
        onPositiveClick: () => {
          state.currentStep = newStep as Step;
          allowChangeCallback();
        },
      });
      return;
    }
  }
  state.currentStep = newStep as Step;
  allowChangeCallback();
};

const tryFinishSetup = async () => {
  if (!targetDatabaseViewRef.value) {
    return;
  }

  const targetDatabaseList = targetDatabaseViewRef.value.targetDatabaseList;
  const targetDatabaseDiffList = targetDatabaseList
    .map((db) => {
      const diff = targetDatabaseViewRef.value!.databaseDiffCache[db.uid];
      return {
        id: db.uid,
        diff: diff.edited,
      };
    })
    .filter((item) => item.diff !== "");
  const databaseIdList = targetDatabaseDiffList.map((item) => item.id);
  const statementList = targetDatabaseDiffList.map((item) => item.diff);
  const project = await projectStore.getOrFetchProjectByUID(projectId.value!);

  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    project: project.uid,
    mode: "normal",
    ghost: undefined,
  };
  query.databaseList = databaseIdList.join(",");
  query.sqlList = JSON.stringify(statementList);
  query.name = generateIssueName(
    targetDatabaseList.map((db) => db.databaseName)
  );

  const routeInfo = {
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  };
  router.push(routeInfo);
};

const generateIssueName = (databaseNameList: string[]) => {
  const issueNameParts: string[] = [];
  if (databaseNameList.length === 1) {
    issueNameParts.push(`[${databaseNameList[0]}]`);
  } else {
    issueNameParts.push(`[${databaseNameList.length} databases]`);
  }
  issueNameParts.push(`Alter schema`);
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);
  return issueNameParts.join(" ");
};

const cancelSetup = () => {
  router.replace({
    name: "workspace.home",
  });
};
</script>
