<template>
  <div class="space-y-4">
    <div
      v-for="(config, index) in rule.componentList"
      :key="index"
      class="space-y-1"
    >
      <p
        :class="[
          'font-medium',
          size !== 'small' && 'text-lg text-control mb-2',
        ]"
      >
        {{ configTitle(config) }}
      </p>
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
        v-else-if="config.payload.type == 'BOOLEAN'"
        :title="configTitle(config)"
        :value="payload[index] as boolean"
        :config="config"
        :disabled="disabled"
        @update:value="payload[index] = $event"
      />
      <StringArrayComponent
        v-else-if="
          config.payload.type == 'STRING_ARRAY' && Array.isArray(payload[index])
        "
        :value="payload[index] as string[]"
        :config="config"
        :disabled="disabled"
        @update:value="payload[index] = $event"
      />
      <TemplateComponent
        v-else-if="config.payload.type == 'TEMPLATE'"
        :rule-type="rule.type"
        :value="payload[index] as string"
        :config="config"
        :disabled="disabled"
        @update:value="payload[index] = $event"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { RuleConfigComponent, RuleTemplateV2 } from "@/types/sqlReview";
import { getRuleLocalizationKey } from "@/types/sqlReview";
import type { PayloadValueType } from "./types";
import { getRulePayload } from "./utils";

const props = defineProps<{
  rule: RuleTemplateV2;
  disabled: boolean;
  size: "small" | "medium";
}>();

const { t } = useI18n();
const payload = ref<PayloadValueType[]>([]);

const configTitle = (config: RuleConfigComponent): string => {
  const key = `sql-review.rule.${getRuleLocalizationKey(
    props.rule.type
  )}.component.${config.key}.title`;
  return t(key);
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
