import { cn } from "@/lib/utils";
import type {
  InputRequirement,
  OutputRequirement,
  ResourceRef,
  FulfillmentStatus,
} from "@/types/resource";
import { ResourceTypeIcon } from "./ResourceBadge";
import {
  CheckCircle2,
  Circle,
  AlertCircle,
  ArrowDownToLine,
  ArrowUpFromLine,
  ChevronRight,
} from "lucide-react";

interface RequirementItemProps {
  requirement: InputRequirement | OutputRequirement;
  linkedResource?: ResourceRef;
  onResourceClick?: (resourceId: string) => void;
}

function FulfillmentIcon({ status }: { status: FulfillmentStatus }) {
  switch (status) {
    case "fulfilled":
      return <CheckCircle2 className="size-3.5 text-emerald-500 shrink-0" />;
    case "partial":
      return <AlertCircle className="size-3.5 text-amber-500 shrink-0" />;
    case "unfulfilled":
      return <Circle className="size-3.5 text-slate-500 shrink-0" />;
  }
}

function RequirementItem({ requirement, linkedResource, onResourceClick }: RequirementItemProps) {
  const isFulfilled = requirement.fulfillmentStatus === "fulfilled";
  const hasErrors = requirement.validationErrors && requirement.validationErrors.length > 0;

  return (
    <div
      className={cn(
        "flex items-start gap-2 rounded-md border p-2",
        isFulfilled && "bg-emerald-500/5 border-emerald-500/20",
        requirement.fulfillmentStatus === "partial" && "bg-amber-500/5 border-amber-500/20",
        requirement.fulfillmentStatus === "unfulfilled" && "border-border",
      )}
    >
      <FulfillmentIcon status={requirement.fulfillmentStatus} />
      <div className="flex-1 min-w-0 space-y-1">
        <div className="flex items-center gap-1.5 flex-wrap">
          <span
            className={cn(
              "text-xs font-medium",
              isFulfilled && "text-muted-foreground line-through",
            )}
          >
            {requirement.name}
          </span>
          <ResourceTypeIcon type={requirement.expectedType} size="sm" />
          {requirement.isRequired && (
            <span className="text-[0.55rem] text-red-400 font-medium">REQ</span>
          )}
        </div>
        <p className="text-[0.65rem] text-muted-foreground line-clamp-2">
          {requirement.description}
        </p>

        {/* Show linked resource if fulfilled or partial */}
        {linkedResource && (
          <button
            onClick={() => onResourceClick?.(linkedResource.resourceId)}
            className="flex items-center gap-1 text-[0.6rem] text-blue-400 hover:text-blue-300 transition-colors"
          >
            <ChevronRight className="size-2.5" />
            <span>{linkedResource.name}</span>
          </button>
        )}

        {/* Show validation errors */}
        {hasErrors && (
          <div className="space-y-0.5">
            {requirement.validationErrors!.map((error, i) => (
              <p key={i} className="text-[0.6rem] text-amber-400">
                {error}
              </p>
            ))}
          </div>
        )}

        {/* Show acceptance criteria if present */}
        {requirement.acceptanceCriteria && requirement.acceptanceCriteria.length > 0 && (
          <div className="text-[0.6rem] text-muted-foreground">
            <span className="font-medium">Criteria:</span>
            <ul className="list-disc list-inside ml-1 mt-0.5">
              {requirement.acceptanceCriteria.map((criterion, i) => (
                <li key={i}>{criterion}</li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  );
}

interface ResourceRequirementListProps {
  inputRequirements?: InputRequirement[];
  outputRequirements?: OutputRequirement[];
  availableResources?: ResourceRef[];
  producedResources?: ResourceRef[];
  onResourceClick?: (resourceId: string) => void;
  compact?: boolean;
  className?: string;
}

export function ResourceRequirementList({
  inputRequirements,
  outputRequirements,
  availableResources = [],
  producedResources = [],
  onResourceClick,
  compact = false,
  className,
}: ResourceRequirementListProps) {
  const hasInputs = inputRequirements && inputRequirements.length > 0;
  const hasOutputs = outputRequirements && outputRequirements.length > 0;

  if (!hasInputs && !hasOutputs) {
    return null;
  }

  // Find linked resources for each requirement
  const findLinkedResource = (resourceId?: string) => {
    if (!resourceId) return undefined;
    return (
      availableResources.find((r) => r.resourceId === resourceId) ||
      producedResources.find((r) => r.resourceId === resourceId)
    );
  };

  // Calculate summary stats
  const inputStats = hasInputs
    ? {
        fulfilled: inputRequirements.filter((r) => r.fulfillmentStatus === "fulfilled").length,
        total: inputRequirements.filter((r) => r.isRequired).length,
      }
    : null;

  const outputStats = hasOutputs
    ? {
        fulfilled: outputRequirements.filter((r) => r.fulfillmentStatus === "fulfilled").length,
        total: outputRequirements.filter((r) => r.isRequired).length,
      }
    : null;

  if (compact) {
    // Compact view - just show summary badges
    return (
      <div className={cn("flex items-center gap-2", className)}>
        {inputStats && inputStats.total > 0 && (
          <div
            className={cn(
              "flex items-center gap-1 text-[0.6rem] px-1.5 py-0.5 rounded border",
              inputStats.fulfilled === inputStats.total
                ? "text-emerald-400 bg-emerald-400/10 border-emerald-500/30"
                : inputStats.fulfilled > 0
                  ? "text-amber-400 bg-amber-400/10 border-amber-500/30"
                  : "text-red-400 bg-red-400/10 border-red-500/30",
            )}
          >
            <ArrowDownToLine className="size-2.5" />
            <span className="font-medium tabular-nums">
              {inputStats.fulfilled}/{inputStats.total}
            </span>
          </div>
        )}
        {outputStats && outputStats.total > 0 && (
          <div
            className={cn(
              "flex items-center gap-1 text-[0.6rem] px-1.5 py-0.5 rounded border",
              outputStats.fulfilled === outputStats.total
                ? "text-emerald-400 bg-emerald-400/10 border-emerald-500/30"
                : outputStats.fulfilled > 0
                  ? "text-amber-400 bg-amber-400/10 border-amber-500/30"
                  : "text-slate-400 bg-slate-400/10 border-slate-500/30",
            )}
          >
            <ArrowUpFromLine className="size-2.5" />
            <span className="font-medium tabular-nums">
              {outputStats.fulfilled}/{outputStats.total}
            </span>
          </div>
        )}
      </div>
    );
  }

  return (
    <div className={cn("space-y-4", className)}>
      {/* Input Requirements */}
      {hasInputs && (
        <div>
          <div className="flex items-center gap-2 mb-2">
            <ArrowDownToLine className="size-3.5 text-blue-400" />
            <span className="text-xs font-medium text-muted-foreground">Input Requirements</span>
            {inputStats && (
              <span className="text-[0.6rem] text-muted-foreground tabular-nums">
                ({inputStats.fulfilled}/{inputStats.total} required)
              </span>
            )}
          </div>
          <div className="space-y-1.5">
            {inputRequirements.map((req) => (
              <RequirementItem
                key={req.id}
                requirement={req}
                linkedResource={findLinkedResource(req.fulfilledByResourceId)}
                onResourceClick={onResourceClick}
              />
            ))}
          </div>
        </div>
      )}

      {/* Output Requirements */}
      {hasOutputs && (
        <div>
          <div className="flex items-center gap-2 mb-2">
            <ArrowUpFromLine className="size-3.5 text-emerald-400" />
            <span className="text-xs font-medium text-muted-foreground">Output Requirements</span>
            {outputStats && (
              <span className="text-[0.6rem] text-muted-foreground tabular-nums">
                ({outputStats.fulfilled}/{outputStats.total} required)
              </span>
            )}
          </div>
          <div className="space-y-1.5">
            {outputRequirements.map((req) => (
              <RequirementItem
                key={req.id}
                requirement={req}
                linkedResource={findLinkedResource(req.fulfilledByResourceId)}
                onResourceClick={onResourceClick}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
