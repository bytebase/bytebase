import { defineStore } from "pinia";
import axios from "axios";
import {
  PrincipalId,
  Bookmark,
  BookmarkCreate,
  BookmarkState,
  ResourceObject,
  unknown,
} from "@/types";

function convert(bookmark: ResourceObject): Bookmark {
  return {
    ...(bookmark.attributes as Omit<Bookmark, "id">),
    id: parseInt(bookmark.id),
  };
}

export const useBookmarkStore = defineStore("bookmark", {
  state: (): BookmarkState => ({
    bookmarkList: new Map(),
  }),

  actions: {
    async fetchBookmarkListByUser(userId: PrincipalId) {
      // API only returns bookmark for the requesting user.
      // User info is retrieved from the context.
      const data = (await axios.get(`/api/bookmark/user/${userId}`)).data;
      const bookmarkList = data.data.map((bookmark: ResourceObject) => {
        return convert(bookmark);
      });
      this.setBookmarkListByPrincipalId({ userId, bookmarkList });
      return bookmarkList;
    },

    async createBookmark(newBookmark: BookmarkCreate) {
      const data = (
        await axios.post(`/api/bookmark`, {
          data: {
            type: "bookmark",
            attributes: newBookmark,
          },
        })
      ).data;
      const createdBookmark = convert(data.data);
      this.appendBookmark(createdBookmark);

      return createdBookmark;
    },

    setBookmarkListByPrincipalId({
      userId,
      bookmarkList,
    }: {
      userId: PrincipalId;
      bookmarkList: Bookmark[];
    }) {
      this.bookmarkList.set(userId, bookmarkList);
    },

    appendBookmark(bookmark: Bookmark) {
      const list = this.bookmarkList.get(bookmark.creatorID);
      if (list) {
        list.push(bookmark);
      } else {
        this.bookmarkList.set(bookmark.creatorID, [bookmark]);
      }
    },

    async deleteBookmark(bookmark: Bookmark) {
      await axios.delete(`/api/bookmark/${bookmark.id}`);

      const list = this.bookmarkList.get(bookmark.creatorID);
      if (list) {
        const i = list.findIndex((item: Bookmark) => item.id == bookmark.id);
        if (i != -1) {
          list.splice(i, 1);
        }
      }
    },

    bookmarkListByUser(userId: PrincipalId): Bookmark[] {
      return this.bookmarkList.get(userId) || [];
    },
    bookmarkByUserAndLink(userId: PrincipalId, link: string): Bookmark {
      const list = this.bookmarkListByUser(userId);
      return (
        list.find((item: Bookmark) => item.link == link) ||
        (unknown("BOOKMARK") as Bookmark)
      );
    },
  },
});
