<template>
  <div class="flex items-center">
    <NTabs
      :value="selectedSpec.id"
      type="line"
      size="small"
      class="flex-1"
      tab-class="first:ml-4"
      @update:value="handleTabChange"
    >
      <NTab v-for="(spec, index) in plan.specs" :key="spec.id" :name="spec.id">
        <span v-if="plan.specs.length > 1" class="mr-1 opacity-80"
          >#{{ index + 1 }}</span
        >
        {{ getSpecTitle(spec) }}
        <span
          v-if="isSpecEmpty(spec)"
          class="text-error ml-0.5"
          :title="$t('plan.navigator.statement-empty')"
        >
          *
        </span>
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
import { PlusIcon } from "lucide-vue-next";
import { NTabs, NTab, NButton } from "naive-ui";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { usePlanContext } from "../../logic/context";
import { getSpecTitle } from "../../logic/utils";
import AddSpecDrawer from "../AddSpecDrawer.vue";
import { useSpecsValidation } from "../common";
import { useSelectedSpec } from "./context";

const router = useRouter();
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
</script>
