<template>
  <div
    v-bind="$attrs"
    class="w-full max-w-full"
    :class="[shouldShowResourceIdField ? 'mt-4' : '']"
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
import { useDebounceFn } from "@vueuse/core";
import { randomString } from "@/utils";
import { ResourceId } from "@/types";

const characters = "abcdefghijklmnopqrstuvwxyz1234567890";

const resourceIdPattern = /^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$/;

interface ValidatedMessage {
  type: "warning" | "error";
  message: string;
}

interface LocalState {
  resourceId: string;
  isResourceIdChanged: boolean;
  validatedMessages: ValidatedMessage[];
}

type ResourceType = "environment" | "instance" | "project" | "idp";

const props = withDefaults(
  defineProps<{
    resource: ResourceType;
    value: string;
    resourceTitle: string;
    readonly: boolean;
    randomString: boolean;
    validator: (resourceId: ResourceId) => Promise<string | undefined>;
  }>(),
  {
    value: "",
    resourceTitle: "",
    randomString: false,
    readonly: false,
    validator: () => Promise.resolve(undefined),
  }
);

const { t } = useI18n();
const state = reactive<LocalState>({
  resourceId: props.value,
  isResourceIdChanged: false,
  validatedMessages: [],
});

const resourceName = computed(() => {
  return t(`resource.${props.resource}`);
});

const getPrefix = (resource: string) => {
  switch (resource) {
    case "environment":
      return "env";
    case "instance":
      return "ins";
    case "project":
      return "proj";
    case "idp":
      return "idp";
    default:
      return "";
  }
};
const randomSuffix = randomString(4).toLowerCase();

const shouldShowResourceIdField = computed(() => {
  return !props.readonly && state.isResourceIdChanged;
});

const handleResourceIdInput = (newValue: string) => {
  if (!state.isResourceIdChanged) {
    return;
  }

  state.validatedMessages = [];
  debounceHandleResourceIdChange(newValue);
};

const debounceHandleResourceIdChange = useDebounceFn(
  async (newValue: string) => {
    state.resourceId = newValue;
    state.validatedMessages = [];

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
    if (props.validator) {
      const message = await props.validator(state.resourceId);
      if (message) {
        state.validatedMessages.push({
          type: "error",
          message,
        });
      }
    }
  },
  200
);

watch(
  () => props.value,
  (newValue) => {
    state.resourceId = newValue;
  }
);

watch(
  () => props.resourceTitle,
  (resourceTitle) => {
    if (props.readonly) {
      return;
    }

    if (!state.isResourceIdChanged) {
      if (resourceTitle) {
        const formatedTitle = resourceTitle
          .toLowerCase()
          .split("")
          .map((char) => {
            if (char === " ") {
              return "";
            }
            if (characters.includes(char)) {
              return char;
            }
            return randomString(1);
          })
          .join("")
          .toLowerCase();

        let name = "";
        if (props.randomString) {
          name = `${getPrefix(
            props.resource
          )}-${formatedTitle}-${randomSuffix}`;
        } else {
          name = `${formatedTitle || randomString(4).toLowerCase()}`;
        }
        debounceHandleResourceIdChange(name);
      } else {
        state.resourceId = "";
        state.validatedMessages = [];
      }
    }
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
