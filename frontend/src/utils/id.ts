import { ResourceId } from "@/types";

export const convertToResourceId = (raw: string): ResourceId => {
  return raw.toLowerCase().replaceAll(" ", "-");
};
