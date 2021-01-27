<template>
  <div>
    <div class="px-4 sm:px-6">
      <h2 class="text-gray-500 text-xs font-medium uppercase tracking-wide">
        Projects
      </h2>
    </div>
    <ul class="mt-3 border-t border-block-border divide-y divide-gray-100">
      <li>
        <a
          href="#"
          class="flex items-center justify-between px-4 py-4 hover:bg-gray-300 sm:px-6"
        >
          <div class="flex items-center truncate space-x-3">
            <div
              class="w-2.5 h-2.5 flex-shrink-0 rounded-full bg-pink-600"
            ></div>
            <p class="font-medium truncate text-sm leading-6">
              GraphQL API
              <span class="truncate font-normal text-gray-500"
                >in Engineering</span
              >
            </p>
          </div>
          <svg
            class="ml-4 h-5 w-5 text-gray-400 group-hover:text-gray-500 group-focus:text-gray-600"
            viewBox="0 0 20 20"
            fill="currentColor"
          >
            <path
              fill-rule="evenodd"
              d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z"
              clip-rule="evenodd"
            />
          </svg>
        </a>
      </li>

      <!-- More project rows... -->
    </ul>
  </div>
</template>
