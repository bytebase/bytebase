import dayjs from "dayjs";

export const fallbackVersionForChange = () => {
  // Example: 20240101061834
  return dayjs().utc().format("YYYYMMDDHHmmss");
};
