import { defineStore } from "pinia";
import { useActuatorStore, useOnboardingGuideStore } from "@/store";

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
      const onboardingGuideName = useOnboardingGuideStore().guideName;

      const demoName = actuatorStore.serverInfo?.demoName;
      const invalidFeatureDemoNameList = ["dev", "prod"];
      const isFeatureDemo =
        demoName && !invalidFeatureDemoNameList.includes(demoName);

      // Do not show help in live demo and onboarding guide
      if (!isFeatureDemo && !onboardingGuideName) {
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
