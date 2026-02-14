import { codeToHtml } from "shiki";
import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import { cn } from "@/lib/utils";
import { memo, useEffect, useState } from "react";

interface MarkdownProps {
  children: string;
  className?: string;
}

const CodeBlock = memo(function CodeBlock({
  className,
  children,
}: {
  className?: string;
  children: string;
}) {
  const [html, setHtml] = useState<string | null>(null);
  const match = /language-(\w+)/.exec(className || "");
  const lang = match?.[1] || "text";

  useEffect(() => {
    let cancelled = false;
    codeToHtml(children, { lang, theme: "one-dark-pro" }).then((result) => {
      if (!cancelled) setHtml(result);
    });
    return () => {
      cancelled = true;
    };
  }, [children, lang]);

  if (!match) {
    return (
      <pre className="bg-zinc-800 text-xs p-3 rounded-lg overflow-x-auto">
        <code>{children}</code>
      </pre>
    );
  }

  return html ? (
    // biome-ignore lint/security/noDangerouslySetInnerHtml: Shiki generates safe HTML for syntax highlighting
    <div className="rounded-lg text-xs overflow-hidden [&_pre]:p-3 [&_pre]:overflow-x-auto" dangerouslySetInnerHTML={{ __html: html }} />
  ) : (
    <pre>
      <code>{children}</code>
    </pre>
  );
});

const components: Components = {
  pre({ children }) {
    return <div className="not-prose">{children}</div>;
  },
  code({ className, children }) {
    const hasLanguage = className?.includes("language-");
    const isMultiline = typeof children === "string" && children.includes("\n");
    if (!hasLanguage && !isMultiline) {
      return <code className={className}>{children}</code>;
    }
    return (
      <CodeBlock className={className}>
        {String(children).replace(/\n$/, "")}
      </CodeBlock>
    );
  },
};

export function Markdown({ children, className }: MarkdownProps) {
  return (
    <div
      className={cn(
        "prose prose-xs dark:prose-invert max-w-none text-sm",
        "prose-p:my-1.5 prose-p:leading-relaxed prose-p:text-sm",
        "prose-headings:font-semibold prose-headings:tracking-tight",
        "prose-h1:text-base prose-h2:text-sm prose-h3:text-sm",
        "prose-ul:my-1.5 prose-ol:my-1.5 prose-li:my-0.5 prose-li:text-sm",
        "prose-code:bg-zinc-200 dark:prose-code:bg-zinc-700 prose-code:px-1.5 prose-code:py-0.5 prose-code:rounded prose-code:text-xs prose-code:font-mono prose-code:before:content-none prose-code:after:content-none",
        "prose-pre:bg-background/50 prose-pre:p-2 prose-pre:rounded-lg prose-pre:text-xs prose-pre:my-1",
        "prose-p:first:mt-0 prose-p:last:mb-0",
        "prose-a:text-blue-500 dark:prose-a:text-blue-400 prose-a:no-underline hover:prose-a:underline",
        "prose-strong:text-foreground prose-strong:font-semibold",
        "prose-blockquote:border-l-2 prose-blockquote:border-border prose-blockquote:pl-3 prose-blockquote:italic prose-blockquote:text-sm",
        "prose-hr:my-2",
        "prose-table:text-xs prose-th:border prose-th:border-border prose-th:px-2 prose-th:py-1 prose-td:border prose-td:border-border prose-td:px-2 prose-td:py-1",
        className,
      )}
    >
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {children}
      </ReactMarkdown>
    </div>
  );
}
