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
