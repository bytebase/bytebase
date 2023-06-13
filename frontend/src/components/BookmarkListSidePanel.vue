<template>
  <BBOutline
    :id="'bookmark'"
    :title="$t('common.bookmarks')"
    :item-list="
      bookmarkList.map((item) => {
        return { id: item.name, name: item.title, link: item.link };
      })
    "
    :allow-delete="true"
    :allow-collapse="true"
    @delete-index="deleteIndex"
  />
</template>

<script lang="ts">
import { computed, defineComponent, watchEffect } from "vue";
import { UNKNOWN_ID } from "../types";
import { Action, defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useBookmarkV1Store, useCurrentUser } from "@/store";

export default defineComponent({
  name: "BookmarkListSidePanel",
  setup() {
    const { t } = useI18n();
    const router = useRouter();
    const bookmarkV1Store = useBookmarkV1Store();

    const currentUser = useCurrentUser();

    const prepareBookmarkList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        bookmarkV1Store.fetchBookmarkList();
      }
    };

    watchEffect(prepareBookmarkList);

    const bookmarkList = computed(() => bookmarkV1Store.bookmarkList);

    const deleteIndex = (index: number) => {
      bookmarkV1Store.deleteBookmark(bookmarkList.value[index].name);
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
