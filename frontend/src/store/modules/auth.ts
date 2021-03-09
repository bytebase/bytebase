import axios from "axios";
import {
  User,
  AuthState,
  LoginInfo,
  SignupInfo,
  ActivateInfo,
  ResourceObject,
} from "../../types";

function convert(user: ResourceObject): User {
  return {
    id: user.id,
    ...(user.attributes as Omit<User, "id">),
  };
}

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
    const loggedInUser = convert(
      (
        await axios.post("/api/auth/login", {
          data: { type: "loginInfo", attributes: loginInfo },
        })
      ).data.data
    );

    localStorage.setItem("bb.auth.user", JSON.stringify(loggedInUser));
    commit("setCurrentUser", loggedInUser);
    return loggedInUser;
  },

  async signup({ commit }: any, signupInfo: SignupInfo) {
    const newUser = convert(
      (
        await axios.post("/api/auth/signup", {
          data: { type: "signupInfo", attributes: signupInfo },
        })
      ).data.data
    );

    localStorage.setItem("bb.auth.user", JSON.stringify(newUser));
    commit("setCurrentUser", newUser);
    return newUser;
  },

  async activate({ commit }: any, activateInfo: ActivateInfo) {
    const activatedUser = convert(
      (
        await axios.post("/api/auth/activate", {
          data: { type: "activateInfo", attributes: activateInfo },
        })
      ).data.data
    );

    localStorage.setItem("bb.auth.user", JSON.stringify(activatedUser));
    commit("setCurrentUser", activatedUser);
    return activatedUser;
  },

  async fetchCurrentUser({ commit }: any) {
    const currentUser = convert((await axios.get("/api/user/1")).data.data);

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
