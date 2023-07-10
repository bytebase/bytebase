import { defineComponent, onMounted } from "vue";
import { useProjectV1Store } from "@/store";
import { provideIssueLogic, useCommonLogic } from "./index";

export default defineComponent({
  name: "GrantRequestModeProvider",
  setup() {
    onMounted(() => {
      useProjectV1Store().fetchProjectList();
    });

    const logic = {
      ...useCommonLogic(),
    };
    provideIssueLogic(logic);
    return logic;
  },
  render() {
    return this.$slots.default?.();
  },
});
