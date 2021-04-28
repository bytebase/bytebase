import axios from "axios";

const getters = {};

const actions = {
  async ping() {
    const message = (await axios.get(`/api/ping`)).data;
    return message;
  },
};

const mutations = {};

export default {
  namespaced: true,
  getters,
  actions,
  mutations,
};
