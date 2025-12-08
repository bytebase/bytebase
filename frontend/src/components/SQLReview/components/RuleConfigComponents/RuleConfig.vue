<template>
  <div class="flex flex-col gap-y-4">
    <div
      v-for="(config, index) in rule.componentList"
      :key="index"
      class="flex flex-col gap-y-1"
    >
      <div
        v-if="config.payload.type !== 'BOOLEAN'"
        class="flex items-center gap-x-1"
      >
        <p
          :class="[
            'font-medium',
            size !== 'small' && 'text-lg text-control mb-2',
          ]"
        >
          {{ configTitle(config) }}
        </p>
        <NTooltip v-if="configTooltip(config)">
          <template #trigger>
            <CircleHelpIcon class="w-4" />
          </template>
          <span>{{ configTooltip(config) }}</span>
        </NTooltip>
      </div>
      <StringComponent
        v-if="config.payload.type === 'STRING'"
        :value="payload[index] as string"
        :config="config"
        :disabled="disabled"
        @update:value="payload[index] = $event"
      />
      <NumberComponent
        v-if="config.payload.type === 'NUMBER'"
        :value="payload[index] as number"
        :config="config"
        :disabled="disabled"
        @update:value="payload[index] = $event"
      />
      <BooleanComponent
        v-else-if="config.payload.type === 'BOOLEAN'"
        :title="configTitle(config)"
        :tooltip="configTooltip(config)"
        :value="payload[index] as boolean"
        :config="config"
        :disabled="disabled"
        @update:value="payload[index] = $event"
      />
      <StringArrayComponent
        v-else-if="
          config.payload.type === 'STRING_ARRAY' &&
          Array.isArray(payload[index])
        "
        :value="payload[index] as string[]"
        :config="config"
        :disabled="disabled"
        @update:value="payload[index] = $event"
      />
      <TemplateComponent
        v-else-if="config.payload.type === 'TEMPLATE'"
        :rule-type="ruleTypeToString(rule.type)"
        :value="payload[index] as string"
        :config="config"
        :disabled="disabled"
        @update:value="payload[index] = $event"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { CircleHelpIcon } from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { RuleConfigComponent, RuleTemplateV2 } from "@/types/sqlReview";
import { getRuleLocalizationKey, ruleTypeToString } from "@/types/sqlReview";
import BooleanComponent from "./BooleanComponent.vue";
import NumberComponent from "./NumberComponent.vue";
import StringArrayComponent from "./StringArrayComponent.vue";
import StringComponent from "./StringComponent.vue";
import TemplateComponent from "./TemplateComponent.vue";
import type { PayloadValueType } from "./types";
import { getRulePayload } from "./utils";

const props = defineProps<{
  rule: RuleTemplateV2;
  disabled: boolean;
  size: "small" | "medium";
}>();

const { t, te } = useI18n();
const payload = ref<PayloadValueType[]>([]);

const configTitle = (config: RuleConfigComponent): string => {
  const key = `sql-review.rule.${getRuleLocalizationKey(
    ruleTypeToString(props.rule.type)
  )}.component.${config.key}.title`;
  return t(key);
};

const configTooltip = (config: RuleConfigComponent): string => {
  const key = `sql-review.rule.${getRuleLocalizationKey(
    ruleTypeToString(props.rule.type)
  )}.component.${config.key}.tooltip`;
  return te(key) ? t(key) : "";
};

watch(
  () => props.rule.componentList,
  () => (payload.value = getRulePayload(props.rule)),
  { immediate: true }
);

defineExpose({
  payload,
});
</script>
