import { defineStore } from "pinia";

export const useRouterStore = defineStore("router", {
  // need not to initialize a state since we store everything into localStorage
  // state: () => ({}),

  getters: {
    backPath: () => () => {
      return localStorage.getItem("ui.backPath") || "/";
    },
  },
  actions: {
    setBackPath(backPath: string) {
      localStorage.setItem("ui.backPath", backPath);
      return backPath;
    },
  },
});
