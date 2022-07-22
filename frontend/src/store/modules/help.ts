import { defineStore } from "pinia";

export const useHelpStore = defineStore("help", {
  state: () => {
    return {
      currHelpId: "",
      openByDefault: false,
    };
  },
});
