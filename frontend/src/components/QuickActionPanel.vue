<template>
  <div class="overflow-hidden grid grid-cols-2 gap-2 sm:inline-flex">
    <div class="flex flex-col items-center w-32">
      <button class="btn-icon">
        <span
          class="inline-flex items-center p-3 bg-blue-500 hover:bg-blue-600 rounded-full"
        >
          <svg
            class="w-6 h-6 text-white"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"
            ></path>
          </svg>
        </span>
      </button>
      <h3 class="mt-1 text-base font-normal text-accent tracking-tight">
        Add Instance
      </h3>
    </div>

    <div class="flex flex-col items-center w-32">
      <button class="btn-icon">
        <span
          class="inline-flex items-center justify-center p-3 bg-blue-500 hover:bg-blue-600 rounded-full"
        >
          <svg
            class="w-6 h-6 text-white"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"
            ></path>
          </svg>
        </span>
      </button>
      <h3
        class="mt-1 text-center text-base font-normal text-accent tracking-tight"
      >
        Invite User
      </h3>
    </div>

    <div class="flex flex-col items-center w-32">
      <button class="btn-icon">
        <span
          class="inline-flex items-center justify-center p-3 bg-blue-500 hover:bg-blue-600 rounded-full"
        >
          <svg
            class="w-6 h-6 text-white"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M10 21h7a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v11m0 5l4.879-4.879m0 0a3 3 0 104.243-4.242 3 3 0 00-4.243 4.242z"
            ></path>
          </svg>
        </span>
      </button>
      <h3
        class="mt-1 text-center text-base font-normal text-accent tracking-tight"
      >
        Query Data
      </h3>
    </div>

    <div class="flex flex-col items-center w-32">
      <button class="btn-icon">
        <span
          class="inline-flex items-center justify-center p-3 bg-blue-500 hover:bg-blue-600 rounded-full"
        >
          <svg
            class="w-6 h-6 text-white"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M3 10h18M3 14h18m-9-4v8m-7 0h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"
            ></path>
          </svg>
        </span>
      </button>
      <h3 class="mt-1 text-center text-base font-normal text-accent">
        Change Schema
      </h3>
    </div>
  </div>
</template>

<script lang="ts">
export default {
  name: "QuickActionPanel",
  setup() {},
};
</script>
