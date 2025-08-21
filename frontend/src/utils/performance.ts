/**
 * Performance utilities for SQL Editor optimizations
 */
import { debounce } from "lodash-es";

/**
 * Optimized content change handler for Monaco Editor
 */
export const createOptimizedContentHandler = (
  onContentChange: (content: string) => void,
  onImmediate?: (content: string) => void
) => {
  const debouncedHandler = debounce(onContentChange, 150);

  return (content: string) => {
    // Call immediate handler for UI responsiveness if provided
    if (onImmediate) {
      onImmediate(content);
    }

    // Debounce the heavy operations
    debouncedHandler(content);
  };
};
