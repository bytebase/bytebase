import { createPinia, defineStore } from "pinia";
import { HintData, ProcessData } from "../types";

export const piniaInstance = createPinia();

interface State {
  demoName: string;
  processDataList: ProcessData[];
  hintDataList: HintData[];
}

const useAppStore = defineStore("appStore", {
  state: (): State => {
    return {
      demoName: "",
      processDataList: [],
      hintDataList: [],
    };
  },
  actions: {
    setState(state: State) {
      this.demoName = state.demoName;
      this.processDataList = state.processDataList;
      this.hintDataList = state.hintDataList;
    },
  },
});

export default useAppStore;
