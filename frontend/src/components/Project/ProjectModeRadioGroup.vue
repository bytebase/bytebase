<template>
  <div class="radio-set-row">
    <NRadioGroup
      :disabled="disabled"
      :value="value"
      @update:value="$emit('update:value', $event)"
    >
      <NRadio :value="TenantMode.TENANT_MODE_DISABLED">
        {{ $t("project.mode.standard") }}
      </NRadio>
      <NRadio :value="TenantMode.TENANT_MODE_ENABLED">
        <div class="flex space-x-1">
          <span>
            {{ $t("project.mode.batch") }}
          </span>
          <LearnMoreLink
            url="https://www.bytebase.com/docs/change-database/batch-change/?source=console"
          />
          <FeatureBadge feature="bb.feature.multi-tenancy" />
        </div>
      </NRadio>
    </NRadioGroup>
  </div>
</template>

<script setup lang="ts">
import { NRadio, NRadioGroup } from "naive-ui";
import { TenantMode } from "@/types/proto/v1/project_service";

withDefaults(
  defineProps<{
    value: TenantMode;
    disabled?: boolean;
  }>(),
  {
    disabled: false,
  }
);

defineEmits<{
  (event: "update:value", value: TenantMode): void;
}>();
</script>
