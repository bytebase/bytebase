<template>
  <BBOutline
    :id="'bookmark'"
    :title="$t('common.bookmarks')"
    :item-list="
      bookmarkList.map((item) => {
        return {
          id: item.name,
          name: item.title,
          link: item.link,
        };
      })
    "
    :allow-delete="true"
    :allow-collapse="true"
    @delete-index="deleteIndex"
  />
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import { UNKNOWN_ID } from "../types";
import { Action, defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useBookmarkV1Store, useCurrentUser } from "@/store";
import { Bookmark } from "@/types/proto/v1/bookmark_service";

const { t } = useI18n();
const router = useRouter();
const bookmarkV1Store = useBookmarkV1Store();

const currentUser = useCurrentUser();

const prepareBookmarkList = async () => {
  // It will also be called when user logout
  if (currentUser.value.id != UNKNOWN_ID) {
    await bookmarkV1Store.fetchBookmarkList();
  }
};

watchEffect(prepareBookmarkList);

const bookmarkList = computed(() => bookmarkV1Store.getBookmarkList());

const deleteIndex = (index: number) => {
  bookmarkV1Store.deleteBookmark(bookmarkList.value[index].name);
};

const kbarActions = computed((): Action[] => {
  const actions = bookmarkList.value.map((item: Bookmark) =>
    defineAction({
      // here `id` looks like "bb.bookmark.12345"
      id: `bb.bookmark.${item.name}`,
      section: t("common.bookmarks"),
      name: item.title,
      keywords: "bookmark",
      perform: () => {
        router.push({ path: item.link });
      },
    })
  );
  return actions;
});
useRegisterActions(kbarActions);
</script>
