import { Binding } from "@/types/proto/v1/iam_policy";

export const getBindingConditionTitle = (binding: Binding): string => {
  return binding.condition?.title || "";
};
