<template>
  <div class="fixed z-10 inset-0 overflow-y-auto">
    <div
      class="
        flex
        items-end
        justify-center
        min-h-screen
        pt-4
        px-4
        pb-20
        text-center
        sm:block sm:p-0
      "
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
        class="
          inline-block
          align-bottom
          bg-white
          rounded-lg
          px-4
          pt-5
          pb-4
          text-left
          overflow-hidden
          shadow-xl
          transform
          transition-all
          sm:my-8 sm:align-middle sm:max-w-lg sm:w-full sm:p-6
        "
        role="dialog"
        aria-modal="true"
        aria-labelledby="modal-headline"
      >
        <div class="sm:flex sm:flex-col sm:items-start">
          <div class="flex items-start">
            <div
              v-if="style == 'SUCCESS'"
              class="
                mx-auto
                flex-shrink-0 flex
                items-center
                justify-center
                h-12
                w-12
                rounded-full
                bg-green-100
                sm:mx-0 sm:h-10 sm:w-10
              "
            >
              <!-- Heroicon name: outline/exclamation -->
              <svg
                class="h-6 w-6 text-success"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M5 13l4 4L19 7"
                />
              </svg>
            </div>
            <div
              v-else-if="style == 'WARN'"
              class="
                mx-auto
                flex-shrink-0 flex
                items-center
                justify-center
                h-12
                w-12
                rounded-full
                bg-yellow-100
                sm:mx-0 sm:h-10 sm:w-10
              "
            >
              <!-- Heroicon name: outline/exclamation -->
              <svg
                class="h-6 w-6 text-yellow-600"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                ></path>
              </svg>
            </div>
            <div
              v-else-if="style == 'CRITICAL'"
              class="
                mx-auto
                flex-shrink-0 flex
                items-center
                justify-center
                h-12
                w-12
                rounded-full
                bg-red-100
                sm:mx-0 sm:h-10 sm:w-10
              "
            >
              <!-- Heroicon name: outline/exclamation -->
              <svg
                class="h-6 w-6 text-red-600"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                />
              </svg>
            </div>
            <div
              v-else
              class="
                mx-auto
                flex-shrink-0 flex
                items-center
                justify-center
                h-12
                w-12
                rounded-full
                bg-blue-100
                sm:mx-0 sm:h-10 sm:w-10
              "
            >
              <!-- Heroicon name: outline/exclamation -->
              <svg
                class="h-6 w-6 text-blue-600"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                ></path>
              </svg>
            </div>
            <h3
              id="modal-headline"
              class="
                ml-4
                flex
                self-center
                text-lg
                leading-6
                font-medium
                text-gray-900
              "
            >
              {{ title }}
            </h3>
          </div>
          <div
            v-if="description"
            class="mt-3 text-center sm:mt-0 sm:ml-14 sm:text-left"
          >
            <div class="mt-2">
              <p class="text-gray-500 whitespace-pre-wrap">
                {{ description }}
              </p>
            </div>
          </div>
        </div>
        <div
          class="mt-5 flex"
          :class="inProgress ? 'justify-between' : 'justify-end'"
        >
          <BBSpin v-if="inProgress" />
          <div>
            <button
              type="button"
              class="btn-normal mt-3 px-4 py-2 sm:mt-0 sm:w-auto"
              :disabled="inProgress"
              @click.prevent="cancel"
            >
              {{ cancelText }}
            </button>
            <button
              type="button"
              class="
                sm:ml-3
                inline-flex
                justify-center
                w-full
                rounded-md
                border border-transparent
                shadow-sm
                px-4
                py-2
                bg-error
                text-base
                font-medium
                text-white
                hover:bg-error-hover
                focus:outline-none
                focus-visible:ring-2
                focus:ring-offset-2 focus:ring-red-500
                sm:w-auto sm:text-sm
              "
              :class="okButtonStyle"
              :disabled="inProgress"
              @click.prevent="$emit('ok', payload)"
            >
              {{ okText }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { onMounted, onUnmounted, PropType } from "vue";
import { BBAlertStyle } from "./types";

export default {
  name: "BBAlert",
  props: {
    style: {
      type: String as PropType<BBAlertStyle>,
      default: "INFO",
    },
    title: {
      required: true,
      type: String,
    },
    description: {
      type: String,
      default: "",
    },
    okText: {
      type: String,
      default: "OK",
    },
    cancelText: {
      type: String,
      default: "Cancel",
    },
    inProgress: {
      type: Boolean,
      default: false,
    },
    // Any payload pass through to "ok" and "cancel" events
    payload: {},
  },
  emits: ["ok", "cancel"],
  setup(props, { emit }) {
    const cancel = () => {
      emit("cancel", props.payload);
    };

    const escHandler = (e: KeyboardEvent) => {
      if (!props.inProgress && e.code == "Escape") {
        cancel();
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", escHandler);
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", escHandler);
    });

    const buttonStyleMap: Record<string, string> = {
      INFO: "btn-primary",
      SUCCESS: "btn-success",
      WARN: "btn-primary",
      CRITICAL: "btn-danger",
    };

    const okButtonStyle = buttonStyleMap[props.style];

    return {
      okButtonStyle,
      cancel,
    };
  },
};
</script>
