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
    v-for="item in bookmarkList"
    :key="item.id"
  >
    <router-link
      :to="item.link"
      class="sidebar-link group flex items-center px-3 py-1 text-sm"
    >
      {{ item.name }}
    </router-link>
  </div>
</template>

<script lang="ts">
import { watchEffect, computed } from "vue";
import { useStore } from "vuex";
import { User } from "../types";

export default {
  name: "BookmarkListSidePanel",
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const currentUser: User = computed(() =>
      store.getters["auth/currentUser"]()
    ).value;

    const prepareBookmarkList = () => {
      store
        .dispatch("bookmark/fetchBookmarkListForUser", currentUser.id)
        .catch((error) => {
          console.log(error);
        });
    };

    const bookmarkList = computed(() =>
      store.getters["bookmark/bookmarkListByUser"](currentUser.id)
    );

    watchEffect(prepareBookmarkList);

    return {
      bookmarkList,
    };
  },
};
</script>
