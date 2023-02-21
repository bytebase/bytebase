<template>
  <div
    v-bind="$attrs"
    class="w-full max-w-full"
    :class="[shouldShowResourceIdField ? 'mt-6' : '']"
  >
    <template v-if="!shouldShowResourceIdField">
      <p v-if="state.resourceId" class="w-full text-gray-500 text-sm pt-2">
        {{ $t("resource-id.self", { resource: resourceName }) }}:
        <span class="text-gray-600 font-medium mr-1">{{
          state.resourceId
        }}</span>
        <template v-if="!readonly">
          {{ $t("resource-id.cannot-be-changed-later") }}
          <span
            class="text-accent font-medium cursor-pointer hover:opacity-80"
            @click="state.isResourceIdChanged = true"
          >
            {{ $t("common.edit") }}
          </span>
        </template>
      </p>
    </template>
    <template v-else>
      <label for="name" class="textlabel">
        {{ $t("resource-id.self", { resource: resourceName }) }}
        <span class="text-red-600">*</span>
      </label>
      <div class="mt-1 textinfolabel">
        {{ $t("resource-id.description", { resource: resourceName }) }}
      </div>
      <BBTextField
        class="mt-2 w-full"
        :value="state.resourceId"
        @input="
          handleResourceIdInput(($event.target as HTMLInputElement).value)
        "
      />
      <ul class="w-full my-2 space-y-2 list-disc list-outside pl-4">
        <li
          v-for="validateMessage in state.validateMessages"
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
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { randomString } from "@/utils";

const characters = "abcdefghijklmnopqrstuvwxyz1234567890";

const resourceIdPattern = /^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$/;

interface ValidateMessage {
  type: "warning" | "error";
  message: string;
}

interface LocalState {
  resourceId: string;
  isResourceIdChanged: boolean;
  validateMessages: ValidateMessage[];
}

const props = withDefaults(
  defineProps<{
    resource: "environment";
    defaultValue: string;
    resourceTitle: string;
    readonly: boolean;
  }>(),
  {
    defaultValue: "",
    resourceTitle: "",
    readonly: false,
  }
);

const { t } = useI18n();
const state = reactive<LocalState>({
  resourceId: props.defaultValue,
  isResourceIdChanged: false,
  validateMessages: [],
});

const getPrefix = (resource: string) => {
  switch (resource) {
    case "environment":
      return "env";
    default:
      return "";
  }
};
const randomSuffix = randomString(4).toLowerCase();

const resourceName = computed(() => {
  return t(`common.${props.resource}`);
});

const shouldShowResourceIdField = computed(() => {
  return !props.readonly && state.isResourceIdChanged;
});

watch(
  () => props.resourceTitle,
  (newValue) => {
    if (props.readonly) {
      return;
    }

    if (!state.isResourceIdChanged) {
      const formatedTitle = newValue
        .toLowerCase()
        .split("")
        .map((char) => {
          if (char === " ") {
            return "-";
          }
          if (characters.includes(char)) {
            return char;
          }
          return randomString(1);
        })
        .join("")
        .toLowerCase();

      state.resourceId = `${getPrefix(props.resource)}-${
        formatedTitle || randomString(4).toLowerCase()
      }-${randomSuffix}`;
    }
  }
);

const handleResourceIdInput = (newValue: string) => {
  if (!state.isResourceIdChanged) {
    return;
  }

  state.resourceId = newValue;
  state.validateMessages = [];
  if (state.resourceId.length < 1) {
    state.validateMessages.push({
      type: "error",
      message: t("resource-id.validation.minlength", {
        resource: resourceName.value,
      }),
    });
  }
  if (state.resourceId.length > 64) {
    state.validateMessages.push({
      type: "error",
      message: t("resource-id.validation.overflow", {
        resource: resourceName.value,
      }),
    });
  }
  if (!resourceIdPattern.test(state.resourceId)) {
    state.validateMessages.push({
      type: "error",
      message: t("resource-id.validation.pattern", {
        resource: resourceName.value,
      }),
    });
  }
};

defineExpose({
  resourceId: computed(() => state.resourceId),
});
</script>
