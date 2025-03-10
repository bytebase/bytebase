import { useCurrentUserV1 } from "@/store";

export const getDefaultPagination = () => {
  if (useCurrentUserV1().value.email.endsWith("@bytebase.com")) {
    return 1;
  }
  return 50;
};
