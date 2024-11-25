<template>
  <div class="flex flex-col gap-y-2">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-x-4">
        <div class="flex items-center gap-x-1 text-sm font-medium">
          <span
            :class="isEmpty(state.statement) ? 'text-red-600' : 'text-control'"
          >
            {{ statementTitle }}
          </span>
          <span v-if="isCreating" class="text-red-600">*</span>
          <NButton
            v-if="!isCreating && !hasFeature('bb.feature.sql-review')"
            size="tiny"
            @click.prevent="state.showFeatureModal = true"
          >
            🎈{{ $t("sql-review.unlock-full-feature") }}
          </NButton>
        </div>
      </div>

      <div class="flex items-center justify-end gap-x-2">
        <template v-if="isCreating">
          <FormatOnSaveCheckbox
            v-model:value="formatOnSave"
            :language="language"
          />
          <SQLUploadButton
            size="tiny"
            :loading="state.isUploadingFile"
            @update:sql="handleUpdateStatement"
          >
            {{ $t("issue.upload-sql") }}
          </SQLUploadButton>
        </template>

        <template v-else>
          <template v-if="!state.isEditing">
            <template v-if="shouldShowEditButton">
              <!-- for small size sheets, show full featured UI editing button group -->
              <NTooltip :disabled="denyEditStatementReasons.length === 0">
                <template #trigger>
                  <NButton
                    v-if="!isSheetOversize"
                    size="tiny"
                    tag="div"
                    :disabled="denyEditStatementReasons.length > 0"
                    @click.prevent="beginEdit"
                  >
                    {{ $t("common.edit") }}
                  </NButton>
                  <!-- for oversized sheets, only allow to upload and overwrite the sheet -->
                  <SQLUploadButton
                    v-else
                    size="tiny"
                    :loading="state.isUploadingFile"
                    @update:sql="handleUpdateStatementAndOverwrite"
                  >
                    {{ $t("issue.upload-sql") }}
                  </SQLUploadButton>
                </template>
                <template #default>
                  <ErrorList :errors="denyEditStatementReasons" />
                </template>
              </NTooltip>
            </template>
          </template>
          <template v-else>
            <FormatOnSaveCheckbox
              v-model:value="formatOnSave"
              :language="language"
            />
            <SQLUploadButton
              size="tiny"
              :loading="state.isUploadingFile"
              @update:sql="handleUpdateStatement"
            >
              {{ $t("issue.upload-sql") }}
            </SQLUploadButton>
            <NButton
              v-if="state.isEditing"
              size="tiny"
              :disabled="!allowSaveSQL"
              @click.prevent="saveEdit"
            >
              {{ $t("common.save") }}
            </NButton>
            <NButton
              v-if="state.isEditing"
              size="tiny"
              quaternary
              @click.prevent="cancelEdit"
            >
              {{ $t("common.cancel") }}
            </NButton>
          </template>
        </template>
      </div>
    </div>

    <BBAttention
      v-if="isSheetOversize"
      type="warning"
      :title="$t('issue.statement-from-sheet-warning')"
    >
      <template #action>
        <DownloadSheetButton v-if="sheetName" :sheet="sheetName" size="small" />
      </template>
    </BBAttention>

    <div
      ref="editorContainerElRef"
      class="whitespace-pre-wrap overflow-hidden min-h-[120px] relative"
      :data-height="editorContainerHeight"
    >
      <MonacoEditor
        ref="monacoEditorRef"
        class="w-full h-auto max-h-[240px] min-h-[120px] border rounded-[3px]"
        :filename="filename"
        :content="state.statement"
        :language="language"
        :auto-focus="false"
        :readonly="isEditorReadonly"
        :dialect="dialect"
        :advices="isEditorReadonly || isCreating ? markers : []"
        :auto-height="{ min: 120, max: 240 }"
        :auto-complete-context="{
          instance: database.instance,
          database: database.name,
          scene: 'all',
        }"
        @update:content="handleStatementChange"
      />
      <div class="absolute bottom-[3px] right-[18px]">
        <NButton
          size="small"
          :quaternary="true"
          style="--n-padding: 0 5px"
          @click="state.showEditorModal = true"
        >
          <template #icon>
            <ExpandIcon class="w-4 h-4" />
          </template>
        </NButton>
      </div>
    </div>
  </div>

  <BBModal
    v-model:show="state.showEditorModal"
    :title="statementTitle"
    :trap-focus="true"
    header-class="!border-b-0"
    container-class="!pt-0 !overflow-hidden"
  >
    <div
      id="modal-editor-container"
      style="
        width: calc(100vw - 10rem);
        height: calc(100vh - 10rem);
        overflow: hidden;
        position: relative;
      "
      class="border rounded-[3px]"
    >
      <MonacoEditor
        v-if="state.showEditorModal"
        class="w-full h-full"
        :filename="filename"
        :content="state.statement"
        :language="language"
        :auto-focus="false"
        :readonly="isEditorReadonly"
        :dialect="dialect"
        :advices="isEditorReadonly || isCreating ? markers : []"
        :auto-complete-context="{
          instance: database.instance,
          database: database.name,
          scene: 'all',
        }"
        @update:content="handleStatementChange"
      />
    </div>
  </BBModal>

  <FeatureModal
    :open="state.showFeatureModal"
    feature="bb.feature.sql-review"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script setup lang="ts">
