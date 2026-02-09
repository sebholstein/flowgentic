import { Link2, ExternalLink } from "lucide-react";
import type { Resource } from "@/types/resource";

interface ExternalLinkPreviewProps {
  resource: Resource;
}

/**
 * Preview component for external link resources.
 */
export function ExternalLinkPreview({ resource }: ExternalLinkPreviewProps) {
  const url = resource.storage.type === "url" ? resource.storage.url : "#";

  return (
    <a
      href={url}
      target="_blank"
      rel="noopener noreferrer"
      className="flex items-center gap-2 p-3 bg-slate-500/10 rounded-md border border-slate-500/20 hover:bg-slate-500/20 transition-colors"
    >
      <Link2 className="size-5 text-slate-400" />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">{resource.name}</p>
        <p className="text-xs text-muted-foreground truncate">{url}</p>
      </div>
      <ExternalLink className="size-4 text-muted-foreground" />
    </a>
  );
}
