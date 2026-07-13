// Convert category value to CEL filter expression
export const buildCategoryFilter = (category: string | undefined): string => {
  if (!category) return "";
  return `category == "${category}"`;
};
