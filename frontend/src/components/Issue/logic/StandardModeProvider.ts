import { defineComponent } from "vue";
import { useCommonLogic } from "./common";
import { provideIssueLogic } from "./index";

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