import { useElementSize } from "@vueuse/core";
import { cloneDeep, head } from "lodash-es";
import { ExpandIcon } from "lucide-vue-next";
import { NButton, NTooltip, useDialog } from "naive-ui";
import { v1 as uuidv1 } from "uuid";
import { computed, h, reactive, ref, toRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBAttention, BBModal } from "@/bbkit";
import { FeatureModal } from "@/components/FeatureGuard";
import { MonacoEditor } from "@/components/MonacoEditor";
import { extensionNameOfLanguage } from "@/components/MonacoEditor/utils";
import { ErrorList } from "@/components/Plan/components/common";
import {
  createEmptyLocalSheet,
  databaseEngineForSpec,
  databaseForSpec,
  isGroupingChangeSpec,
} from "@/components/Plan/logic";
import { usePlanContext } from "@/components/Plan/logic";
import DownloadSheetButton from "@/components/Sheet/DownloadSheetButton.vue";
import SQLUploadButton from "@/components/misc/SQLUploadButton.vue";
import { planServiceClient } from "@/grpcweb";
import { hasFeature, pushNotification, useSheetV1Store } from "@/store";
import type { SQLDialect } from "@/types";
import { EMPTY_ID, dialectOfEngineV1 } from "@/types";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import { Sheet } from "@/types/proto/v1/sheet_service";
import type { Advice } from "@/types/proto/v1/sql_service";
import {
  defer,
  getSheetStatement,
  setSheetStatement,
  useInstanceV1EditorLanguage,
  getStatementSize,
  flattenSpecList,
} from "@/utils";
import { useSQLAdviceMarkers } from "../useSQLAdviceMarkers";
import FormatOnSaveCheckbox from "./FormatOnSaveCheckbox.vue";
import type { EditState } from "./useTempEditState";
import { useTempEditState } from "./useTempEditState";

type LocalState = EditState & {
  showFeatureModal: boolean;
  showEditorModal: boolean;
  isUploadingFile: boolean;
};

const props = defineProps<{
  advices?: Advice[];
}>();

const { t } = useI18n();
const context = usePlanContext();
const { isCreating, plan, selectedSpec, formatOnSave, events } =
  usePlanContext();
const project = computed(() => plan.value.projectEntity);
const dialog = useDialog();
const editorContainerElRef = ref<HTMLElement>();
const monacoEditorRef = ref<InstanceType<typeof MonacoEditor>>();
const { height: editorContainerHeight } = useElementSize(editorContainerElRef);

const state = reactive<LocalState>({
  isEditing: false,
  statement: "",
  showFeatureModal: false,
  showEditorModal: false,
  isUploadingFile: false,
});

