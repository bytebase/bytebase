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
import { computed } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { User } from "../types";

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

    const deleteIndex = (index: number) => {
      store
        .dispatch("bookmark/deleteBookmark", bookmarkList.value[index])
        .catch((error) => {
          console.log(error);
        });
    };

    return {
      bookmarkList,
      deleteIndex,
    };
  },
};
</script>
