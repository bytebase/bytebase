// Convert category value to CEL filter expression
export const buildCategoryFilter = (category: string | undefined): string => {
  if (!category) return "";
  return `category == "${category}"`;
};

// Parse category from URL query params
export const parseCategoryFromUrl = (
  query: Record<string, unknown>
): string | undefined => {
  const category = query.category;
  if (!category || typeof category !== "string") return undefined;
  return category;
};

// Build URL query params from category
export const buildCategoryQuery = (
  category: string | undefined
): Record<string, string> => {
  if (!category) return {};
  return { category };
};
