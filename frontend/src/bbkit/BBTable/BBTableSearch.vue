<template>
  <div class="">
    <label for="search" class="sr-only">Search</label>
    <div class="relative rounded-md shadow-sm">
      <div
        class="
          absolute
          inset-y-0
          left-0
          pl-3
          flex
          items-center
          pointer-events-none
        "
        aria-hidden="true"
      >
        <!-- Heroicon name: solid/search -->
        <svg
          class="mr-3 h-4 w-4 text-control"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
        >
          <path
            fill-rule="evenodd"
            d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z"
            clip-rule="evenodd"
          />
        </svg>
      </div>
      <input
        id="search"
        type="text"
        autocomplete="off"
        name="search"
        class="
          focus:ring-main focus:border-main
          block
          w-full
          pl-9
          sm:text-sm
          border-control-border
          rounded-md
        "
        :placeholder="placeholder"
        @input="$emit('change-text', $event.target.value)"
      />
    </div>
  </div>
</template>

<script lang="ts">
export default {
  name: "BBTableSearch",
  props: {
    placeholder: {
      default: "Search",
      type: String,
    },
  },
  emits: ["change-text"],
};
</script>
