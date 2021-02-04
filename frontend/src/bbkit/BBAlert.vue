<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div class="fixed z-10 inset-0 overflow-y-auto">
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
        <div
          class="absolute inset-0 bg-gray-500 opacity-75"
          @click.self="clickBackground"
        ></div>
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
        <div>
          <div
            v-if="style == 'SUCCESS'"
            class="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-green-100"
          >
            <svg
              class="h-6 w-6 text-green-600"
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
            class="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-yellow-100"
          >
            <svg
              class="w-6 h-6 text-yellow-600"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
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
            class="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-red-100"
          >
            <svg
              class="w-6 h-6 text-red-600"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              ></path>
            </svg>
          </div>
          <div
            v-else
            class="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-blue-100"
          >
            <svg
              class="w-6 h-6 text-blue-600"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              ></path>
            </svg>
          </div>
          <div class="mt-3 text-center sm:mt-5">
            <h3
              class="text-lg leading-6 font-medium text-gray-900"
              id="modal-headline"
            >
              {{ title }}
            </h3>
            <div class="mt-2">
              <p class="text-sm text-gray-500">
                {{ description }}
              </p>
            </div>
          </div>
        </div>
        <div
          class="mt-5 sm:mt-6 sm:grid sm:grid-cols-2 sm:gap-3 sm:grid-flow-row-dense"
        >
          <button
            type="button"
            class="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 text-base font-medium text-white focus:outline-none focus-visible:ring-2 focus:ring-accent sm:col-start-2 sm:text-sm"
            v-bind:class="okButtonStyle"
            @click.prevent="$emit('ok')"
          >
            {{ okText }}
          </button>
          <button
            type="button"
            class="btn-normal mt-3 w-full inline-flex justify-center shadow-sm px-4 py-2 sm:mt-0 sm:col-start-1 sm:text-sm"
            @click.prevent="cancel"
          >
            {{ cancelText }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { onMounted, PropType } from "vue";

export default {
  name: "BBAlert",
  emits: ["ok", "cancel"],
  props: {
    style: {
      default: Object as PropType<"INFO" | "SUCCESS" | "WARN" | "CRITICAL">,
      type: String,
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
    backgroundClose: {
      type: Boolean,
      default: true,
    },
  },
  setup(props, { emit }) {
    const cancel = () => {
      emit("cancel");
    };

    const clickBackground = () => {
      if (props.backgroundClose) {
        cancel();
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", (e) => {
        if (e.keyCode == 27) {
          cancel();
        }
      });
    });

    const buttonStyleMap: Record<string, string> = {
      INFO: "bg-blue-600 hover:bg-blue-700",
      SUCCESS: "bg-blue-600 hover:bg-blue-700",
      WARN: "bg-red-600 hover:bg-red-700",
      CRITICAL: "bg-red-600 hover:bg-red-700",
    };

    const okButtonStyle = buttonStyleMap[props.style];

    return {
      okButtonStyle,
      cancel,
      clickBackground,
    };
  },
};
</script>
