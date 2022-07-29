import { createPinia, defineStore } from "pinia";
import { ProcessData } from "../types";

export const piniaInstance = createPinia();

interface State {
  demoName: string;
  processDataList: ProcessData[];
}

const useAppStore = defineStore("appStore", {
  state: (): State => {
    return {
      demoName: "",
      processDataList: [],
    };
  },
  actions: {
    setState(state: State) {
      this.demoName = state.demoName;
      this.processDataList = state.processDataList;
    },
  },
});

export default useAppStore;
