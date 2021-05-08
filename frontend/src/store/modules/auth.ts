import axios from "axios";
import isEqual from "lodash-es/isEqual";
import isEmpty from "lodash-es/isEmpty";
import {
  Principal,
  AuthState,
  LoginInfo,
  SignupInfo,
  ActivateInfo,
  ResourceObject,
  unknown,
} from "../../types";
import { getCookie } from "../../utils";

function convert(user: ResourceObject, rootGetters: any): Principal {
  return rootGetters["principal/principalById"](user.id);
}

const state: () => AuthState = () => ({
  currentUser: unknown("PRINCIPAL") as Principal,
});

const getters = {
  isLoggedIn: (state: AuthState) => (): boolean => {
    return !isEmpty(getCookie("user"));
  },

  currentUser: (state: AuthState) => (): Principal => {
    return state.currentUser;
  },
};

const actions = {
  async login({ commit, rootGetters }: any, loginInfo: LoginInfo) {
    const loggedInUser = convert(
      (
        await axios.post("/api/auth/login", {
          data: { type: "loginInfo", attributes: loginInfo },
        })
      ).data.data,
      rootGetters
    );

    localStorage.setItem("bb.auth.user", JSON.stringify(loggedInUser));
    commit("setCurrentUser", loggedInUser);
    return loggedInUser;
  },

  async signup({ commit, dispatch, rootGetters }: any, signupInfo: SignupInfo) {
    const newUser = (
      await axios.post("/api/auth/signup", {
        data: { type: "signupInfo", attributes: signupInfo },
      })
    ).data.data;

    // Refresh the corresponding principal
    await dispatch("principal/fetchPrincipalById", newUser.id, { root: true });

    // The conversion relies on the above task.
    const convertedUser = convert(newUser, rootGetters);

    localStorage.setItem("bb.auth.user", JSON.stringify(convertedUser));
    commit("setCurrentUser", convertedUser);

    return convertedUser;
  },

  async activate(
    { commit, dispatch, rootGetters }: any,
    activateInfo: ActivateInfo
  ) {
    const activatedUser = (
      await axios.post("/api/auth/activate", {
        data: { type: "activateInfo", attributes: activateInfo },
      })
    ).data.data;

    // Refresh the corresponding principal
    dispatch("principal/fetchPrincipalById", activatedUser.id, { root: true });

    // The conversion relies on the above task to get the lastest data
    const convertedUser = convert(activatedUser, rootGetters);

    localStorage.setItem("bb.auth.user", JSON.stringify(convertedUser));
    commit("setCurrentUser", convertedUser);

    return convertedUser;
  },

  async restoreUser({ commit }: any) {
    const jsonUser = localStorage.getItem("bb.auth.user");
    if (jsonUser) {
      const user: Principal = JSON.parse(jsonUser);
      commit("setCurrentUser", user);
      return user;
    }
    return unknown("PRINCIPAL") as Principal;
  },

  async refreshUser({ commit, state, rootGetters }: any) {
    const refreshedUser = rootGetters["principal/principalById"](
      state.currentUser.id
    );
    if (!isEqual(refreshedUser, state.currentUser)) {
      localStorage.setItem("bb.auth.user", JSON.stringify(refreshedUser));
      commit("setCurrentUser", refreshedUser);
    }
    return refreshedUser;
  },

  async logout({ commit }: any) {
    localStorage.removeItem("bb.auth.user");
    commit("setCurrentUser", unknown("PRINCIPAL") as Principal);
    return unknown("PRINCIPAL") as Principal;
  },
};

const mutations = {
  setCurrentUser(state: AuthState, user: Principal) {
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
