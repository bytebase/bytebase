<template>
  <div class="flex items-center">
    <NTabs
      :key="`${plan.specs.length}-${selectedSpec.id}`"
      :value="selectedSpec.id"
      type="card"
      size="small"
      class="flex-1"
      @update:value="handleTabChange"
    >
      <template #prefix>
        <div class=""></div>
        <div v-if="!plan.issue" class="pl-4 text-base font-medium">
          {{ $t("plan.navigator.changes") }}
        </div>
      </template>

      <NTab v-for="(spec, index) in plan.specs" :key="spec.id" :name="spec.id">
        <div class="flex items-center gap-1">
          <span v-if="plan.specs.length > 1" class="opacity-80"
            >#{{ index + 1 }}</span
          >
          {{ getSpecTitle(spec) }}
          <NTooltip v-if="isSpecEmpty(spec)">
            <template #trigger>
              <span class="text-error">*</span>
            </template>
            {{ $t("plan.navigator.statement-empty") }}
          </NTooltip>
          <NDropdown
            v-if="
              canModifySpecs &&
              (canEditMigrationType(spec) || plan.specs.length > 1)
            "
            :options="getDropdownOptions(spec)"
            trigger="click"
            placement="bottom-end"
            :size="'small'"
            @select="(key) => handleDropdownSelect(key, spec)"
          >
            <NButton @click.stop text size="tiny">
              <template #icon>
                <MoreVerticalIcon :size="14" class="text-control-light" />
              </template>
            </NButton>
          </NDropdown>
        </div>
      </NTab>

      <template #suffix>
        <div class="pr-4">
          <NButton
            v-if="canModifySpecs"
            type="default"
            size="small"
            @click="showAddSpecDrawer = true"
          >
            {{ $t("plan.add-spec") }}
          </NButton>
        </div>
      </template>
    </NTabs>
  </div>

  <AddSpecDrawer
    v-model:show="showAddSpecDrawer"
    @created="handleSpecCreated"
  />

  <EditMigrationTypeModal
    v-if="editingSpec"
    :show="showEditModal"
    :migration-type="
      editingSpec.config.case === 'changeDatabaseConfig'
        ? editingSpec.config.value.migrationType
        : MigrationType.DDL
    "
    @update:show="showEditModal = $event"
    @save="handleSaveMigrationType"
  />
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { MoreVerticalIcon, PencilIcon, TrashIcon } from "lucide-vue-next";
import type { DropdownOption } from "naive-ui";
import { NButton, NDropdown, NTab, NTabs, NTooltip, useDialog } from "naive-ui";
import { computed, h, nextTick, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { planServiceClientConnect } from "@/grpcweb";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useCurrentProjectV1,
  useSheetV1Store,
} from "@/store";
import { MigrationType } from "@/types/proto-es/v1/common_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { UpdatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { extractSheetUID } from "@/utils";
import { databaseEngineForSpec, getLocalSheetByName } from "../../logic";
import { usePlanContext } from "../../logic/context";
import { useEditorState } from "../../logic/useEditorState";
import { getSpecTitle } from "../../logic/utils";
import AddSpecDrawer from "../AddSpecDrawer.vue";
import { useSpecsValidation } from "../common";
import { useSelectedSpec } from "./context";
import EditMigrationTypeModal from "./EditMigrationTypeModal.vue";

const router = useRouter();
const dialog = useDialog();
const { t } = useI18n();
const { plan, isCreating } = usePlanContext();
const sheetStore = useSheetV1Store();
const selectedSpec = useSelectedSpec();
const { project } = useCurrentProjectV1();
const { isSpecEmpty } = useSpecsValidation(computed(() => plan.value.specs));
const { setEditingState } = useEditorState();

const showAddSpecDrawer = ref(false);
const editingSpec = ref<Plan_Spec | null>(null);
const showEditModal = ref(false);

// Allow adding/removing specs when:
// 1. Plan is being created (isCreating = true), OR
// 2. Plan is created but rollout is empty (plan.rollout === "")
const canModifySpecs = computed(() => {
  return isCreating.value || plan.value.rollout === "";
});

const handleTabChange = (specId: string) => {
  if (selectedSpec.value.id !== specId) {
    gotoSpec(specId);
  }
};

const handleSpecCreated = async (spec: Plan_Spec) => {
  // Add the new spec to the plan.

  // If the plan is not being created (already exists), call UpdatePlan API
  if (!isCreating.value) {
    // If the spec references a sheet that is pending creation (UID starts with "-"),
    // we need to create the sheet first.
    if (
      spec.config.case === "changeDatabaseConfig" ||
      spec.config.case === "exportDataConfig"
    ) {
      const uid = extractSheetUID(spec.config.value.sheet);
      if (uid.startsWith("-")) {
        // The sheet is pending create.
        const sheetToCreate = getLocalSheetByName(spec.config.value.sheet);
        const engine = await databaseEngineForSpec(spec);
        sheetToCreate.engine = engine;
        sheetToCreate.title = plan.value.title;
        const createdSheet = await sheetStore.createSheet(
          project.value.name,
          sheetToCreate
        );
        spec.config.value.sheet = createdSheet.name;
      }
    }
    plan.value.specs.push(spec);
    try {
      const request = create(UpdatePlanRequestSchema, {
        plan: plan.value,
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
      // If the API call fails, remove the spec from local state
      const specIndex = plan.value.specs.findIndex((s) => s.id === spec.id);
      if (specIndex >= 0) {
        plan.value.specs.splice(specIndex, 1);
      }
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: `Failed to add spec: ${error}`,
      });
      return;
    }
  } else {
    plan.value.specs.push(spec);
  }

  const enableEditing = !isCreating.value;
  gotoSpec(spec.id, enableEditing);
};

