<template>
  <!-- Navbar -->
  <div class="max-w-full mx-auto px-2">
    <div class="relative flex items-center justify-between h-16">
      <!-- Sidebar toggle, controls the 'sidebarOpen' sidebar state. -->
      <button
        @click.prevent="$emit('open-sidebar')"
        class="px-2 text-gray-500 focus:outline-none focus:bg-gray-100 focus:text-gray-600 lg:hidden"
        aria-label="Open sidebar"
      >
        <svg
          class="h-6 w-6"
          fill="none"
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path d="M4 6h16M4 12h8m-8 6h16" />
        </svg>
      </button>
      <div class="flex-1 flex">
        <form class="w-full flex md:ml-0" action="#" method="GET">
          <label for="search_field" class="sr-only">Search</label>
          <div class="relative w-full text-gray-400 focus-within:text-gray-600">
            <div
              class="pointer-events-none absolute inset-y-0 left-0 flex items-center"
            >
              <!-- Heroicon name: search -->
              <svg
                class="flex-shrink-0 h-5 w-5"
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path
                  fill-rule="evenodd"
                  d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z"
                  clip-rule="evenodd"
                />
              </svg>
            </div>
            <input
              id="search_field_mobile"
              class="h-full w-full rounded-md py-2 pl-8 pr-3 border-transparent text-base text-gray-900 placeholder-gray-500 focus:outline-none focus:placeholder-gray-400 sm:hidden"
              placeholder="Search"
              type="search"
            />
            <input
              id="search_field"
              class="bg-normal hidden h-full w-full py-2 pl-8 pr-3 border-transparent text-sm text-gray-900 placeholder-gray-500 focus:outline-none focus:placeholder-gray-400 sm:block"
              placeholder="Search jobs, applicants, and more"
              type="search"
            />
          </div>
        </form>
      </div>
      <!-- Links section -->
      <div class="hidden lg:block lg:w-80">
        <div class="flex items-center justify-end">
          <div class="flex">
            <a href="#" class="link px-3 py-2">Documentation</a>
            <a href="#" class="link px-3 py-2">Support</a>
          </div>
        </div>
      </div>

      <!--
      Mobile menu, toggle classes based on menu state.

      Menu open: "block", Menu closed: "hidden"
    -->
      <div class="hidden lg:hidden">
        <div class="px-2 pt-2 pb-3">
          <a
            href="#"
            class="block px-3 py-2 rounded-md text-base font-medium text-white bg-indigo-800 focus:outline-none focus:text-indigo-100 focus:bg-indigo-600"
            >Dashboard</a
          >
          <a
            href="#"
            class="mt-1 block px-3 py-2 rounded-md text-base font-medium text-indigo-200 hover:text-indigo-100 hover:bg-indigo-600 focus:outline-none focus:text-white focus:bg-indigo-600"
            >Support</a
          >
        </div>
        <div class="pt-4 pb-3 border-t border-indigo-800">
          <div class="px-2">
            <a
              href="#"
              class="block px-3 py-2 rounded-md text-base font-medium text-indigo-200 hover:text-indigo-100 hover:bg-indigo-600 focus:outline-none focus:text-white focus:bg-indigo-600"
              >Your Profile</a
            >
            <a
              href="#"
              class="mt-1 block px-3 py-2 rounded-md text-base font-medium text-indigo-200 hover:text-indigo-100 hover:bg-indigo-600 focus:outline-none focus:text-white focus:bg-indigo-600"
              >Settings</a
            >
            <a
              href="#"
              class="mt-1 block px-3 py-2 rounded-md text-base font-medium text-indigo-200 hover:text-indigo-100 hover:bg-indigo-600 focus:outline-none focus:text-white focus:bg-indigo-600"
              >Sign out</a
            >
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
export default {
  name: "TheHeader",
  components: {},
};
</script>
