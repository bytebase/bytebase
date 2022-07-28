import { defineStore } from "pinia";
import { useActuatorStore } from "@/store";

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
      const actuatorStore = useActuatorStore();
      const demoName = actuatorStore.serverInfo?.demoName;
      const invalidFeatureDemoNameList = ["dev", "prod"];
      const isFeatureDemo =
        demoName && !invalidFeatureDemoNameList.includes(demoName);
      if (!isFeatureDemo) {
        // only show help when not in live demo
        this.currHelpId = id;
        this.openByDefault = openByDefault;
      }
    },
    exitHelp() {
      this.currHelpId = "";
      this.openByDefault = false;
    },
  },
});
