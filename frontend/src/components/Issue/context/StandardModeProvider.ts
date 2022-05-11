import { defineComponent } from "vue";
import { provideIssueContext } from "./index";
import { useCommonLogic } from "./common";

export default defineComponent({
  name: "StandardModeProvider",
  setup() {
    const common = useCommonLogic();

    const context = common;
    provideIssueContext(context);
    return { ...context };
  },
  render() {
    return this.$slots.default?.();
  },
});
