import { useState } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Copy, Check } from "lucide-react";
import type { Resource } from "@/types/resource";

interface CodePreviewProps {
  resource: Resource;
  content: string;
  expanded: boolean;
}

/**
 * Preview component for code resources with syntax highlighting and copy functionality.
 */
export function CodePreview({ resource, content, expanded }: CodePreviewProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="relative">
      <Button
        variant="ghost"
        size="sm"
        className="absolute top-1 right-1 h-6 w-6 p-0 opacity-70 hover:opacity-100"
        onClick={handleCopy}
      >
        {copied ? <Check className="size-3 text-emerald-400" /> : <Copy className="size-3" />}
      </Button>
      <pre
        className={cn(
          "text-[0.7rem] bg-slate-900 rounded-md p-3 overflow-x-auto",
          !expanded && "max-h-24 overflow-hidden",
        )}
      >
        <code className={`language-${resource.metadata.language || "text"}`}>{content}</code>
      </pre>
    </div>
  );
}
