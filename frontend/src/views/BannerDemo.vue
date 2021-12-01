<template>
  <div v-if="show" class="bg-accent">
    <div class="mx-auto py-3 px-3">
      <div class="flex items-center justify-between flex-wrap">
        <div class="w-0 flex-1 flex items-center">
          <p class="ml-3 font-medium text-white truncate">
            <span>
              {{
                isReadonly
                  ? "This is a demo version of Bytebase in read-only mode."
                  : "This is a demo version of Bytebase."
              }}
            </span>
          </p>
        </div>
        <div
          class="order-3 mt-2 flex-shrink-0 w-full sm:order-2 sm:mt-0 sm:w-auto"
        >
          <a
            href="https://github.com/bytebase/bytebase"
            target="_blank"
            class="
              flex
              items-center
              justify-center
              pl-4
              pr-2
              py-2
              border border-transparent
              rounded-md
              shadow-sm
              text-base
              font-medium
              text-accent
              bg-white
              hover:bg-indigo-50
            "
          >
            Deploy yours in 5 seconds
            <svg
              class="ml-1 w-6 h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M13 10V3L4 14h7v7l9-11h-7z"
              ></path>
            </svg>
          </a>
        </div>
        <div class="order-2 flex-shrink-0 sm:order-3 sm:ml-3">
          <button
            type="button"
            class="
              -mr-1
              flex
              p-2
              rounded-md
              hover:bg-accent-hover
              focus:outline-none focus:ring-2 focus:ring-white
              sm:-mr-2
            "
            @click.prevent="show = false"
          >
            <span class="sr-only">Dismiss</span>
            <!-- Heroicon name: outline/x -->
            <svg
              class="h-6 w-6 text-white"
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
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, ref } from "vue";
import { useStore } from "vuex";

export default {
  name: "BannerDemo",
  setup() {
    const store = useStore();
    const show = ref(true);

    const isReadonly = computed(() => store.getters["actuator/isReadonly"]());

    return { show, isReadonly };
  },
};
</script>
