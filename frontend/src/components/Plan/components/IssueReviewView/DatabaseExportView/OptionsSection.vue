<template>
  <div class="flex flex-col gap-y-2">
    <div class="flex items-center justify-between">
      <h3 class="text-base">
        {{ $t("issue.data-export.options") }}
      </h3>
      <div
        v-if="shouldShowEditButton || isEditing"
        class="flex flex-row justify-end items-center gap-2"
      >
        <template v-if="!isEditing">
          <NButton size="small" @click="handleEdit">
            {{ $t("common.edit") }}
          </NButton>
        </template>
        <template v-else>
          <NButton size="small" :disabled="!hasChanges" @click="handleSave">
            {{ $t("common.save") }}
          </NButton>
          <NButton size="small" quaternary @click="handleCancel">
            {{ $t("common.cancel") }}
          </NButton>
        </template>
      </div>
    </div>

    <div class="p-3 border rounded-sm flex flex-col gap-y-3">
      <!-- Format Display/Edit -->
      <div class="flex items-center gap-4">
        <span class="text-sm">
          {{ $t("issue.data-export.format") }}
        </span>
        <ExportFormatSelector
          v-model:format="editableConfig.format"
          :editable="optionsEditable"
        />
      </div>

      <!-- Password Protection Display/Edit -->
      <ExportPasswordInputer
        v-model:password="editableConfig.password"
        :editable="optionsEditable"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { NButton } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { planServiceClientConnect } from "@/connect";
import { pushNotification } from "@/store";
import type { Plan_ExportDataConfig } from "@/types/proto-es/v1/plan_service_pb";
import { UpdatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { usePlanContext } from "../../../logic";
import ExportFormatSelector from "../../ExportOption/ExportFormatSelector.vue";
import ExportPasswordInputer from "../../ExportOption/ExportPasswordInputer.vue";

const { t } = useI18n();
const { plan, readonly, isCreating } = usePlanContext();

// Get export config from plan context
const exportDataConfig = computed(() => {
  const exportDataSpec = plan.value.specs.find(
    (spec) => spec.config?.case === "exportDataConfig"
  );
  if (exportDataSpec?.config.case === "exportDataConfig") {
    return exportDataSpec.config.value;
  }
  return null;
});

// Local editing state
const state = reactive({
  isEditing: false,
  editableConfig: {} as Plan_ExportDataConfig,
});

// Initialize editableConfig when exportDataConfig becomes available
watchEffect(() => {
  if (
    exportDataConfig.value &&
    Object.keys(state.editableConfig).length === 0
  ) {
    state.editableConfig = { ...exportDataConfig.value };
  }
});

const isEditing = computed(() => state.isEditing);

// Check if we should show edit button (similar to statement section logic)
const shouldShowEditButton = computed(() => {
  // Not allowed to edit if readonly
  if (readonly.value) {
    return false;
  }
  // Need not to show "Edit" while the plan is still pending create
  if (isCreating.value) {
    return false;
  }
  // Will show another button group as [Save][Cancel] while editing
  if (state.isEditing) {
    return false;
  }
  if (plan.value.hasRollout) {
    return false;
  }
  return true;
});

const optionsEditable = computed(() => {
  return isCreating.value || state.isEditing;
});

const editableConfig = computed(() => state.editableConfig);

// Check if the editable config has changes compared to the original
const hasChanges = computed(() => {
  if (!state.isEditing || !exportDataConfig.value) {
    return false;
  }

  // Compare the current editable config with the original config
  return (
    state.editableConfig.format !== exportDataConfig.value.format ||
    state.editableConfig.password !== exportDataConfig.value.password
  );
});

// Event handlers
const handleEdit = () => {
  if (!exportDataConfig.value) return;

  state.isEditing = true;
  state.editableConfig = { ...exportDataConfig.value };
};

const handleSave = async () => {
  try {
    const exportDataSpec = plan.value.specs.find(
      (spec) => spec.config?.case === "exportDataConfig"
    );

    if (!exportDataSpec) {
      throw new Error("Cannot find export data spec to update");
    }

    const planPatch = cloneDeep(plan.value);
    const specToPatch = planPatch.specs.find(
      (spec) => spec.id === exportDataSpec.id
    );

    if (!specToPatch || specToPatch.config.case !== "exportDataConfig") {
      throw new Error("Cannot find export data spec to update");
    }

    // Update the export config
    specToPatch.config.value = { ...state.editableConfig };

    const request = create(UpdatePlanRequestSchema, {
      plan: planPatch,
      updateMask: { paths: ["specs"] },
    });

    const response = await planServiceClientConnect.updatePlan(request);
    Object.assign(plan.value, response);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: `Failed to update export options: ${error}`,
    });
    return; // Don't exit editing mode if save failed
  }

  state.isEditing = false;
};

const handleCancel = () => {
  state.isEditing = false;
  if (exportDataConfig.value) {
    state.editableConfig = { ...exportDataConfig.value };
  }
};
</script>
