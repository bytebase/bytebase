import { Search } from "lucide-react";
import type { ComponentProps } from "react";
import { forwardRef } from "react";
import { cn } from "@/react/lib/utils";
import { Input } from "./input";

type SearchInputProps = Omit<ComponentProps<typeof Input>, "className"> & {
  className?: string;
  wrapperClassName?: string;
};

export const SearchInput = forwardRef<HTMLInputElement, SearchInputProps>(
  function SearchInput({ className, wrapperClassName, ...props }, ref) {
    return (
      <div className={cn("relative flex-1", wrapperClassName)}>
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-control-placeholder pointer-events-none" />
        <Input
          ref={ref}
          className={cn("h-9 text-sm pl-8", className)}
          {...props}
        />
      </div>
    );
  }
);
