export const RESIZE_POINTER_MEDIA_QUERY = "(any-pointer: fine)";

export function supportsWindowBorderResize(
  matchMedia: (query: string) => { matches: boolean }
): boolean {
  return matchMedia(RESIZE_POINTER_MEDIA_QUERY).matches;
}
