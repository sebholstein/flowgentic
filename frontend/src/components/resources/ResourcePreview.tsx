import { useState } from "react";
import { cn } from "@/lib/utils";
import { getResourceTypeInfo, getResourceStatusInfo } from "@/lib/resource-flow";
import type { Resource, ResourceRef } from "@/types/resource";
import { ResourceBadge, ResourceStatusIndicator } from "./ResourceBadge";
import { Button } from "@/components/ui/button";
import {
  FileText,
  Code,
  Image,
  FileType,
  Database,
  Link2,
  ChevronDown,
  ChevronUp,
  User,
  Bot,
  Cpu,
} from "lucide-react";

// Import type-specific preview components
import { CodePreview } from "./previews/CodePreview";
import { MarkdownPreview } from "./previews/MarkdownPreview";
import { ImagePreview } from "./previews/ImagePreview";
import { PdfPreview } from "./previews/PdfPreview";
import { DataPreview } from "./previews/DataPreview";
import { ExternalLinkPreview } from "./previews/ExternalLinkPreview";

// ============================================================================
// Resource Content Renderer
// ============================================================================

interface ResourceContentProps {
  resource: Resource;
  expanded: boolean;
}

/**
 * Routes to the appropriate type-specific preview component.
 */
function ResourceContent({ resource, expanded }: ResourceContentProps) {
  // For now, we don't have actual content - this would load from storage
  // In a real implementation, this would fetch content based on resource.storage
  const mockContent = getMockContent(resource);

  switch (resource.type) {
    case "markdown":
      return <MarkdownPreview content={mockContent} expanded={expanded} />;

    case "code":
      return <CodePreview resource={resource} content={mockContent} expanded={expanded} />;

    case "image":
      return <ImagePreview resource={resource} expanded={expanded} />;

    case "pdf":
      return <PdfPreview resource={resource} />;

    case "data":
      return <DataPreview content={mockContent} expanded={expanded} />;

    case "external":
      return <ExternalLinkPreview resource={resource} />;

    default:
      return (
        <div className="text-xs text-muted-foreground p-2">
          Preview not available for this resource type
        </div>
      );
  }
}

// ============================================================================
// Provenance Display
// ============================================================================

function ProvenanceInfo({ resource }: { resource: Resource }) {
  const { createdBy } = resource.provenance;

  return (
    <div className="flex items-center gap-1.5 text-[0.6rem] text-muted-foreground">
      {createdBy.type === "user" && (
        <>
          <User className="size-3" />
          <span>{createdBy.userName}</span>
        </>
      )}
      {createdBy.type === "execution" && (
        <>
          <Bot className="size-3" />
          <span>Agent execution</span>
        </>
      )}
      {createdBy.type === "system" && (
        <>
          <Cpu className="size-3" />
          <span>{createdBy.source}</span>
        </>
      )}
      <span className="opacity-50">|</span>
      <span>{resource.provenance.createdAt}</span>
      {resource.version > 1 && (
        <>
          <span className="opacity-50">|</span>
          <span>v{resource.version}</span>
        </>
      )}
    </div>
  );
}

// ============================================================================
// Main Resource Preview Component
// ============================================================================

interface ResourcePreviewProps {
  resource: Resource;
  onExpand?: () => void;
  compact?: boolean;
  className?: string;
}

