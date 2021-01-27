<template>
  <a
    href="#"
    class="bg-normal flex-1 flex items-center justify-between border border-block-border rounded-md hover:shadow-md truncate"
  >
    <div class="flex-1 px-4 py-2 text-sm leading-5 truncate">
      <div class="text-gray-900 font-medium hover:text-gray-600">
        GraphQL API
      </div>
    </div>
    <div class="flex-shrink-0 pr-2">
      <button
        @click.prevent="$refs.menu.toggle($event)"
        @contextmenu.capture.prevent="$refs.menu.toggle($event)"
        id="pinned-project-options-menu-0"
        aria-has-popup="true"
        class="w-8 h-8 inline-flex items-center justify-center text-gray-400 rounded-full bg-transparent hover:text-gray-500 focus:outline-none focus:text-gray-500 focus:bg-gray-100"
      >
        <svg class="w-5 h-5" viewBox="0 0 20 20" fill="currentColor">
          <path
            d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z"
          />
        </svg>
      </button>
      <BBContextMenu
        ref="menu"
        class="z-10 ml-2 origin-top-right absolute top-13 w-48 rounded-md shadow-lg"
      >
        <div class="py-1">
          <a
            href="#"
            class="block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
            role="menuitem"
            >View</a
          >
        </div>
        <div class="border-t border-gray-100"></div>
        <div class="py-1">
          <a
            href="#"
            class="block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
            role="menuitem"
            >Removed from pinned</a
          >
          <a
            href="#"
            class="block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
            role="menuitem"
            >Share</a
          >
        </div>
      </BBContextMenu>
    </div>
  </a>
</template>

<script lang="ts">
export default {
  name: "PinnedProjectCard",
  components: {},
};
</script>
