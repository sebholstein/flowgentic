import { Button } from "@/components/ui/button";
import { FileType, ExternalLink } from "lucide-react";
import type { Resource } from "@/types/resource";

interface PdfPreviewProps {
  resource: Resource;
}

/**
 * Preview component for PDF resources with file info and open button.
 */
export function PdfPreview({ resource }: PdfPreviewProps) {
  return (
    <div className="flex items-center gap-2 p-3 bg-red-500/10 rounded-md border border-red-500/20">
      <FileType className="size-8 text-red-400" />
      <div>
        <p className="text-sm font-medium">{resource.name}</p>
        <p className="text-xs text-muted-foreground">
          {resource.metadata.size
            ? `${(resource.metadata.size / 1024).toFixed(1)} KB`
            : "PDF Document"}
        </p>
      </div>
      <Button variant="outline" size="sm" className="ml-auto gap-1.5">
        <ExternalLink className="size-3" />
        Open
      </Button>
    </div>
  );
}
