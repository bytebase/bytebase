import axios from "axios";
import {
  PrincipalId,
  Bookmark,
  BookmarkCreate,
  BookmarkState,
  ResourceObject,
  unknown,
} from "../../types";
import { getPrincipalFromIncludedList } from "../pinia";

function convert(
  bookmark: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Bookmark {
  return {
    ...(bookmark.attributes as Omit<Bookmark, "id" | "creator" | "updater">),
    id: parseInt(bookmark.id),
    creator: getPrincipalFromIncludedList(
      bookmark.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      bookmark.relationships!.updater.data,
      includedList
    ),
  };
}

const state: () => BookmarkState = () => ({
  bookmarkListByUser: new Map(),
});

const getters = {
  bookmarkListByUser:
    (state: BookmarkState) =>
    (userId: PrincipalId): Bookmark[] => {
      return state.bookmarkListByUser.get(userId) || [];
    },
  bookmarkByUserAndLink:
    (state: BookmarkState, getters: any) =>
    (userId: PrincipalId, link: string): Bookmark => {
      const list = getters["bookmarkListByUser"](userId);
      return (
        list.find((item: Bookmark) => item.link == link) ||
        (unknown("BOOKMARK") as Bookmark)
      );
    },
};

const actions = {
  async fetchBookmarkListByUser(
    { commit, rootGetters }: any,
    userId: PrincipalId
  ) {
    // API only returns bookmark for the requesting user.
    // User info is retrieved from the context.
    const data = (await axios.get(`/api/bookmark/user/${userId}`)).data;
    const bookmarkList = data.data.map((bookmark: ResourceObject) => {
      return convert(bookmark, data.included, rootGetters);
    });
    commit("setBookmarkListByPrincipalId", { userId, bookmarkList });
    return bookmarkList;
  },

  async createBookmark(
    { commit, rootGetters }: any,
    newBookmark: BookmarkCreate
  ) {
    const data = (
      await axios.post(`/api/bookmark`, {
        data: {
          type: "bookmark",
          attributes: newBookmark,
        },
      })
    ).data;
    const createdBookmark = convert(data.data, data.included, rootGetters);

    commit("appendBookmark", createdBookmark);

    return createdBookmark;
  },

  async deleteBookmark({ commit }: any, bookmark: Bookmark) {
    await axios.delete(`/api/bookmark/${bookmark.id}`);

    commit("deleteBookmark", bookmark);
  },
};

const mutations = {
  setBookmarkListByPrincipalId(
    state: BookmarkState,
    {
      userId,
      bookmarkList,
    }: {
      userId: PrincipalId;
      bookmarkList: Bookmark[];
    }
  ) {
    state.bookmarkListByUser.set(userId, bookmarkList);
  },

  appendBookmark(state: BookmarkState, bookmark: Bookmark) {
    const list = state.bookmarkListByUser.get(bookmark.creator.id);
    if (list) {
      list.push(bookmark);
    } else {
      state.bookmarkListByUser.set(bookmark.creator.id, [bookmark]);
    }
  },

  deleteBookmark(state: BookmarkState, bookmark: Bookmark) {
    const list = state.bookmarkListByUser.get(bookmark.creator.id);
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
