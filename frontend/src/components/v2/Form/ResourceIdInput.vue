<template>
  <input
    v-model="state.resourceId"
    required
    class="textfield w-full"
    type="text"
    :disabled="readonly"
    :readonly="readonly"
  />

  <ul
    v-if="state.validatedMessages.length > 0"
    class="w-full my-2 space-y-2 list-disc list-outside pl-4"
  >
    <li
      v-for="validateMessage in state.validatedMessages"
      :key="validateMessage.message"
      class="break-words w-full text-xs"
      :class="[
        validateMessage.type === 'warning' && 'text-yellow-600',
        validateMessage.type === 'error' && 'text-red-600',
      ]"
    >
      {{ validateMessage.message }}
    </li>
  </ul>
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useDebounceFn } from "@vueuse/core";

import type { ResourceId, ValidatedMessage } from "@/types";

const resourceIdPattern = /^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$/;

interface LocalState {
  resourceId: string;
  manualEdit: boolean;
  validatedMessages: ValidatedMessage[];
}

type ResourceType = "database-group" | "schema-group";

const props = withDefaults(
  defineProps<{
    value?: string;
    resourceType: ResourceType;
    readonly?: boolean;
    validate?: (resourceId: ResourceId) => Promise<ValidatedMessage[]>;
  }>(),
  {
    value: "",
    readonly: false,
    validate: () => Promise.resolve([]),
  }
);

const emit = defineEmits<{
  (event: "update:value", value: string): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  resourceId: props.value,
  manualEdit: false,
  validatedMessages: [],
});

const resourceName = computed(() => {
  return t(`resource.${props.resourceType}`);
});

const handleResourceIdChange = useDebounceFn(async (newValue: string) => {
  if (props.readonly) {
    return;
  }

  state.validatedMessages = [];

  emit("update:value", newValue);

  // if the resource id is empty, do not validate.
  if (newValue === "" && !state.manualEdit) {
    return;
  }

  // common validation for resource id (min length, max length, pattern).
  if (state.resourceId.length < 1) {
    state.validatedMessages.push({
      type: "error",
      message: t("resource-id.validation.minlength", {
        resource: resourceName.value,
      }),
    });
  } else if (state.resourceId.length > 64) {
    state.validatedMessages.push({
      type: "error",
      message: t("resource-id.validation.overflow", {
        resource: resourceName.value,
      }),
    });
  } else if (!resourceIdPattern.test(state.resourceId)) {
    state.validatedMessages.push({
      type: "error",
      message: t("resource-id.validation.pattern", {
        resource: resourceName.value,
      }),
    });
  }

  // custom validation for resource id. (e.g. check if the resource id is already used)
  if (state.validatedMessages.length === 0 && props.validate) {
    const messages = await props.validate(state.resourceId);
    if (Array.isArray(messages)) {
      state.validatedMessages.push(...messages);
    }
  }
}, 500);

watch(
  () => props.value,
  (newValue) => {
    state.resourceId = newValue;
  }
);

watch(
  () => state.resourceId,
  (newValue) => {
    handleResourceIdChange(newValue);
  },
  {
    immediate: true,
  }
);

defineExpose({
  resourceId: computed(() => state.resourceId),
  isValidated: computed(() => {
    return state.validatedMessages.length === 0;
  }),
});
</script>
