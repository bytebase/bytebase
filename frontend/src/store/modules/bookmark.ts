import axios from "axios";
import { UserId, Bookmark, BookmarkState, ResourceObject } from "../../types";

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
  bookmarkListByUser: (state: BookmarkState) => (userId: UserId) => {
    return state.bookmarkListByUser.get(userId);
  },
};

const actions = {
  async fetchBookmarkListForUser({ commit }: any, userId: UserId) {
    const bookmarkList = (
      await axios.get(`/api/bookmark?userid=${userId}`)
    ).data.data.map((bookmark: ResourceObject) => {
      return convert(bookmark);
    });
    commit("setBookmarkListForUser", { userId, bookmarkList });
    return bookmarkList;
  },
};

const mutations = {
  setBookmarkListForUser(
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
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
