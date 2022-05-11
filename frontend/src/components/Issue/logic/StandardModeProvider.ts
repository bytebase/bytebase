import { defineComponent } from "vue";
import { provideIssueLogic } from "./index";
import { useCommonLogic } from "./common";

export default defineComponent({
  name: "StandardModeProvider",
  setup() {
    const common = useCommonLogic();

    provideIssueLogic(common);
    return common;
  },
  render() {
    return this.$slots.default?.();
  },
});
