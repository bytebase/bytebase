<template>
  <div class="flex items-center">
    <NTabs
      :key="`${plan.specs.length}-${selectedSpec.id}`"
      :value="selectedSpec.id"
      type="line"
      size="small"
      class="flex-1"
      tab-class="first:ml-4"
      @update:value="handleTabChange"
    >
      <NTab v-for="(spec, index) in plan.specs" :key="spec.id" :name="spec.id">
        <div class="flex items-center gap-1">
          <span v-if="plan.specs.length > 1" class="opacity-80"
            >#{{ index + 1 }}</span
          >
          {{ getSpecTitle(spec) }}
          <span
            v-if="isSpecEmpty(spec)"
            class="text-error"
            :title="$t('plan.navigator.statement-empty')"
          >
            *
          </span>
          <NButton
            v-if="canModifySpecs && plan.specs.length > 1"
            quaternary
            size="tiny"
            :title="$t('common.delete')"
            @click.stop="handleDeleteSpec(spec)"
          >
            <template #icon>
              <XIcon :size="14" class="text-control-light" />
            </template>
          </NButton>
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
            <template #icon>
              <PlusIcon class="w-4 h-4" />
            </template>
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
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { PlusIcon, XIcon } from "lucide-vue-next";
import { NTabs, NTab, NButton, useDialog } from "naive-ui";
import { ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { planServiceClientConnect } from "@/grpcweb";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";
import { pushNotification } from "@/store";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { UpdatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { usePlanContext } from "../../logic/context";
import { getSpecTitle } from "../../logic/utils";
import AddSpecDrawer from "../AddSpecDrawer.vue";
import { useSpecsValidation } from "../common";
import { useSelectedSpec } from "./context";

const router = useRouter();
const dialog = useDialog();
const { t } = useI18n();
const { plan, isCreating } = usePlanContext();
const selectedSpec = useSelectedSpec();
const { isSpecEmpty } = useSpecsValidation(plan.value.specs);

const showAddSpecDrawer = ref(false);

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
  plan.value.specs.push(spec);

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
  }

  gotoSpec(spec.id);
};

const gotoSpec = (specId: string) => {
  const currentRoute = router.currentRoute.value;
  router.push({
    name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
    params: {
      ...(currentRoute.params || {}),
      specId,
    },
    query: currentRoute.query || {},
  });
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
