import axios from "axios";
import {
  UserId,
  Bookmark,
  BookmarkNew,
  BookmarkState,
  ResourceObject,
  unknown,
} from "../../types";

function convert(bookmark: ResourceObject): Bookmark {
  return {
    id: bookmark.id,
    ...(bookmark.attributes as Omit<Bookmark, "id">),
  };
}

const state: () => BookmarkState = () => ({
  bookmarkListByUser: new Map(),
});

const getters = {
  bookmarkListByUser: (state: BookmarkState) => (
    userId: UserId
  ): Bookmark[] => {
    return state.bookmarkListByUser.get(userId) || [];
  },
  bookmarkByUserAndLink: (state: BookmarkState, getters: any) => (
    userId: UserId,
    link: string
  ): Bookmark => {
    const list = getters["bookmarkListByUser"](userId);
    return (
      list.find((item: Bookmark) => item.link == link) ||
      (unknown("BOOKMARK") as Bookmark)
    );
  },
};

const actions = {
  async fetchBookmarkListByUser({ commit }: any, userId: UserId) {
    const bookmarkList = (
      await axios.get(`/api/bookmark?user=${userId}`)
    ).data.data.map((bookmark: ResourceObject) => {
      return convert(bookmark);
    });
    commit("setBookmarkListByUserId", { userId, bookmarkList });
    return bookmarkList;
  },

  async createBookmark({ commit }: any, newBookmark: BookmarkNew) {
    const createdBookmark = convert(
      (
        await axios.post(`/api/bookmark`, {
          data: {
            type: "bookmark",
            attributes: newBookmark,
          },
        })
      ).data.data
    );

    commit("appendBookmark", createdBookmark);

    return createdBookmark;
  },

  async patchBookmark({ commit }: any, bookmark: Bookmark) {
    const { id, ...attrs } = bookmark;
    const updatedBookmark = convert(
      (
        await axios.patch(`/api/bookmark/${bookmark.id}`, {
          data: {
            type: "bookmark",
            attributes: attrs,
          },
        })
      ).data.data
    );

    commit("replaceBookmark", updatedBookmark);

    return updatedBookmark;
  },

  async deleteBookmark({ commit }: any, bookmark: Bookmark) {
    await axios.delete(`/api/bookmark/${bookmark.id}`);

    commit("deleteBookmark", bookmark);
  },
};

const mutations = {
  setBookmarkListByUserId(
    state: BookmarkState,
    {
      userId,
      bookmarkList,
    }: {
      userId: UserId;
      bookmarkList: Bookmark[];
    }
  ) {
    state.bookmarkListByUser.set(userId, bookmarkList);
  },

  appendBookmark(state: BookmarkState, bookmark: Bookmark) {
    const list = state.bookmarkListByUser.get(bookmark.creatorId);
    if (list) {
      list.push(bookmark);
    } else {
      state.bookmarkListByUser.set(bookmark.creatorId, [bookmark]);
    }
  },

  replaceBookmark(state: BookmarkState, updatedBookmark: Bookmark) {
    const list = state.bookmarkListByUser.get(updatedBookmark.creatorId);
    if (list) {
      const i = list.findIndex(
        (item: Bookmark) => item.id == updatedBookmark.id
      );
      if (i != -1) {
        list[i] = updatedBookmark;
      }
    }
  },

  deleteBookmark(state: BookmarkState, bookmark: Bookmark) {
    const list = state.bookmarkListByUser.get(bookmark.creatorId);
    if (list) {
      const i = list.findIndex((item: Bookmark) => item.id == bookmark.id);
      if (i != -1) {
        list.splice(i, 1);
      }
    }
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
