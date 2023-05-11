// parse expired time string from expression string for issue grant request paylod.
// e.g. timestamp("2021-08-31T00:00:00Z") => Date("2021-08-31T00:00:00Z")
export const parseExpiredTimeString = (expiredTime: string): Date => {
  const regex = /timestamp\("(.+?)"\)/;
  const match = expiredTime.match(regex);
  if (!match) {
    throw new Error(`Invalid expired time: ${expiredTime}`);
  }
  return new Date(match[1]);
};
