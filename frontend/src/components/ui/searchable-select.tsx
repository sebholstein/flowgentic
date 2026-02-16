import { useState, useMemo } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Check, ChevronsUpDown } from "lucide-react";
import Fuse from "fuse.js";

export interface SearchableSelectItem {
  id: string;
  name: string;
  description?: string;
  icon?: React.ReactNode;
  trailing?: React.ReactNode;
}

interface SearchableSelectProps {
  items: SearchableSelectItem[];
  selectedId: string;
  onSelect: (id: string) => void;
  placeholder?: string;
}

export function SearchableSelect({
  items,
  selectedId,
  onSelect,
  placeholder = "Search…",
}: SearchableSelectProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const selected = items.find((item) => item.id === selectedId);

  const fuse = useMemo(
    () =>
      new Fuse(items, {
        keys: ["id", "name", "description"],
        threshold: 0.4,
        ignoreLocation: true,
      }),
    [items],
  );

  const filtered = query ? fuse.search(query).map((r) => r.item) : items;

  return (
    <Popover open={open} onOpenChange={(v) => { setOpen(v); if (!v) setQuery(""); }}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full h-7 justify-between text-xs font-normal bg-input/20 dark:bg-input/30 border-input px-2"
        >
          {selected ? (
            <span className="flex items-center gap-2 truncate">
              {selected.icon}
              <span className="truncate">{selected.name}</span>
              {selected.trailing}
            </span>
          ) : (
            <span className="text-muted-foreground">{placeholder}</span>
          )}
          <ChevronsUpDown className="ml-auto size-3 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[--radix-popover-trigger-width] p-0">
        <Command shouldFilter={false} className="p-0">
          <CommandInput
            placeholder={placeholder}
            className="h-7 text-xs"
            value={query}
            onValueChange={setQuery}
          />
          <CommandList>
            <CommandEmpty>No results.</CommandEmpty>
            <CommandGroup className="p-1">
              {filtered.map((item) => (
                <CommandItem
                  key={item.id}
                  value={item.id}
                  onSelect={(val) => {
                    onSelect(val);
                    setOpen(false);
                    setQuery("");
                  }}
                  className="py-1 px-2 gap-1.5"
                >
                  <Check
                    className={cn(
                      "size-3 shrink-0",
                      selectedId === item.id ? "opacity-100" : "opacity-0",
                    )}
                  />
                  {item.icon}
                  <div className="flex-1 min-w-0">
                    <span className="truncate block">{item.name}</span>
                    {item.description && (
                      <span className="text-xs text-muted-foreground truncate block">{item.description}</span>
                    )}
                  </div>
                  {item.trailing}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
