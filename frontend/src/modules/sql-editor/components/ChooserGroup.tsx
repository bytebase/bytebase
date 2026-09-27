import { ContainerChooser } from "./ContainerChooser";
import { DatabaseChooser } from "./DatabaseChooser";
import { SchemaChooser } from "./SchemaChooser";

export function ChooserGroup() {
  return (
    <div className="flex items-center">
      <DatabaseChooser />
      <SchemaChooser />
      <ContainerChooser />
    </div>
  );
}