const database = computed(() => {
  return databaseForSpec(plan.value, selectedSpec.value);
});

const language = useInstanceV1EditorLanguage(
  computed(() => database.value.instanceResource)
);
const filename = computed(() => {
  const name = uuidv1();
  const ext = extensionNameOfLanguage(language.value);
  return `${name}.${ext}`;
});
const dialect = computed((): SQLDialect => {
  const db = database.value;
  return dialectOfEngineV1(db.instanceResource.engine);
});
const statementTitle = computed(() => {
  return language.value === "sql" ? t("common.sql") : t("common.statement");
});
const { markers } = useSQLAdviceMarkers(context, toRef(props, "advices"));

/**
 * to set the MonacoEditor as readonly
 * This happens when
 * - Not in edit mode
 * - Disallowed to edit statement
 */
const isEditorReadonly = computed(() => {
  if (isCreating.value) {
    return false;
  }
  return !state.isEditing || isSheetOversize.value || false;
});

const {
  sheet,
  sheetName,
  sheetReady,
  sheetStatement,
  reset: resetTempEditState,
} = useTempEditState(state);

const isSheetOversize = computed(() => {
  if (isCreating.value) return false;
  if (state.isEditing) return false;
  if (!sheetReady.value) return false;
  if (!sheet.value) return false;
  return getStatementSize(getSheetStatement(sheet.value)).lt(
    sheet.value.contentSize
  );
});

const denyEditStatementReasons = computed(() => {
  return [];
});

const shouldShowEditButton = computed(() => {
  // Need not to show "Edit" while the plan is still pending create.
  if (isCreating.value) {
    return false;
  }
  // Will show another button group as [Upload][Cancel][Save]
  // while editing
  if (state.isEditing) {
    return false;
  }
  if (plan.value.issue) {
    return false;
  }
  return true;
});

const allowSaveSQL = computed((): boolean => {
  if (state.statement === "") {
    // Not allowed if the statement is empty.
    return false;
  }
  if (!sheetReady.value) {
    return false;
  }
  if (state.statement === sheetStatement.value) {
    // Not allowed if the statement is not modified.
    return false;
  }

  // Allowed to save otherwise
  return true;
});

const beginEdit = () => {
  state.isEditing = true;
};

const saveEdit = async () => {
  try {
    await updateStatement(state.statement);
    resetTempEditState();
  } finally {
    state.isEditing = false;
  }
};

const cancelEdit = () => {
  state.statement = sheetStatement.value;
  state.isEditing = false;
};

const chooseUpdateStatementTarget = () => {
  type Target = "CANCELED" | "SPEC" | "ALL";
  const d = defer<{ target: Target; specs: Plan_Spec[] }>();

  const targets: Record<Target, Plan_Spec[]> = {
    CANCELED: [],
    SPEC: [selectedSpec.value],
    ALL: flattenSpecList(plan.value),
  };

  // If there is only one spec, we don't need to ask the user to choose.
  if (targets.ALL.length === 1) {
    d.resolve({ target: "SPEC", specs: targets.SPEC });
    return d.promise;
  }

  const $d = dialog.create({
    title: t("issue.update-statement.self", { type: statementTitle.value }),
    content: t("issue.update-statement.apply-current-change-to"),
    type: "info",
    autoFocus: false,
    closable: false,
    maskClosable: false,
    closeOnEsc: false,
    showIcon: false,
    action: () => {
      const finish = (target: Target) => {
        d.resolve({ target, specs: targets[target] });
        $d.destroy();
      };

      const CANCEL = h(
        NButton,
        { size: "small", onClick: () => finish("CANCELED") },
        {
          default: () => t("common.cancel"),
        }
      );
      const SPEC = h(
        NButton,
        { size: "small", onClick: () => finish("SPEC") },
        {
          default: () => t("issue.update-statement.target.selected-task"),
        }
      );
      const ALL = h(
        NButton,
        { size: "small", onClick: () => finish("ALL") },
        {
          default: () => t("issue.update-statement.target.all-tasks"),
        }
      );

      const buttons = [CANCEL];
      // For no grouping change spec, we can choose to update the current spec.
      if (!isGroupingChangeSpec(selectedSpec.value)) {
        buttons.push(SPEC);
      }
      buttons.push(ALL);

      return h(
        "div",
        { class: "flex items-center justify-end gap-x-2" },
        buttons
      );
    },
    onClose() {
      d.resolve({ target: "CANCELED", specs: [] });
    },
  });

  return d.promise;
};

