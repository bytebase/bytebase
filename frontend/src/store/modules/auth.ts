import axios from "axios";
import { User, AuthState, LoginInfo, SignupInfo } from "../../types";

const state: () => AuthState = () => ({
  currentUser: null,
});

const getters = {
  currentUser: (state: AuthState) => () => {
    if (state.currentUser) {
      return state.currentUser;
    }
    const user = localStorage.getItem("bb.auth.user");
    if (user) {
      return JSON.parse(user);
    }
    return null;
  },
};

const actions = {
  async login({ commit }: any, loginInfo: LoginInfo) {
    const loggedInUser = (
      await axios.post("/api/auth/login", {
        data: loginInfo,
      })
    ).data.data;

    localStorage.setItem("bb.auth.user", JSON.stringify(loggedInUser));
    commit("setCurrentUser", loggedInUser);
    return loggedInUser;
  },

  async signup({ commit }: any, signupInfo: SignupInfo) {
    const newUser = (
      await axios.post("/api/auth/signup", {
        data: signupInfo,
      })
    ).data.data;

    localStorage.setItem("bb.auth.user", JSON.stringify(newUser));
    commit("setCurrentUser", newUser);
    return newUser;
  },

  async fetchCurrentUser({ commit }: any) {
    const currentUser = (await axios.get("/api/user/1")).data.data;
    localStorage.setItem("bb.auth.user", JSON.stringify(currentUser));
    commit("setCurrentUser", currentUser);
    return currentUser;
  },

  async logout({ commit }: any) {
    localStorage.removeItem("bb.auth.user");
    commit("setCurrentUser", null);
    return null;
  },
};

const mutations = {
  setCurrentUser(state: AuthState, user: User) {
    state.currentUser = user;
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
