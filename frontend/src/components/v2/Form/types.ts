export type RadioGridOption<T extends string | number> = {
  value: T;
  label?: string;
};
export type RadioGridItem<T extends string | number> = {
  option: RadioGridOption<T>;
  index: number;
};