const showOverwriteConfirmDialog = () => {
  return new Promise((resolve, reject) => {
    // Show a confirm dialog before replacing if the editing statement is not empty.
    dialog.create({
      positiveText: t("common.confirm"),
      negativeText: t("common.cancel"),
      title: t("issue.overwrite-current-statement"),
      autoFocus: false,
      closable: false,
      maskClosable: false,
      closeOnEsc: false,
      onNegativeClick: () => {
        reject();
      },
      onPositiveClick: () => {
        resolve(undefined);
      },
    });
  });
};

const handleUpdateStatementAndOverwrite = async (
  statement: string,
  filename: string
) => {
  try {
    await showOverwriteConfirmDialog();
  } catch {
    return;
  }

  state.isEditing = true;
  state.statement = statement;
  await handleUpdateStatement(statement, filename);
};

const handleUpdateStatement = async (statement: string, filename: string) => {
  try {
    state.isUploadingFile = true;
    handleStatementChange(statement);
    if (sheet.value) {
      sheet.value.title = filename;
    }
    resetTempEditState();
  } finally {
    state.isUploadingFile = false;
  }
};

const updateStatement = async (statement: string) => {
  const planPatch = cloneDeep(plan.value);
  if (!planPatch) {
    return;
  }

  const specsIdList: string[] = [];
  const { target, specs } = await chooseUpdateStatementTarget();
  if (target === "CANCELED" || specs.length === 0) {
    cancelEdit();
    return;
  }

  specs.forEach((spec) => {
    if (spec) {
      specsIdList.push(spec.id);
    }
  });

  const distinctSpecsIds = new Set(
    specsIdList.filter((id) => id && id !== String(EMPTY_ID))
  );
  if (distinctSpecsIds.size === 0) {
    return;
  }

  const specsToPatch = planPatch.steps
    .flatMap((step) => step.specs)
    .filter((spec) => distinctSpecsIds.has(spec.id));
  const sheet = Sheet.fromPartial({
    ...createEmptyLocalSheet(),
    title: plan.value.title,
    engine: await databaseEngineForSpec(project.value, head(specsToPatch)),
  });
  setSheetStatement(sheet, statement);
  const createdSheet = await useSheetV1Store().createSheet(
    plan.value.project,
    sheet
  );

  for (const spec of specsToPatch) {
    if (spec.changeDatabaseConfig) {
      spec.changeDatabaseConfig.sheet = createdSheet.name;
    } else {
      console.error("Unexpected spec type", spec);
    }
  }

  const updatedPlan = await planServiceClient.updatePlan({
    plan: planPatch,
    updateMask: ["steps"],
  });

  Object.assign(plan.value, updatedPlan);
  events.emit("status-changed", { eager: true });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const handleStatementChange = (statement: string) => {
  if (isEditorReadonly.value) {
    return;
  }

  state.statement = statement;
  if (isCreating.value) {
    // When creating an plan, update the local sheet directly.
    if (!sheet.value) return;
    setSheetStatement(sheet.value, statement);
  }
};

watch(
  sheetStatement,
  (statement) => {
    state.statement = statement;
  },
  { immediate: true }
);

watch(isCreating, (curr, prev) => {
  // Reset the edit state after creating the plan.
  if (!curr && prev) {
    state.isEditing = false;
  }
});

defineExpose({
  get editor() {
    return monacoEditorRef.value;
  },
});
</script>
