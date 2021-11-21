import axios from "axios";
import isEqual from "lodash-es/isEqual";
import {
  Principal,
  AuthState,
  LoginInfo,
  SignupInfo,
  ActivateInfo,
  ResourceObject,
  unknown,
  PrincipalID,
} from "../../types";
import { getIntCookie } from "../../utils";

function convert(user: ResourceObject, rootGetters: any): Principal {
  return rootGetters["principal/principalByID"](user.id);
}

const state: () => AuthState = () => ({
  currentUser: unknown("PRINCIPAL") as Principal,
});

const getters = {
  isLoggedIn: (state: AuthState) => (): boolean => {
    return getIntCookie("user") != undefined;
  },

  currentUser: (state: AuthState) => (): Principal => {
    return state.currentUser;
  },
};

const actions = {
  async login({ commit, dispatch, rootGetters }: any, loginInfo: LoginInfo) {
    const loggedInUser = (
      await axios.post("/api/auth/login", {
        data: { type: "loginInfo", attributes: loginInfo },
      })
    ).data.data;

    // Refresh the corresponding principal
    await dispatch("principal/fetchPrincipalByID", loggedInUser.id, {
      root: true,
    });

    // The conversion relies on the above refresh.
    const convertedUser = convert(loggedInUser, rootGetters);
    commit("setCurrentUser", convertedUser);
    return convertedUser;
  },

  async logout({ commit }: any) {
    await axios.post("/api/auth/logout");

    commit("setCurrentUser", unknown("PRINCIPAL") as Principal);
    return unknown("PRINCIPAL") as Principal;
  },

  async signup({ commit, dispatch, rootGetters }: any, signupInfo: SignupInfo) {
    const newUser = (
      await axios.post("/api/auth/signup", {
        data: { type: "signupInfo", attributes: signupInfo },
      })
    ).data.data;

    // Refresh the corresponding principal
    await dispatch("principal/fetchPrincipalByID", newUser.id, { root: true });

    // The conversion relies on the above refresh.
    const convertedUser = convert(newUser, rootGetters);
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
    dispatch("principal/fetchPrincipalByID", activatedUser.id, { root: true });

    // The conversion relies on the above task to get the lastest data
    const convertedUser = convert(activatedUser, rootGetters);
    commit("setCurrentUser", convertedUser);
    return convertedUser;
  },

  async restoreUser({ commit, dispatch }: any) {
    const userID = getIntCookie("user");
    if (userID) {
      const loggedInUser = await dispatch(
        "principal/fetchPrincipalByID",
        userID,
        {
          root: true,
        }
      );

      commit("setCurrentUser", loggedInUser);
      return loggedInUser;
    }
    return unknown("PRINCIPAL") as Principal;
  },

  async refreshUserIfNeeded(
    { commit, state, rootGetters }: any,
    principalID: PrincipalID
  ) {
    if (principalID == state.currentUser.id) {
      const refreshedUser = rootGetters["principal/principalByID"](
        state.currentUser.id
      );
      if (!isEqual(refreshedUser, state.currentUser)) {
        commit("setCurrentUser", refreshedUser);
      }
    }
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
