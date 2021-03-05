<template>
  <BBOutline
    :title="'Bookmarks'"
    :itemList="bookmarkList"
    :allowDelete="true"
    @click-item="clickItem"
    @delete-item="deleteItem"
  />
</template>

<script lang="ts">
import { computed } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { Bookmark, User } from "../types";

export default {
  name: "BookmarkListSidePanel",
  props: {},
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const currentUser: User = computed(() =>
      store.getters["auth/currentUser"]()
    ).value;

    const bookmarkList = computed(() =>
      store.getters["bookmark/bookmarkListByUser"](currentUser.id)
    );

    const clickItem = (bookmark: Bookmark) => {
      router.push(bookmark.link);
    };

    const deleteItem = (bookmark: Bookmark) => {
      store.dispatch("bookmark/deleteBookmark", bookmark).catch((error) => {
        console.log(error);
      });
    };

    return {
      bookmarkList,
      clickItem,
      deleteItem,
    };
  },
};
</script>
