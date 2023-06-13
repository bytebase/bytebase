import { defineStore } from "pinia";
import { reactive } from "vue";

import { bookmarkServiceClient } from "@/grpcweb";
import { Bookmark } from "@/types/proto/v1/bookmark_service";

export const useBookmarkV1Store = defineStore("bookmark_v1", () => {
  let bookmarkList = reactive<Bookmark[]>([]);

  const fetchBookmarkList = async () => {
    const resp = await bookmarkServiceClient.listBookmarks({});
    bookmarkList = resp.bookmarks;
    return bookmarkList;
  };

  const createBookmark = async ({
    title,
    link,
  }: {
    title: string;
    link: string;
  }) => {
    const bookmark = await bookmarkServiceClient.createBookmark({
      bookmark: {
        title,
        link,
      },
    });
    bookmarkList.push(bookmark);
    return bookmark;
  };

  const deleteBookmark = async (name: string) => {
    await bookmarkServiceClient.deleteBookmark({
      name,
    });
    const index = bookmarkList.findIndex((b) => b.name === name);
    bookmarkList = [
      ...bookmarkList.slice(0, index),
      ...bookmarkList.slice(index + 1),
    ];
  };

  const findBookmarkByLink = (link: string) => {
    return bookmarkList.find((b) => b.link === link);
  };

  return {
    bookmarkList,
    fetchBookmarkList,
    createBookmark,
    deleteBookmark,
    findBookmarkByLink,
  };
});
