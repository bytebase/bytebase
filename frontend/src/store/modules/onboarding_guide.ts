import { defineStore } from "pinia";
import { OnboardingGuideState } from "@/types";

export const useOnboardingGuideStore = defineStore("onboarding_guide", {
  state: (): OnboardingGuideState => ({
    guideName: "",
  }),
  actions: {
    setGuideName(guideName: string) {
      this.guideName = guideName;
    },
  },
});
