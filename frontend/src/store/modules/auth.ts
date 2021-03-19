import axios from "axios";
import {
  Principal,
  AuthState,
  LoginInfo,
  SignupInfo,
  ActivateInfo,
  ResourceObject,
} from "../../types";

const GUEST: Principal = {
  id: "0",
  status: "ACTIVE",
  name: "Guest",
  email: "guest@bytebase.com",
  role: "GUEST",
};

function convert(user: ResourceObject, rootGetters: any): Principal {
  return rootGetters["principal/principalById"](user.id);
}

const state: () => AuthState = () => ({
  currentUser: GUEST,
});

const getters = {
  currentUser: (state: AuthState) => (): Principal => {
    if (state.currentUser) {
      return state.currentUser;
    }
    const user = localStorage.getItem("bb.auth.user");
    if (user) {
      return JSON.parse(user);
    }
    return GUEST;
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

  async signup({ commit, rootGetters }: any, signupInfo: SignupInfo) {
    const newUser = convert(
      (
        await axios.post("/api/auth/signup", {
          data: { type: "signupInfo", attributes: signupInfo },
        })
      ).data.data,
      rootGetters
    );

    localStorage.setItem("bb.auth.user", JSON.stringify(newUser));
    commit("setCurrentUser", newUser);
    return newUser;
  },

  async activate({ commit, rootGetters }: any, activateInfo: ActivateInfo) {
    const activatedUser = convert(
      (
        await axios.post("/api/auth/activate", {
          data: { type: "activateInfo", attributes: activateInfo },
        })
      ).data.data,
      rootGetters
    );

    localStorage.setItem("bb.auth.user", JSON.stringify(activatedUser));
    commit("setCurrentUser", activatedUser);
    return activatedUser;
  },

  async restoreUser({ commit }: any) {
    const jsonUser = localStorage.getItem("bb.auth.user");
    if (jsonUser) {
      const user: Principal = JSON.parse(jsonUser);
      commit("setCurrentUser", user);
      return user;
    }
    return undefined;
  },

  async logout({ commit }: any) {
    localStorage.removeItem("bb.auth.user");
    commit("setCurrentUser", undefined);
    return GUEST;
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
