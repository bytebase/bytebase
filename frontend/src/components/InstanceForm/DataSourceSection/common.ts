export const onlyAllowNumber = (value: string) => {
  return value === "" || /^\d+$/.test(value);
};
