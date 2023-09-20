import { defineStore } from "pinia";
import { useActuatorV1Store } from "@/store";

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
      const actuatorStore = useActuatorV1Store();

      const demoName = actuatorStore.serverInfo?.demoName;
      const invalidFeatureDemoNameList = ["dev", "prod"];
      const isFeatureDemo =
        demoName && !invalidFeatureDemoNameList.includes(demoName);

      // Do not show help in live demo.
      if (!isFeatureDemo) {
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
