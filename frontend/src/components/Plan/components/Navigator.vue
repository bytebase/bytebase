<template>
  <div class="w-60 flex flex-col border-r">
    <!-- Navigator Content -->
    <NScrollbar class="flex-1">
      <div class="py-2">
        <!-- Overview Tab -->
        <div
          class="mx-2 px-3 py-2 rounded-md cursor-pointer group transition-all"
          :class="[
            !selectedSpec
              ? 'bg-accent bg-opacity-10 shadow-sm'
              : 'hover:bg-gray-50',
          ]"
          @click="handleSelectOverview"
        >
          <div class="flex items-center gap-2">
            <LayoutDashboardIcon
              class="w-4 h-4 transition-colors"
              :class="[
                !selectedSpec
                  ? 'text-accent'
                  : 'text-control-light group-hover:text-main',
              ]"
            />
            <span
              class="text-sm font-medium transition-colors"
              :class="[
                !selectedSpec
                  ? 'text-accent'
                  : 'text-control group-hover:text-main',
              ]"
            >
              {{ $t("plan.navigator.overview") }}
            </span>
          </div>
        </div>

        <!-- Specs Section -->
        <div class="px-4 pb-2 mt-3">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <span
                class="text-xs font-medium uppercase tracking-wider text-control-light"
              >
                {{ $t("plan.navigator.specifications") }}
              </span>
              <NBadge
                v-if="specs.length > 10"
                type="info"
                :value="specs.length"
                :max="99"
              />
            </div>
            <NButton
              v-if="isCreating"
              type="default"
              size="tiny"
              @click="handleAddSpec"
            >
              <template #icon>
                <PlusIcon class="w-4 h-4" />
              </template>
            </NButton>
          </div>
        </div>

        <!-- Spec List -->
        <div class="px-2 space-y-1">
          <div
            v-for="(spec, index) in specs"
            :key="spec.id"
            class="px-3 py-2 rounded-md cursor-pointer group transition-all"
            :class="[
              selectedSpec?.id === spec.id
                ? 'bg-accent bg-opacity-10 shadow-sm'
                : 'hover:bg-gray-50',
            ]"
            @click="handleSelectSpec(spec)"
          >
            <div class="flex items-start justify-between gap-2">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <span
                    class="text-sm font-medium"
                    :class="[
                      selectedSpec?.id === spec.id
                        ? 'text-accent'
                        : 'text-control',
                    ]"
                  >
                    {{
                      $t("plan.navigator.spec-number", { number: index + 1 })
                    }}
                    <span
                      v-if="isSpecEmpty(spec)"
                      class="text-error ml-0.5"
                      :title="$t('plan.navigator.statement-empty')"
                    >
                      *
                    </span>
                  </span>
                  <SpecStatusBadge
                    v-if="getSpecCheckStatus(spec) !== 'STATUS_UNSPECIFIED'"
                    :status="getSpecCheckStatus(spec)"
                  />
                </div>

                <div
                  class="flex flex-row items-center gap-1 mt-1 text-xs text-control-light"
                >
                  <div class="truncate">
                    {{ getSpecTypeName(spec) }}
                  </div>
                  <span class="mx-0.5">·</span>
                  <div class="opacity-80">
                    {{ getTargetCountText(spec) }}
                  </div>
                </div>
              </div>

              <ChevronRightIcon
                v-if="selectedSpec?.id === spec.id"
                class="w-4 h-4 text-accent mt-0.5 flex-shrink-0"
              />
            </div>
          </div>
        </div>

        <div
          v-if="specs.length === 0"
          class="text-sm text-control-light text-center py-8 px-4"
        >
          {{ $t("plan.navigator.no-specifications") }}
        </div>
      </div>
    </NScrollbar>

    <!-- Add Spec Drawer -->
    <AddSpecDrawer
      v-model:show="showAddSpecDrawer"
      @created="handleSpecCreated"
    />
  </div>
</template>

