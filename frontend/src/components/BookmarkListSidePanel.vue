<template>
  <BBOutline
    :id="'bookmark'"
    :title="'Bookmarks'"
    :itemList="
      bookmarkList.map((item) => {
        return { name: item.name, link: item.link };
      })
    "
    :allowDelete="true"
    :allowCollapse="true"
    @delete-index="deleteIndex"
  />
</template>

<script lang="ts">
import { computed, watchEffect } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { UNKNOWN_ID } from "../types";

export default {
  name: "BookmarkListSidePanel",
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareBookmarkList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        store.dispatch(
          "bookmark/fetchBookmarkListByUser",
          currentUser.value.id
        );
      }
    };

    watchEffect(prepareBookmarkList);

    const bookmarkList = computed(() =>
      store.getters["bookmark/bookmarkListByUser"](currentUser.value.id)
    );

    const deleteIndex = (index: number) => {
      store.dispatch("bookmark/deleteBookmark", bookmarkList.value[index]);
    };

    return {
      bookmarkList,
      deleteIndex,
    };
  },
};
</script>
