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
            v-if="isCreating && plan.specs.length > 1"
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
            v-if="isCreating"
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
import { PlusIcon, XIcon } from "lucide-vue-next";
import { NTabs, NTab, NButton, useDialog } from "naive-ui";
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
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

const handleTabChange = (specId: string) => {
  if (selectedSpec.value.id !== specId) {
    gotoSpec(specId);
  }
};

const handleSpecCreated = async (spec: Plan_Spec) => {
  // Add the new spec to the plan.
  plan.value.specs.push(spec);
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
    onPositiveClick: () => {
      const specIndex = plan.value.specs.findIndex((s) => s.id === spec.id);
      if (specIndex >= 0) {
        plan.value.specs.splice(specIndex, 1);

        // If we deleted the currently selected spec, navigate to the first remaining spec
        if (selectedSpec.value.id === spec.id && plan.value.specs.length > 0) {
          gotoSpec(plan.value.specs[0].id);
        }
      }
    },
  });
};
</script>
