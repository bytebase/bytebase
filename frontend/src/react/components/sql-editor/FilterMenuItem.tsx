import { Switch } from "@/react/components/ui/switch";

type Props = {
  readonly label: string;
  readonly value: boolean;
  readonly onValueChange: (value: boolean) => void;
};

export function FilterMenuItem({ label, value, onValueChange }: Props) {
  return (
    <div className="flex flex-row items-center justify-between gap-x-4 px-3 py-2 text-sm">
      <span>{label}</span>
      <Switch size="sm" checked={value} onCheckedChange={onValueChange} />
    </div>
  );
}
