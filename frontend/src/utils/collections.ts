export const groupBy = <T, K>(
  array: T[],
  keyOf: (item: T) => K
): Map<K, T[]> => {
  return array.reduce((map, item) => {
    const key = keyOf(item);
    const list = map.get(key) ?? map.set(key, []).get(key)!;
    list.push(item);

    return map;
  }, new Map<K, T[]>());
};

export const keyBy = <T, K>(array: T[], keyOf: (item: T) => K): Map<K, T> => {
  return array.reduce((map, item) => {
    const key = keyOf(item);
    map.set(key, item);
    return map;
  }, new Map<K, T>());
};

export const wrapArray = <T>(arrayOrPrimitive: T | T[]): T[] => {
  if (Array.isArray(arrayOrPrimitive)) return arrayOrPrimitive;
  return [arrayOrPrimitive];
};

export const unwrapArray = <T>(arrayOrPrimitive: T | T[]): T => {
  if (Array.isArray(arrayOrPrimitive)) return arrayOrPrimitive[0];
  return arrayOrPrimitive;
};
