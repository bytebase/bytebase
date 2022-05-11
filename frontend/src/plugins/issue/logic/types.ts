import { Ref } from "vue";
import { useRoute } from "vue-router";
import { IssueTemplate } from "@/plugins/types";

export type BuildNewIssueContext = {
  template: Ref<IssueTemplate>;
  route: ReturnType<typeof useRoute>;
};
