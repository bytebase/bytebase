<template>
  <div class="w-full flex flex-col gap-y-2">
    <span class="text-base">{{ $t("issue.data-export.limits") }}</span>

    <div class="flex items-center gap-x-2">
      <span class="text-sm">
        {{ $t("settings.general.workspace.maximum-sql-result.size.self") }}
      </span>
      <span class=" font-medium">
        {{ Number(effectiveQueryDataPolicy.maximumResultSize) / 1024 / 1024 }} MB
      </span>
    </div>
    <div class="flex items-center gap-x-2">
      <span class="text-sm">
        {{ $t("settings.general.workspace.maximum-sql-result.rows.self") }}
      </span>
      <span class=" font-medium">
        {{ maximumResultRows }}
      </span>
    </div>
  </div>
</template>

<script lang="tsx" setup>
import { computed } from "vue";
import { t } from "@/plugins/i18n";
import { useCurrentProjectV1, usePolicyV1Store } from "@/store";

const policyStore = usePolicyV1Store();
const { project } = useCurrentProjectV1();

const effectiveQueryDataPolicy = computed(() => {
  return policyStore.getEffectiveQueryDataPolicyForProject(project.value.name);
});

const maximumResultRows = computed(() => {
  const { maximumResultRows } = effectiveQueryDataPolicy.value;
  if (maximumResultRows === Number.MAX_VALUE) {
    return t("common.unlimited");
  }
  return `${maximumResultRows} ${t("settings.general.workspace.maximum-sql-result.rows.rows")}`;
});
</script>