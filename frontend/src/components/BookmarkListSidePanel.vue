<template>
  <!-- Secondary navigation -->
  <h3
    class="px-3 text-xs leading-4 font-semibold text-gray-500 uppercase tracking-wider"
    id="bookmark-headline"
  >
    Bookmarks
  </h3>
  <div
    class="mt-2 space-y-1"
    role="group"
    aria-labelledby="bookmark-headline"
    v-for="(item, index) in bookmarkList"
    :key="item.id"
    @mouseenter="state.hoverIndex = index"
    @mouseleave="state.hoverIndex = -1"
  >
    <router-link
      :to="item.link"
      class="sidebar-link group flex justify-between items-center px-3 py-1 text-sm"
    >
      <span class="truncate">{{ item.name }}</span>
      <button
        v-if="index == state.hoverIndex"
        class="focus:outline-none"
        @click.prevent="deleteItem(item)"
      >
        <svg
          class="w-4 h-4 hover:text-control-hover"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M6 18L18 6M6 6l12 12"
          ></path>
        </svg>
      </button>
    </router-link>
  </div>
</template>

<script lang="ts">
interface LocalState {
  hoverIndex: number;
}

import { watchEffect, computed, reactive } from "vue";
import { useStore } from "vuex";
import { Bookmark, User } from "../types";

export default {
  name: "BookmarkListSidePanel",
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const state = reactive({
      hoverIndex: -1,
    });

    const currentUser: User = computed(() =>
      store.getters["auth/currentUser"]()
    ).value;

    const bookmarkList = computed(() =>
      store.getters["bookmark/bookmarkListByUser"](currentUser.id)
    );

    const deleteItem = (bookmark: Bookmark) => {
      store.dispatch("bookmark/deleteBookmark", bookmark).catch((error) => {
        console.log(error);
      });
    };

    return {
      state,
      bookmarkList,
      deleteItem,
    };
  },
};
</script>
