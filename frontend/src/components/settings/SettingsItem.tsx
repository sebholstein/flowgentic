import { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface SettingsItemProps {
  label: string;
  description?: ReactNode;
  children: ReactNode;
  className?: string;
  /** Shows a subtle separator above this item (automatic for all but first item in a section) */
  showSeparator?: boolean;
}

export function SettingsItem({
  label,
  description,
  children,
  className,
  showSeparator = true,
}: SettingsItemProps) {
  return (
    <div
      className={cn(
        "flex items-center justify-between gap-4 px-4 py-3",
        showSeparator && "border-t border-border-card first:border-t-0",
        className,
      )}
    >
      <div className="min-w-0 flex-1">
        <div className="text-sm font-medium text-foreground">{label}</div>
        {description && <div className="mt-0.5 text-xs text-muted-foreground">{description}</div>}
      </div>
      <div className="flex-shrink-0">{children}</div>
    </div>
  );
}
