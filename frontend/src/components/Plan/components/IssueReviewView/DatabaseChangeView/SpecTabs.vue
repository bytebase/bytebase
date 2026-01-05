<template>
  <div>
    <!-- Tab Bar -->
    <div class="flex items-center justify-between border-b border-control-border">
      <div class="flex items-center">
        <button
          v-for="(spec, index) in specs"
          :key="spec.id"
          class="flex items-center gap-1 px-3 py-1.5 -mb-px text-sm border rounded-t-md cursor-pointer transition-colors"
          :class="
            selectedSpecId === spec.id
              ? 'font-medium text-main bg-white border-control-border border-b-white'
              : 'text-control-light bg-transparent border-transparent hover:text-control'
          "
          @click="$emit('update:selectedSpecId', spec.id)"
        >
          <span class="opacity-60">#{{ index + 1 }}</span>
          <span>{{ getSpecTitle(spec) }}</span>
          <NTooltip v-if="isSpecEmpty(spec)">
            <template #trigger>
              <span class="text-error">*</span>
            </template>
            {{ $t("plan.navigator.statement-empty") }}
          </NTooltip>
          <NDropdown
            v-if="canModifySpecs && specs.length > 1"
            :options="getDropdownOptions()"
            trigger="click"
            placement="bottom-end"
            size="small"
            @select="(key) => handleDropdownSelect(key, spec)"
          >
            <NButton @click.stop text size="tiny">
              <template #icon>
                <MoreVerticalIcon :size="14" class="text-control-light" />
              </template>
            </NButton>
          </NDropdown>
        </button>
      </div>

      <NButton
        v-if="canModifySpecs"
        type="default"
        size="small"
        @click="showAddSpecDrawer = true"
      >
        {{ $t("plan.add-spec") }}
      </NButton>
    </div>

    <!-- Content Area -->
    <div class="bg-white border-x border-b border-control-border rounded-b-lg py-2 px-3">
      <slot />
    </div>
  </div>

  <AddSpecDrawer v-model:show="showAddSpecDrawer" @created="handleSpecCreated" />
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { MoreVerticalIcon, TrashIcon } from "lucide-vue-next";
import type { DropdownOption } from "naive-ui";
import { NButton, NDropdown, NTooltip, useDialog } from "naive-ui";
import { computed, h, ref } from "vue";
import { useI18n } from "vue-i18n";
import { planServiceClientConnect } from "@/connect";
import {
  pushNotification,
  useCurrentProjectV1,
  useSheetV1Store,
} from "@/store";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { UpdatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { extractSheetUID } from "@/utils";
import { getLocalSheetByName, usePlanContext } from "../../../logic";
import { getSpecTitle } from "../../../logic/utils";
import AddSpecDrawer from "../../AddSpecDrawer.vue";
import { useSpecsValidation } from "../../common";

const props = defineProps<{ selectedSpecId: string }>();
const emit = defineEmits<{ "update:selectedSpecId": [specId: string] }>();

const dialog = useDialog();
const { t } = useI18n();
const { plan, issue, isCreating } = usePlanContext();
const sheetStore = useSheetV1Store();
const { project } = useCurrentProjectV1();

const specs = computed(() => plan.value.specs);
const { isSpecEmpty } = useSpecsValidation(specs);
const showAddSpecDrawer = ref(false);
const canModifySpecs = computed(
  () =>
    (isCreating.value || !plan.value.hasRollout) &&
    issue.value?.status !== IssueStatus.CANCELED &&
    issue.value?.status !== IssueStatus.DONE
);

const getDropdownOptions = (): DropdownOption[] =>
  specs.value.length > 1
    ? [
        {
          key: "delete",
          label: t("common.delete"),
          icon: () => h(TrashIcon, { size: 16 }),
        },
      ]
    : [];

const handleDropdownSelect = (key: string, spec: Plan_Spec) => {
  if (key === "delete") handleDeleteSpec(spec);
};

// Helper to update plan on server with optimistic update rollback on failure
const updatePlanSpecs = async (onError: () => void) => {
  const request = create(UpdatePlanRequestSchema, {
    plan: plan.value,
    updateMask: { paths: ["specs"] },
  });
  try {
    const response = await planServiceClientConnect.updatePlan(request);
    Object.assign(plan.value, response);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    return true;
  } catch (error) {
    onError();
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: String(error),
    });
    return false;
  }
};

const handleSpecCreated = async (spec: Plan_Spec) => {
  // For existing plans, create sheet on server first if needed
  if (!isCreating.value) {
    const config = spec.config;
    if (
      config.case === "changeDatabaseConfig" ||
      config.case === "exportDataConfig"
    ) {
      const uid = extractSheetUID(config.value.sheet);
      if (uid.startsWith("-")) {
        const createdSheet = await sheetStore.createSheet(
          project.value.name,
          getLocalSheetByName(config.value.sheet)
        );
        config.value.sheet = createdSheet.name;
      }
    }
    plan.value.specs.push(spec);
    const success = await updatePlanSpecs(() => {
      const idx = plan.value.specs.findIndex((s) => s.id === spec.id);
      if (idx >= 0) plan.value.specs.splice(idx, 1);
    });
    if (!success) return;
  } else {
    plan.value.specs.push(spec);
  }
  emit("update:selectedSpecId", spec.id);
};

const handleDeleteSpec = (spec: Plan_Spec) => {
  dialog.warning({
    title: t("plan.spec.delete-change.title"),
    content: t("plan.spec.delete-change.content"),
    positiveText: t("common.delete"),
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      if (specs.value.length <= 1) {
        pushNotification({
          module: "bytebase",
          style: "WARN",
          title: "Cannot delete last spec",
        });
        return;
      }

      const specIndex = plan.value.specs.findIndex((s) => s.id === spec.id);
      if (specIndex < 0) return;

      const removedSpec = plan.value.specs.splice(specIndex, 1)[0];

      if (!isCreating.value) {
        const success = await updatePlanSpecs(() => {
          plan.value.specs.splice(specIndex, 0, removedSpec);
        });
        if (!success) return;
      }

      if (props.selectedSpecId === spec.id && specs.value.length > 0) {
        emit("update:selectedSpecId", specs.value[0].id);
      }
    },
  });
};
</script>
