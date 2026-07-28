export type EnvironmentSelectionItem = {
  id: string;
};

export const getEnvironmentListKey = (
  environmentList: EnvironmentSelectionItem[]
): string => environmentList.map((environment) => environment.id).join("\0");

export const resolveSelectedEnvironmentId = ({
  currentId,
  environmentList,
  hash,
}: {
  currentId: string;
  environmentList: EnvironmentSelectionItem[];
  hash: string;
}): string => {
  if (environmentList.length === 0) {
    return "";
  }

  const hashMatch = hash
    ? environmentList.find((environment) => environment.id === hash)
    : undefined;
  if (hashMatch) {
    return hashMatch.id;
  }

  if (
    currentId &&
    environmentList.some((environment) => environment.id === currentId)
  ) {
    return currentId;
  }

  return environmentList[0].id;
};
