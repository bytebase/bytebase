import { defineStore } from "pinia";

export const useHelpStore = defineStore("help", {
  state: (): {
    currHelpId: string;
    openByDefault: boolean;
    timer: number | null;
  } => {
    return {
      currHelpId: "",
      openByDefault: false,
      timer: null,
    };
  },
  actions: {
    showHelp(id: string, openByDefault: boolean): void {
      this.currHelpId = id;
      this.openByDefault = openByDefault;
      this.timer = null;
    },
    exitHelp() {
      this.currHelpId = "";
      this.openByDefault = false;
    },
    setTimer(timer: number) {
      this.timer = timer;
    },
  },
});
