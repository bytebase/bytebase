export type VueClass = string | Record<string, unknown> | Array<VueClass>;

export type ExtractPromiseType<T> = T extends Promise<infer U> ? U : unknown;
