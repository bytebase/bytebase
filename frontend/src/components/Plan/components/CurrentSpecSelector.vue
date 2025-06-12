<template>
  <div v-if="shouldShow" class="flex flex-row justify-start items-center">
    <NSelect
      size="small"
      :options="options"
      :value="selectedSpec.id"
      :render-label="labelRender"
      :consistent-menu-width="false"
      :bordered="false"
      @update-value="handleSpecChange"
    />
    <NButton
      v-if="isCreating"
      type="default"
      size="small"
      @click="showAddSpecDrawer = true"
    >
      <template #icon>
        <PlusIcon class="w-4 h-4" />
      </template>
    </NButton>
  </div>

  <AddSpecDrawer
    v-model:show="showAddSpecDrawer"
    @created="handleSpecCreated"
  />
</template>

<script setup lang="tsx">
import { head } from "lodash-es";
import { PlusIcon } from "lucide-vue-next";
import { NButton, NSelect } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import {
  Plan_ChangeDatabaseConfig_Type,
  type Plan_Spec,
} from "@/types/proto/v1/plan_service";
import { usePlanContext } from "../logic";
import AddSpecDrawer from "./AddSpecDrawer.vue";
import { gotoSpec } from "./common/utils";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { isCreating, plan } = usePlanContext();
const showAddSpecDrawer = ref(false);

const shouldShow = computed(() => {
  return isCreating.value || plan.value.specs.length > 1;
});

const specs = computed(() => plan.value.specs);

const options = computed(() => {
  return specs.value.map((spec) => {
    return {
      label: getSpecTypeName(spec),
      value: spec.id,
    };
  });
});

const labelRender = (option: { label: string; value: string }) => {
  const spec = specs.value.find((s) => s.id === option.value);
  if (!spec) {
    return option.label;
  }
  const index = specs.value.indexOf(spec) + 1;
  const suffix = specs.value.length > 1 ? ` #${index}` : "";
  return (
    <span>
      {option.label}
      {suffix && <span class="text-gray-500">{suffix}</span>}
    </span>
  );
};

const selectedSpec = computed(() => {
  return (specs.value.find((spec) => spec.id === route.params.specId) ||
    head(specs.value)) as Plan_Spec;
});

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

const handleSpecChange = (specId: string) => {
  gotoSpec(router, specId);
};

const handleSpecCreated = async (spec: Plan_Spec) => {
  // Add the new spec to the plan
  plan.value.specs.push(spec);
  gotoSpec(router, spec.id);
};
</script>
