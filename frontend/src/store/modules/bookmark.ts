import axios from "axios";
import {
  PrincipalID,
  Bookmark,
  BookmarkCreate,
  BookmarkState,
  ResourceObject,
  unknown,
} from "../../types";

function convert(bookmark: ResourceObject, rootGetters: any): Bookmark {
  return {
    ...(bookmark.attributes as Omit<Bookmark, "id">),
    id: parseInt(bookmark.id),
  };
}

const state: () => BookmarkState = () => ({
  bookmarkListByUser: new Map(),
});

const getters = {
  bookmarkListByUser:
    (state: BookmarkState) =>
    (userID: PrincipalID): Bookmark[] => {
      return state.bookmarkListByUser.get(userID) || [];
    },
  bookmarkByUserAndLink:
    (state: BookmarkState, getters: any) =>
    (userID: PrincipalID, link: string): Bookmark => {
      const list = getters["bookmarkListByUser"](userID);
      return (
        list.find((item: Bookmark) => item.link == link) ||
        (unknown("BOOKMARK") as Bookmark)
      );
    },
};

const actions = {
  async fetchBookmarkListByUser(
    { commit, rootGetters }: any,
    userID: PrincipalID
  ) {
    // API only returns bookmark for the requesting user.
    // User info is retrieved from the context.
    const bookmarkList = (await axios.get(`/api/bookmark`)).data.data.map(
      (bookmark: ResourceObject) => {
        return convert(bookmark, rootGetters);
      }
    );
    commit("setBookmarkListByPrincipalID", { userID, bookmarkList });
    return bookmarkList;
  },

  async createBookmark(
    { commit, rootGetters }: any,
    newBookmark: BookmarkCreate
  ) {
    const createdBookmark = convert(
      (
        await axios.post(`/api/bookmark`, {
          data: {
            type: "bookmark",
            attributes: newBookmark,
          },
        })
      ).data.data,
      rootGetters
    );

    commit("appendBookmark", createdBookmark);

    return createdBookmark;
  },

  async deleteBookmark({ commit }: any, bookmark: Bookmark) {
    await axios.delete(`/api/bookmark/${bookmark.id}`);

    commit("deleteBookmark", bookmark);
  },
};

const mutations = {
  setBookmarkListByPrincipalID(
    state: BookmarkState,
    {
      userID,
      bookmarkList,
    }: {
      userID: PrincipalID;
      bookmarkList: Bookmark[];
    }
  ) {
    state.bookmarkListByUser.set(userID, bookmarkList);
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
