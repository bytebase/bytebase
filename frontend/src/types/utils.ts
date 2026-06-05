export type MaybeRef<T> = { value: T } | T;

export type ValidatedMessage = {
  type: "warning" | "error";
  message: string;
};
