import { cn } from "@/lib/utils";
import { getResourceTypeInfo, getResourceStatusInfo } from "@/lib/resource-flow";
import type { ResourceType, ResourceStatus } from "@/types/resource";
import { FileText, Code, Image, FileType, Database, Link2 } from "lucide-react";

interface ResourceBadgeProps {
  type: ResourceType;
  status?: ResourceStatus;
  showStatus?: boolean;
  size?: "sm" | "md";
  className?: string;
}

const typeIcons: Record<ResourceType, typeof FileText> = {
  markdown: FileText,
  code: Code,
  image: Image,
  pdf: FileType,
  data: Database,
  external: Link2,
};

export function ResourceBadge({
  type,
  status,
  showStatus = false,
  size = "sm",
  className,
}: ResourceBadgeProps) {
  const typeInfo = getResourceTypeInfo(type);
  const statusInfo = status ? getResourceStatusInfo(status) : null;
  const Icon = typeIcons[type];

  const sizeClasses = {
    sm: "h-5 text-[0.6rem] gap-1 px-1.5",
    md: "h-6 text-xs gap-1.5 px-2",
  };

  const iconSizes = {
    sm: "size-3",
    md: "size-3.5",
  };

  return (
    <div
      className={cn(
        "inline-flex items-center rounded border",
        sizeClasses[size],
        typeInfo.bgColor,
        typeInfo.color,
        "border-current/20",
        className,
      )}
    >
      <Icon className={iconSizes[size]} />
      <span>{typeInfo.label}</span>
      {showStatus && statusInfo && (
        <>
          <span className="opacity-50">|</span>
          <span className={statusInfo.color}>{statusInfo.label}</span>
        </>
      )}
    </div>
  );
}

interface ResourceTypeBadgeProps {
  type: ResourceType;
  size?: "sm" | "md";
  className?: string;
}

/**
 * Icon-only badge for compact displays (e.g., task nodes)
 */
export function ResourceTypeIcon({ type, size = "sm", className }: ResourceTypeBadgeProps) {
  const typeInfo = getResourceTypeInfo(type);
  const Icon = typeIcons[type];

  const sizeClasses = {
    sm: "size-4 p-0.5",
    md: "size-5 p-1",
  };

  const iconSizes = {
    sm: "size-3",
    md: "size-3.5",
  };

  return (
    <div
      className={cn(
        "inline-flex items-center justify-center rounded",
        sizeClasses[size],
        typeInfo.bgColor,
        typeInfo.color,
        className,
      )}
    >
      <Icon className={iconSizes[size]} />
    </div>
  );
}

interface ResourceStatusIndicatorProps {
  status: ResourceStatus;
  size?: "sm" | "md";
  className?: string;
}

/**
 * Small status dot indicator
 */
export function ResourceStatusIndicator({
  status,
  size = "sm",
  className,
}: ResourceStatusIndicatorProps) {
  const statusInfo = getResourceStatusInfo(status);

  const sizeClasses = {
    sm: "size-2",
    md: "size-2.5",
  };

  return (
    <div
      className={cn(
        "rounded-full",
        sizeClasses[size],
        status === "approved" && "bg-emerald-500",
        status === "pending" && "bg-amber-500",
        status === "draft" && "bg-slate-500",
        status === "superseded" && "bg-purple-500",
        status === "rejected" && "bg-red-500",
        className,
      )}
      title={statusInfo.label}
    />
  );
}

interface FulfillmentIndicatorProps {
  fulfilled: number;
  total: number;
  direction: "input" | "output";
  size?: "sm" | "md";
  className?: string;
}

/**
 * Shows input/output fulfillment status with color coding
 */
export function FulfillmentIndicator({
  fulfilled,
  total,
  direction,
  size = "sm",
  className,
}: FulfillmentIndicatorProps) {
  if (total === 0) return null;

  const ratio = fulfilled / total;
  const colorClass =
    ratio >= 1
      ? "text-emerald-400 bg-emerald-400/10 border-emerald-500/30"
      : ratio > 0
        ? "text-amber-400 bg-amber-400/10 border-amber-500/30"
        : "text-red-400 bg-red-400/10 border-red-500/30";

  const sizeClasses = {
    sm: "h-4 text-[0.55rem] gap-0.5 px-1",
    md: "h-5 text-[0.6rem] gap-1 px-1.5",
  };

  return (
    <div
      className={cn(
        "inline-flex items-center rounded border font-medium tabular-nums",
        sizeClasses[size],
        colorClass,
        className,
      )}
      title={`${direction === "input" ? "Inputs" : "Outputs"}: ${fulfilled}/${total} fulfilled`}
    >
      <span>{direction === "input" ? "IN" : "OUT"}</span>
      <span>
        {fulfilled}/{total}
      </span>
    </div>
  );
}
