<template>
  <div
    v-if="state.visible"
    v-zindexable="{ enabled: true }"
    class="fixed inset-0 overflow-y-auto"
  >
    <div
      class="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0"
    >
      <!--
      Background overlay, show/hide based on modal state.

      Entering: "ease-out duration-300"
        From: "opacity-0"
        To: "opacity-100"
      Leaving: "ease-in duration-200"
        From: "opacity-100"
        To: "opacity-0"
    -->
      <div class="fixed inset-0 transition-opacity" aria-hidden="true">
        <div class="absolute inset-0 bg-gray-500 opacity-75"></div>
      </div>

      <!-- This element is to trick the browser into centering the modal contents. -->
      <span
        class="hidden sm:inline-block sm:align-middle sm:h-screen"
        aria-hidden="true"
        >&#8203;</span
      >
      <!--
      Modal panel, show/hide based on modal state.

      Entering: "ease-out duration-300"
        From: "opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
        To: "opacity-100 translate-y-0 sm:scale-100"
      Leaving: "ease-in duration-200"
        From: "opacity-100 translate-y-0 sm:scale-100"
        To: "opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
    -->
      <div
        class="inline-block align-bottom bg-white rounded-lg px-4 pt-5 pb-4 text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full sm:p-6"
        role="dialog"
        aria-modal="true"
        aria-labelledby="modal-headline"
      >
        <div class="sm:flex sm:flex-col sm:items-start">
          <div class="flex items-start">
            <div
              v-if="style == 'SUCCESS'"
              class="mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full bg-green-100 sm:mx-0 sm:h-10 sm:w-10"
            >
              <!-- Heroicons name: outline/exclamation -->
              <heroicons-outline:exclamation class="h-6 w-6 text-success" />
            </div>
            <div
              v-else-if="style == 'WARN'"
              class="mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full bg-yellow-100 sm:mx-0 sm:h-10 sm:w-10"
            >
              <!-- Heroicons name: outline/exclamation -->
              <heroicons-outline:exclamation class="h-6 w-6 text-yellow-600" />
            </div>
            <div
              v-else-if="style == 'CRITICAL'"
              class="mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full bg-red-100 sm:mx-0 sm:h-10 sm:w-10"
            >
              <!-- Heroicons name: outline/exclamation -->
              <heroicons-outline:exclamation class="h-6 w-6 text-red-600" />
            </div>
            <div
              v-else
              class="mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full bg-blue-100 sm:mx-0 sm:h-10 sm:w-10"
            >
              <!-- Heroicons name: outline/exclamation -->
              <heroicons-outline:exclamation class="h-6 w-6 text-blue-600" />
            </div>
            <h3
              id="modal-headline"
              class="ml-4 flex self-center text-lg leading-6 font-medium text-gray-900"
            >
              {{ $t(title) }}
            </h3>
          </div>
          <div
            v-if="description"
            class="mt-3 text-center sm:mt-0 sm:ml-14 sm:text-left"
          >
            <div class="mt-2">
              <p class="text-gray-500 whitespace-pre-wrap">
                {{ $t(description) }}
              </p>
            </div>
          </div>
        </div>
        <div class="mt-5 flex flex-col space-y-4">
          <div class="flex justify-end">
            <button
              type="button"
              class="btn-normal mt-3 px-4 py-2 sm:mt-0 sm:w-auto"
              data-label="bb-alert-cancel-button"
              :disabled="inProgress"
              @click.prevent="onNegativeClick"
            >
              {{ $t(cancelText) }}
            </button>
            <button
              type="button"
              class="sm:ml-3 inline-flex justify-center w-full rounded-md border border-transparent shadow-sm px-4 py-2 bg-error text-base font-medium text-white focus:outline-none focus-visible:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:w-auto sm:text-sm"
              data-label="bb-alert-ok-button"
              :class="okButtonStyle"
              :disabled="inProgress"
              @click.prevent="onPositiveClick()"
            >
              {{ $t(okText) }}
            </button>
          </div>
          <div v-if="inProgress" class="flex flex-row justify-end items-center">
            <BBSpin />{{ progressText }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useEventListener } from "@vueuse/core";
import { zindexable as vZindexable } from "vdirs";
import { withDefaults, computed, reactive } from "vue";
import { Defer, defer } from "@/utils";
import { BBAlertStyle } from "./types";

const props = withDefaults(
  defineProps<{
    style?: BBAlertStyle;
    title: string;
    description?: string;
    okText?: string;
    cancelText?: string;
    inProgress?: boolean;
    progressText?: string;
    onBeforePositiveClick?: () => boolean;
    onBeforeNegativeClick?: () => boolean;
  }>(),
  {
    style: "INFO",
    description: "",
    okText: "bbkit.common.ok",
    cancelText: "bbkit.common.cancel",
    inProgress: false,
    progressText: "",
    onBeforeNegativeClick: undefined,
    onBeforePositiveClick: undefined,
  }
);

const state = reactive({
  visible: false,
  defer: undefined as Defer<boolean> | undefined,
});

const open = () => {
  if (state.defer) {
    state.defer.reject(new Error("duplicated call"));
  }

  state.defer = defer<boolean>();
  state.visible = true;

  return state.defer.promise;
};

const onPositiveClick = () => {
  if (props.onBeforePositiveClick) {
    if (!props.onBeforePositiveClick()) {
      return;
    }
  }

  state.visible = false;
  state.defer?.resolve(true);
};

const onNegativeClick = () => {
  if (props.onBeforeNegativeClick) {
    if (!props.onBeforeNegativeClick()) {
      return;
    }
  }

  state.visible = false;
  state.defer?.resolve(false);
};

defineExpose({ open });

useEventListener(document, "keydown", (e) => {
  if (!props.inProgress && e.code == "Escape") {
    onNegativeClick();
  }
});

const buttonStyleMap: Record<string, string> = {
  INFO: "btn-primary",
  SUCCESS: "btn-success",
  WARN: "btn-primary",
  CRITICAL: "btn-danger",
};

const okButtonStyle = computed(() => buttonStyleMap[props.style]);
</script>
