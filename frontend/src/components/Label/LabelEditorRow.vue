<template>
  <div class="flex flex-col gap-y-1">
    <div class="text-sm flex gap-x-2">
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
        <div class="flex items-center gap-x-2">
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
          <MiniActionButton
            :class="['ml-1', readonly ? 'invisible' : 'visible']"
            @click="$emit('remove')"
          >
            <Trash2Icon />
          </MiniActionButton>
        </div>
        <div v-if="kv.message" class="textinfolabel">{{ kv.message }}</div>
      </div>
      <div />
    </div>
    <div v-if="combinedErrors.length > 0" class="text-xs text-error col-span-3">
      <ErrorList :errors="combinedErrors" bullets="always" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { Trash2Icon } from "lucide-vue-next";
import { NInput } from "naive-ui";
import { computed } from "vue";
import { MiniActionButton } from "@/components/v2";
import ErrorList from "../misc/ErrorList.vue";
import type { Label } from "./types";

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
