import { ref } from "vue";
import { useChangelistDetailContext } from "../context";

export const useReorderChangelist = () => {
  const { reorderMode } = useChangelistDetailContext();

  const updating = ref(false);

  const begin = () => {
    reorderMode.value = true;
  };
  const cancel = () => {
    reorderMode.value = false;
  };
  const confirm = async () => {
    updating.value = true;
    await new Promise((r) => setTimeout(r, 1000));
    reorderMode.value = false;
    updating.value = false;
  };
  return {
    updating,
    begin,
    cancel,
    confirm,
  };
};
