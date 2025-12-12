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

export const upsertArray = <T>(array: T[], item: T) => {
  const index = array.indexOf(item);
  if (index < 0) {
    array.push(item);
  }
};

export const sortByDictionary = <T, K>(
  array: T[],
  dictionary: K[],
  mapper: (item: T) => K
) => {
  array.sort((a, b) => {
    let orderA = dictionary.indexOf(mapper(a));
    let orderB = dictionary.indexOf(mapper(b));
    if (orderA < 0) orderA = Number.MAX_VALUE;
    if (orderB < 0) orderB = Number.MAX_VALUE;
    return orderA - orderB;
  });
};
