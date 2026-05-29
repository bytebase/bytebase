import { getCurrentUserV1 } from "@/store";

export const getDefaultPagination = () => {
  if (getCurrentUserV1().email.endsWith("@bytebase.com")) {
    return 10;
  }
  return 50;
};
