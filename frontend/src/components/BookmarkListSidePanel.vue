<template>
  <BBOutline
    :id="'bookmark'"
    :title="$t('common.bookmarks')"
    :item-list="
      bookmarkList.map((item) => {
        return { name: item.name, link: item.link };
      })
    "
    :allow-delete="true"
    :allow-collapse="true"
    @delete-index="deleteIndex"
  />
</template>

<script lang="ts">
import { computed, defineComponent, watchEffect } from "vue";
import { useStore } from "vuex";
import { UNKNOWN_ID } from "../types";
import { Action, defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useCurrentUser } from "@/store";

export default defineComponent({
  name: "BookmarkListSidePanel",
  setup() {
    const { t } = useI18n();
    const store = useStore();
    const router = useRouter();

    const currentUser = useCurrentUser();

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

    const kbarActions = computed((): Action[] => {
      const actions = bookmarkList.value.map((item: any) =>
        defineAction({
          // here `id` looks like "bb.bookmark.12345"
          id: `bb.bookmark.${item.id}`,
          section: t("common.bookmarks"),
          name: item.name,
          keywords: "bookmark",
          perform: () => {
            router.push({ path: item.link });
          },
        })
      );
      return actions;
    });
    useRegisterActions(kbarActions);

    return {
      bookmarkList,
      deleteIndex,
    };
  },
});
</script>