const gotoSpec = (specId: string, enableEditing = false) => {
  const currentRoute = router.currentRoute.value;
  router
    .push({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
      params: {
        ...(currentRoute.params || {}),
        specId,
      },
      query: currentRoute.query || {},
    })
    .then(() => {
      // Enable editing mode if requested (for newly created specs)
      if (enableEditing) {
        // Use nextTick to ensure the route navigation completes first
        nextTick(() => {
          setEditingState(true);
        });
      }
    });
};

const canEditMigrationType = (spec: Plan_Spec) => {
  return spec.config.case === "changeDatabaseConfig";
};

const getDropdownOptions = (spec: Plan_Spec): DropdownOption[] => {
  const options: DropdownOption[] = [];
  if (canEditMigrationType(spec)) {
    options.push({
      key: "edit",
      label: t("common.edit"),
      icon: () => h(PencilIcon, { size: 16 }),
    });
  }
  if (plan.value.specs.length > 1) {
    options.push({
      key: "delete",
      label: t("common.delete"),
      icon: () => h(TrashIcon, { size: 16 }),
    });
  }
  return options;
};

const handleDropdownSelect = (key: string, spec: Plan_Spec) => {
  if (key === "edit") {
    handleOpenEditModal(spec);
  } else if (key === "delete") {
    handleDeleteSpec(spec);
  }
};

const handleOpenEditModal = (spec: Plan_Spec) => {
  if (spec.config.case === "changeDatabaseConfig") {
    editingSpec.value = spec;
    showEditModal.value = true;
  }
};

const handleSaveMigrationType = async (newType: MigrationType) => {
  const spec = editingSpec.value;
  if (!spec || spec.config.case !== "changeDatabaseConfig") {
    showEditModal.value = false;
    editingSpec.value = null;
    return;
  }

  const oldType = spec.config.value.migrationType;

  if (oldType === newType) {
    showEditModal.value = false;
    editingSpec.value = null;
    return;
  }

  // Find the spec in the plan's specs array
  const specIndex = plan.value.specs.findIndex((s) => s.id === spec.id);
  if (specIndex < 0) {
    showEditModal.value = false;
    editingSpec.value = null;
    return;
  }

  // Update the spec in the plan
  const targetSpec = plan.value.specs[specIndex];
  if (targetSpec.config.case !== "changeDatabaseConfig") {
    showEditModal.value = false;
    editingSpec.value = null;
    return;
  }

  // Update local state
  targetSpec.config.value.migrationType = newType;

  // If the plan is not being created (already exists), call UpdatePlan API
  if (!isCreating.value) {
    try {
      const request = create(UpdatePlanRequestSchema, {
        plan: plan.value,
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
      // If the API call fails, restore the old migration type
      targetSpec.config.value.migrationType = oldType;
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: `Failed to update migration type: ${error}`,
      });
      showEditModal.value = false;
      editingSpec.value = null;
      return;
    }
  }

  showEditModal.value = false;
  editingSpec.value = null;
};

const handleDeleteSpec = (spec: Plan_Spec) => {
  dialog.warning({
    title: t("plan.spec.delete-change.title"),
    content: t("plan.spec.delete-change.content"),
    positiveText: t("common.delete"),
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      if (plan.value.specs.length <= 1) {
        pushNotification({
          module: "bytebase",
          style: "WARN",
          title: "Cannot delete last spec",
        });
        return;
      }

      const specIndex = plan.value.specs.findIndex((s) => s.id === spec.id);
      if (specIndex < 0) return;

      // Remove the spec from local state
      const removedSpec = plan.value.specs.splice(specIndex, 1)[0];

      // If the plan is not being created (already exists), call UpdatePlan API
      if (!isCreating.value) {
        try {
          const request = create(UpdatePlanRequestSchema, {
            plan: plan.value,
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
          // If the API call fails, restore the spec to local state
          plan.value.specs.splice(specIndex, 0, removedSpec);
          pushNotification({
            module: "bytebase",
            style: "CRITICAL",
            title: t("common.error"),
            description: `Failed to delete spec: ${error}`,
          });
          return;
        }
      }

      // If we deleted the currently selected spec, navigate to the first remaining spec
      if (selectedSpec.value.id === spec.id && plan.value.specs.length > 0) {
        gotoSpec(plan.value.specs[0].id);
      }
    },
  });
};
</script>