<script setup lang="ts">
import {
  LayoutDashboardIcon,
  ChevronRightIcon,
  PlusIcon,
} from "lucide-vue-next";
import { NButton, NScrollbar, NBadge } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter, useRoute } from "vue-router";
import {
  Plan_ChangeDatabaseConfig_Type,
  type Plan_Spec,
  type PlanCheckRun,
} from "@/types/proto/v1/plan_service";
import { usePlanContext } from "../logic/context";
import { targetsForSpec } from "../logic/plan";
import AddSpecDrawer from "./AddSpecDrawer.vue";
import SpecStatusBadge from "./SpecStatusBadge.vue";
import { useSpecsValidation } from "./common/validateSpec";

const router = useRouter();
const route = useRoute();
const { t } = useI18n();
const { plan, selectedSpec, isCreating, events, planCheckRunList } =
  usePlanContext();

const specs = computed(() => plan.value.specs);
const showAddSpecDrawer = ref(false);

// Use the validation hook for all specs
const { isSpecEmpty } = useSpecsValidation(specs.value);

const handleSelectOverview = () => {
  // Clear spec selection to show overview
  router.replace({
    query: {
      ...route.query,
      spec: undefined,
      target: undefined,
    },
    hash: route.hash,
  });
};

const handleSelectSpec = (spec: Plan_Spec) => {
  events.emit("select-spec", { spec });
};

const handleAddSpec = () => {
  showAddSpecDrawer.value = true;
};

const handleSpecCreated = (spec: Plan_Spec) => {
  // Add the new spec to the plan
  if (!plan.value?.specs) {
    plan.value.specs = [];
  }
  plan.value.specs.push(spec);

  // Select the newly created spec
  events.emit("select-spec", { spec });
};

const getSpecCheckStatus = (spec: Plan_Spec) => {
  // Get aggregated check status for the spec
  const checkRuns = planCheckRunList.value || [];
  const targets = targetsForSpec(spec);
  const sheet = getSheetFromSpec(spec);

  if (!sheet || targets.length === 0) {
    return "STATUS_UNSPECIFIED";
  }

  const relevantCheckRuns = checkRuns.filter((check: PlanCheckRun) => {
    return targets.includes(check.target) && check.sheet === sheet;
  });

  if (relevantCheckRuns.length === 0) {
    return "STATUS_UNSPECIFIED";
  }

  // Find worst status
  let hasError = false;
  let hasWarning = false;

  for (const checkRun of relevantCheckRuns) {
    for (const result of checkRun.results) {
      if (result.status === "ERROR") {
        hasError = true;
      } else if (result.status === "WARNING") {
        hasWarning = true;
      }
    }
  }

  if (hasError) return "ERROR";
  if (hasWarning) return "WARNING";
  return "SUCCESS";
};

const getSheetFromSpec = (spec: Plan_Spec): string | undefined => {
  if (spec.changeDatabaseConfig) {
    return spec.changeDatabaseConfig.sheet;
  } else if (spec.exportDataConfig) {
    return spec.exportDataConfig.sheet;
  }
  return undefined;
};

const getSpecTypeName = (spec: Plan_Spec): string => {
  if (spec.createDatabaseConfig) {
    return t("plan.spec.type.create-database");
  } else if (spec.changeDatabaseConfig) {
    const changeType = spec.changeDatabaseConfig.type;
    switch (changeType) {
      case Plan_ChangeDatabaseConfig_Type.MIGRATE:
        return t("plan.spec.type.schema-change");
      case Plan_ChangeDatabaseConfig_Type.DATA:
        return t("plan.spec.type.data-change");
      case Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST:
        return t("plan.spec.type.ghost-migration");
      default:
        return t("plan.spec.type.database-change");
    }
  } else if (spec.exportDataConfig) {
    return t("plan.spec.type.export-data");
  }
  return t("plan.spec.type.unknown");
};

const getTargetCountText = (spec: Plan_Spec): string => {
  const targets = targetsForSpec(spec);
  if (targets.length === 0) return t("plan.targets.no-targets");
  return targets.length === 1
    ? t("plan.targets.one-target")
    : t("plan.targets.multiple-targets", { count: targets.length });
};
</script>
