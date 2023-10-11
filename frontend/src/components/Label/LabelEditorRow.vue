<template>
  <div class="contents text-sm space-x-2">
    <div class="flex flex-col">
      <span class="text-xs font-medium mb-1"> Key {{ index + 1 }} </span>
      <span v-if="readonly" class="leading-[34px]">
        {{ kv.key }}
      </span>
      <NInput
        v-else
        :value="kv.key"
        :placeholder="$t('setting.label.key-placeholder')"
        :status="errors.key.length > 0 ? 'error' : undefined"
        @update:value="$emit('update-key', $event)"
      />
    </div>
    <div class="flex flex-col">
      <span class="text-xs font-medium mb-1"> Value {{ index + 1 }} </span>
      <div class="flex items-center space-x-2">
        <span v-if="readonly" class="leading-[34px]">
          <template v-if="kv.value">{{ kv.value }}</template>
          <span v-else class="text-control-placeholder">
            {{ $t("label.empty-label-value") }}
          </span>
        </span>
        <NInput
          v-else
          :value="kv.value"
          :placeholder="$t('setting.label.value-placeholder')"
          :status="errors.value.length > 0 ? 'error' : undefined"
          @update:value="$emit('update-value', $event)"
        />
        <NButton
          quaternary
          size="small"
          style="--n-padding: 0 6px"
          :class="['ml-1', readonly ? 'invisible' : 'visible']"
          @click="$emit('remove')"
        >
          <template #icon>
            <heroicons:trash />
          </template>
        </NButton>
      </div>
      <div v-if="kv.message" class="textinfolabel">{{ kv.message }}</div>
    </div>
    <div />
  </div>
  <div v-if="combinedErrors.length > 0" class="text-xs text-error col-span-3">
    <ErrorList :errors="combinedErrors" bullets="always" />
  </div>
</template>

<script setup lang="ts">
import { NButton, NInput } from "naive-ui";
import { computed } from "vue";
import ErrorList from "../misc/ErrorList.vue";
import { Label } from "./types";

const props = defineProps<{
  kv: Label;
  index: number;
  readonly: boolean;
  errors: {
    key: string[];
    value: string[];
  };
}>();

defineEmits<{
  (event: "update-key", key: string): void;
  (event: "update-value", value: string): void;
  (event: "remove"): void;
}>();

const combinedErrors = computed(() => {
  return [...props.errors.key, ...props.errors.value];
});
</script>
