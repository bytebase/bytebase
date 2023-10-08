import { useChangelistDetailContext } from "../context";

export const useReorderChangelist = () => {
  const { reorderMode, events } = useChangelistDetailContext();

  const begin = () => {
    reorderMode.value = true;
  };
  const cancel = () => {
    events.emit("reorder-cancel");
  };
  const confirm = () => {
    events.emit("reorder-confirm");
  };
  return {
    begin,
    cancel,
    confirm,
  };
};
