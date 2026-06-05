export type VueClass = string | Record<string, unknown> | Array<VueClass>;

export type VueStyle =
  | string
  | Record<string, string | number | undefined>
  | Array<VueStyle>;

export type ExtractPromiseType<T> = T extends Promise<infer U> ? U : unknown;
