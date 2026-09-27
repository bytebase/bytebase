export const useGracefulRequest = async <T>(
  fn: () => Promise<T>,
  errorHandler: (err: unknown) => void = (err) => {
    throw err;
  }
): Promise<T | void> => {
  try {
    const result = await fn();
    return result;
  } catch (err) {
    return errorHandler(err);
  }
};
