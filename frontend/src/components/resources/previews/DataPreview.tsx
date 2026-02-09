import { cn } from "@/lib/utils";

interface DataPreviewProps {
  content: string;
  expanded: boolean;
}

/**
 * Preview component for structured data resources (JSON, etc).
 */
export function DataPreview({ content, expanded }: DataPreviewProps) {
  return (
    <div className="relative">
      <pre
        className={cn(
          "text-[0.65rem] bg-slate-900 rounded-md p-3 overflow-x-auto",
          !expanded && "max-h-24 overflow-hidden",
        )}
      >
        <code>{content}</code>
      </pre>
    </div>
  );
}
