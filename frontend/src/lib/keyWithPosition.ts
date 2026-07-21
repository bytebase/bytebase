const KEY_WITH_POSITION_DELIMITER = "###";

export const keyWithPosition = (key: string, position: number) => {
  return `${key}${KEY_WITH_POSITION_DELIMITER}${position}`;
};

export const extractKeyWithPosition = (key: string) => {
  const [maybeKey, maybePosition] = key.split(KEY_WITH_POSITION_DELIMITER);
  const position = parseInt(maybePosition, 10);
  return [maybeKey, Number.isNaN(position) ? -1 : position];
};
