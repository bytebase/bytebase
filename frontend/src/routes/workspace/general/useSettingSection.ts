/**
 * Handle exposed by each setting section via useImperativeHandle.
 * The parent GeneralPage calls these methods to coordinate save/cancel.
 */
export interface SectionHandle {
  isDirty: () => boolean;
  revert: () => void;
  update: () => Promise<void>;
}
