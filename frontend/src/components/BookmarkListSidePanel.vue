<template>
  <BBOutline
    :id="'bookmark'"
    :title="'Bookmarks'"
    :itemList="bookmarkList.map((item) => item.name)"
    :allowDelete="true"
    :allowCollapse="true"
    @click-index="clickIndex"
    @delete-index="deleteIndex"
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

    const clickIndex = (index: number) => {
      router.push(bookmarkList.value[index].link);
    };

    const deleteIndex = (index: number) => {
      store
        .dispatch("bookmark/deleteBookmark", bookmarkList.value[index])
        .catch((error) => {
          console.log(error);
        });
    };

    return {
      bookmarkList,
      clickIndex,
      deleteIndex,
    };
  },
};
</script>
