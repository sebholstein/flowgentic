import { cn } from "@/lib/utils";
import { Markdown } from "@/components/ui/markdown";

interface MarkdownPreviewProps {
  content: string;
  expanded: boolean;
}

/**
 * Preview component for markdown resources.
 */
export function MarkdownPreview({ content, expanded }: MarkdownPreviewProps) {
  return (
    <div className={cn("prose prose-sm prose-invert max-w-none", !expanded && "line-clamp-4")}>
      <Markdown>{content}</Markdown>
    </div>
  );
}
