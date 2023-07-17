<template>
  <div v-if="visible">
    <template v-if="readonly || !state.manualEdit">
      <div v-if="state.resourceId" class="textinfolabel text-sm">
        {{ $t("resource-id.self", { resource: resourceName }) }}:
        <span class="text-gray-600 font-medium mr-1">{{
          state.resourceId
        }}</span>
        <template v-if="!readonly">
          {{ $t("resource-id.cannot-be-changed-later") }}
          <span
            class="text-accent font-medium cursor-pointer hover:opacity-80"
            @click="state.manualEdit = true"
          >
            {{ $t("common.edit") }}
          </span>
        </template>
      </div>
    </template>
    <template v-else>
      <label for="name" class="textlabel">
        {{ $t("resource-id.self", { resource: resourceName }) }}
        <span class="ml-0.5 text-error">*</span>
      </label>
      <div class="textinfolabel my-2">
        {{ $t("resource-id.description", { resource: resourceName }) }}
      </div>
      <NInput
        :value="state.resourceId"
        :status="inputStatus"
        v-bind="inputProps"
        @update:value="handleResourceIdInput($event)"
      />
    </template>
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
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { NInput, type InputProps } from "naive-ui";

import type { ResourceId, ValidatedMessage } from "@/types";
import { randomString } from "@/utils";

// characters is the validated characters for resource id.
const characters = "abcdefghijklmnopqrstuvwxyz1234567890-";

// randomCharacter returns a random character from the english alphabet.
const randomCharacter = (ch?: string): string => {
  const characters = "abcdefghijklmnopqrstuvwxyz";
  const index = ch
    ? ch.charCodeAt(0) % characters.length
    : Math.floor(Math.random() * characters.length);
  return characters.charAt(index);
};

const resourceIdPattern = /^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$/;

interface LocalState {
  resourceId: string;
  manualEdit: boolean;
  validatedMessages: ValidatedMessage[];
}

type ResourceType =
  | "environment"
  | "instance"
  | "project"
  | "idp"
  | "role"
  | "database-group"
  | "schema-group";

const props = withDefaults(
  defineProps<{
    value?: string;
    resourceType: ResourceType;
    resourceTitle?: string;
    suffix?: boolean;
    readonly?: boolean;
    inputProps?: Partial<InputProps>;
    validate?: (resourceId: ResourceId) => Promise<ValidatedMessage[]>;
  }>(),
  {
    value: "",
    resourceTitle: "",
    suffix: false,
    readonly: false,
    inputProps: undefined,
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
let initialized = false;
// Won't change after the component instance initialized.
const randomSuffix = randomString(4).toLowerCase();

const resourceName = computed(() => {
  return t(`resource.${props.resourceType}`);
});

const visible = computed(() => {
  if (props.readonly || !state.manualEdit) {
    return !!state.resourceId;
  }
  return true;
});

const inputStatus = computed(() => {
  const { validatedMessages } = state;
  if (validatedMessages.some((m) => m.type === "error")) return "error";
  if (validatedMessages.some((m) => m.type === "warning")) return "warning";
  return undefined;
});

const handleResourceIdInput = (newValue: string) => {
  if (!state.manualEdit) {
    return;
  }

  state.validatedMessages = [];
  handleResourceIdChange(newValue);
};

const handleResourceIdChange = async (newValue: string) => {
  state.resourceId = newValue;
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
  if (props.validate) {
    const messages = await props.validate(state.resourceId);
    if (Array.isArray(messages)) {
      state.validatedMessages.push(...messages);
    }
  }
};

watch(
  () => props.value,
  (newValue) => {
    state.resourceId = newValue;
  }
);

const escape = (str: string) => {
  return str
    .toLowerCase()
    .split("")
    .map((char) => {
      if (char == " ") return "-";
      if (char.match(/\s/)) return "";
      if (characters.includes(char)) return char;
      return randomCharacter(char);
    })
    .join("")
    .toLowerCase();
};

watch(
  () => props.resourceTitle,
  async (resourceTitle) => {
    if (props.readonly) {
      return;
    }
    if (state.manualEdit) {
      return;
    }

    // If we are not in manual edit mode, update the auto-generated resource id
    // according to resource title.
    const parts: string[] = [];
    if (resourceTitle) {
      const escapedTitle = escape(resourceTitle);
      if (props.suffix) {
        parts.push(escapedTitle, randomSuffix);
      } else if (escapedTitle) {
        parts.push(escapedTitle);
      } else {
        parts.push(randomString(4).toLowerCase());
      }
    }
    const name = parts.join("-");
    await handleResourceIdChange(name);

    // We should keep the first auto-generated resource id is valid.
    if (!initialized) {
      const messages = state.validatedMessages;
      if (messages.length > 0) {
        await handleResourceIdChange(
          name + "-" + randomString(4).toLowerCase()
        );
        return;
      }
    }
    initialized = true;
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
