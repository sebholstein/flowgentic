import { cn } from "@/lib/utils";
import type { Resource } from "@/types/resource";

interface ImagePreviewProps {
  resource: Resource;
  expanded: boolean;
}

/**
 * Preview component for image resources.
 */
export function ImagePreview({ resource, expanded }: ImagePreviewProps) {
  const src = resource.storage.type === "file" ? resource.storage.path : resource.storage.url;

  return (
    <div
      className={cn("relative bg-slate-900 rounded-md overflow-hidden", !expanded && "max-h-32")}
    >
      <img src={src} alt={resource.name} className="w-full h-auto object-contain" />
    </div>
  );
}
