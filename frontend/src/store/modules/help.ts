import { defineStore } from "pinia";

export const useHelpStore = defineStore("help", {
  state: (): {
    currHelpId: string;
    openByDefault: boolean;
  } => {
    return {
      currHelpId: "",
      openByDefault: false,
    };
  },
  actions: {
    showHelp(id: string, openByDefault: boolean): void {
      this.currHelpId = id;
      this.openByDefault = openByDefault;
    },
    exitHelp() {
      this.currHelpId = "";
      this.openByDefault = false;
    },
  },
});