export function ResourcePreview({
  resource,
  onExpand,
  compact = false,
  className,
}: ResourcePreviewProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const typeInfo = getResourceTypeInfo(resource.type);
  const statusInfo = getResourceStatusInfo(resource.status);

  const toggleExpand = () => {
    setIsExpanded(!isExpanded);
    if (!isExpanded) {
      onExpand?.();
    }
  };

  if (compact) {
    // Minimal preview for lists
    return (
      <div
        className={cn(
          "flex items-center gap-2 rounded-md border p-2 hover:bg-muted/50 transition-colors cursor-pointer",
          className,
        )}
        onClick={toggleExpand}
      >
        <ResourceBadge type={resource.type} size="sm" />
        <span className="text-xs font-medium truncate flex-1">{resource.name}</span>
        <ResourceStatusIndicator status={resource.status} />
      </div>
    );
  }

  return (
    <div className={cn("rounded-lg border bg-card", className)}>
      {/* Header */}
      <div className="flex items-start gap-3 p-3 border-b">
        <div className={cn("rounded-md p-2", typeInfo.bgColor)}>
          {resource.type === "markdown" && <FileText className={cn("size-4", typeInfo.color)} />}
          {resource.type === "code" && <Code className={cn("size-4", typeInfo.color)} />}
          {resource.type === "image" && <Image className={cn("size-4", typeInfo.color)} />}
          {resource.type === "pdf" && <FileType className={cn("size-4", typeInfo.color)} />}
          {resource.type === "data" && <Database className={cn("size-4", typeInfo.color)} />}
          {resource.type === "external" && <Link2 className={cn("size-4", typeInfo.color)} />}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <h4 className="text-sm font-medium">{resource.name}</h4>
            <span
              className={cn(
                "text-[0.55rem] px-1.5 py-0.5 rounded",
                statusInfo.bgColor,
                statusInfo.color,
              )}
            >
              {statusInfo.label}
            </span>
          </div>
          {resource.description && (
            <p className="text-xs text-muted-foreground mt-0.5 line-clamp-1">
              {resource.description}
            </p>
          )}
          <ProvenanceInfo resource={resource} />
        </div>
        <Button variant="ghost" size="sm" className="h-6 w-6 p-0 shrink-0" onClick={toggleExpand}>
          {isExpanded ? <ChevronUp className="size-4" /> : <ChevronDown className="size-4" />}
        </Button>
      </div>

      {/* Content preview */}
      <div className="p-3">
        <ResourceContent resource={resource} expanded={isExpanded} />
      </div>

      {/* Tags */}
      {resource.tags && resource.tags.length > 0 && (
        <div className="flex items-center gap-1.5 px-3 pb-3 flex-wrap">
          {resource.tags.map((tag) => (
            <span
              key={tag}
              className="text-[0.55rem] px-1.5 py-0.5 rounded bg-muted text-muted-foreground"
            >
              #{tag}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Resource Card (Compact List Item)
// ============================================================================

interface ResourceCardProps {
  resource: ResourceRef;
  onClick?: () => void;
  className?: string;
}

export function ResourceCard({ resource, onClick, className }: ResourceCardProps) {
  const typeInfo = getResourceTypeInfo(resource.type);

  return (
    <button
      onClick={onClick}
      className={cn(
        "flex items-center gap-2 w-full rounded-md border p-2 text-left hover:bg-muted/50 transition-colors",
        className,
      )}
    >
      <div className={cn("rounded p-1", typeInfo.bgColor)}>
        {resource.type === "markdown" && <FileText className={cn("size-3", typeInfo.color)} />}
        {resource.type === "code" && <Code className={cn("size-3", typeInfo.color)} />}
        {resource.type === "image" && <Image className={cn("size-3", typeInfo.color)} />}
        {resource.type === "pdf" && <FileType className={cn("size-3", typeInfo.color)} />}
        {resource.type === "data" && <Database className={cn("size-3", typeInfo.color)} />}
        {resource.type === "external" && <Link2 className={cn("size-3", typeInfo.color)} />}
      </div>
      <span className="text-xs font-medium truncate flex-1">{resource.name}</span>
      <ResourceStatusIndicator status={resource.status} />
    </button>
  );
}

// ============================================================================
// Mock Content Generator
// ============================================================================

function getMockContent(resource: Resource): string {
  switch (resource.type) {
    case "markdown":
      return `## ${resource.name}\n\nThis is a preview of the markdown content. In a real implementation, this would be loaded from the storage location.\n\n- Item 1\n- Item 2\n- Item 3`;
    case "code":
      return `// ${resource.name}\nfunction example() {\n  console.log("Hello, world!");\n  return true;\n}`;
    case "data":
      return JSON.stringify(
        { name: resource.name, type: resource.type, version: resource.version },
        null,
        2,
      );
    default:
      return "";
  }
}
