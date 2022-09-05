import { defineStore } from "pinia";
import { OnboardingGuideType, OnboardingGuideState } from "@/types";

export const useOnboardingGuideStore = defineStore("onboarding_guide", {
  state: (): OnboardingGuideState => ({
    guideName: undefined,
  }),
  actions: {
    setGuideName(guideName: OnboardingGuideType) {
      this.guideName = guideName;
    },
    removeGuide() {
      this.guideName = undefined;
    },
  },
});
