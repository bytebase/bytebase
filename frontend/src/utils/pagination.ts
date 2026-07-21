import { getCurrentUserV1 } from "@/stores";

export const getDefaultPagination = () => {
  if (getCurrentUserV1().email.endsWith("@bytebase.com")) {
    return 10;
  }
  return 50;
};
