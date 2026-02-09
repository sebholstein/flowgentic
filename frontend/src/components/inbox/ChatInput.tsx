import { useState, useRef, useCallback, useEffect, type KeyboardEvent } from "react";
import { Send } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
  Command,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
} from "@/components/ui/command";
import { IconFile, IconFolder } from "@tabler/icons-react";

// Mock file list - replace with actual file fetching later
const MOCK_FILES = [
  { path: "src/index.ts", type: "file" as const },
  { path: "src/components/Button.tsx", type: "file" as const },
  { path: "src/components/Input.tsx", type: "file" as const },
  { path: "src/components/Modal.tsx", type: "file" as const },
  { path: "src/hooks/useAuth.ts", type: "file" as const },
  { path: "src/hooks/useForm.ts", type: "file" as const },
  { path: "src/utils/api.ts", type: "file" as const },
  { path: "src/utils/helpers.ts", type: "file" as const },
  { path: "src/types/index.ts", type: "file" as const },
  { path: "package.json", type: "file" as const },
  { path: "tsconfig.json", type: "file" as const },
  { path: "README.md", type: "file" as const },
  { path: "src/components", type: "folder" as const },
  { path: "src/hooks", type: "folder" as const },
  { path: "src/utils", type: "folder" as const },
];

type FileItem = (typeof MOCK_FILES)[number];

// Simple fuzzy match
function fuzzyMatch(query: string, text: string): boolean {
  const lowerQuery = query.toLowerCase();
  const lowerText = text.toLowerCase();
  let queryIdx = 0;
  for (let i = 0; i < lowerText.length && queryIdx < lowerQuery.length; i++) {
    if (lowerText[i] === lowerQuery[queryIdx]) queryIdx++;
  }
  return queryIdx === lowerQuery.length;
}

interface ChatInputProps {
  onSend: (message: string) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
}

export function ChatInput({
  onSend,
  placeholder = "Type a message... Use @ to mention files",
  disabled = false,
  className,
}: ChatInputProps) {
  const [message, setMessage] = useState("");
  const [isSending, setIsSending] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  // Mention state
  const [mentionOpen, setMentionOpen] = useState(false);
  const [mentionQuery, setMentionQuery] = useState("");
  const [mentionTriggerIndex, setMentionTriggerIndex] = useState<number | null>(null);
  const [selectedIndex, setSelectedIndex] = useState(0);

  const filteredFiles = mentionQuery
    ? MOCK_FILES.filter((f) => fuzzyMatch(mentionQuery, f.path))
    : MOCK_FILES;

  // Reset selection when filtered results change
  useEffect(() => {
    setSelectedIndex(0);
  }, [filteredFiles.length]);

  // Scroll selected item into view
  useEffect(() => {
    if (!listRef.current || !mentionOpen) return;
    const selectedItem = listRef.current.querySelector(`[data-index="${selectedIndex}"]`);
    selectedItem?.scrollIntoView({ block: "nearest" });
  }, [selectedIndex, mentionOpen]);

  const closeMention = useCallback(() => {
    setMentionOpen(false);
    setMentionQuery("");
    setMentionTriggerIndex(null);
    setSelectedIndex(0);
  }, []);

  const selectFile = useCallback(
    (file: FileItem) => {
      const textarea = textareaRef.current;
      if (!textarea || mentionTriggerIndex === null) return;

      const cursorPos = textarea.selectionStart;
      const textBeforeTrigger = message.slice(0, mentionTriggerIndex);
      const textAfterCursor = message.slice(cursorPos);

      const newMessage = `${textBeforeTrigger}@${file.path} ${textAfterCursor}`;
      setMessage(newMessage);
      closeMention();

      const newCursorPos = mentionTriggerIndex + file.path.length + 2;
      requestAnimationFrame(() => {
        textarea.focus();
        textarea.setSelectionRange(newCursorPos, newCursorPos);
      });
    },
    [message, mentionTriggerIndex, closeMention],
  );

  const adjustTextareaHeight = useCallback(() => {
    const textarea = textareaRef.current;
    if (textarea) {
      textarea.style.height = "auto";
      textarea.style.height = `${Math.min(textarea.scrollHeight, 320)}px`;
    }
  }, []);

  const handleSend = useCallback(() => {
    const trimmed = message.trim();
    if (!trimmed || disabled || isSending) return;

    setIsSending(true);
    onSend(trimmed);
    setMessage("");
    closeMention();

    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
    }

    setIsSending(false);
  }, [message, disabled, isSending, onSend, closeMention]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      if (mentionOpen) {
        switch (e.key) {
          case "ArrowUp":
            e.preventDefault();
            setSelectedIndex((prev) => (prev <= 0 ? filteredFiles.length - 1 : prev - 1));
            return;
          case "ArrowDown":
            e.preventDefault();
            setSelectedIndex((prev) => (prev >= filteredFiles.length - 1 ? 0 : prev + 1));
            return;
          case "Enter":
          case "Tab":
            e.preventDefault();
            if (filteredFiles[selectedIndex]) {
              selectFile(filteredFiles[selectedIndex]);
            }
            return;
          case "Escape":
            e.preventDefault();
            closeMention();
            return;
        }
      }

      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        handleSend();
      }
    },
    [mentionOpen, filteredFiles, selectedIndex, selectFile, closeMention, handleSend],
  );

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      const newValue = e.target.value;
      const cursorPosition = e.target.selectionStart;

      setMessage(newValue);
      adjustTextareaHeight();

      // Check for @ mention trigger
      const textBeforeCursor = newValue.slice(0, cursorPosition);
      const lastAtIndex = textBeforeCursor.lastIndexOf("@");

      if (lastAtIndex === -1) {
        if (mentionOpen) closeMention();
        return;
      }

      // Check if @ is at start or after whitespace
      const charBeforeAt = lastAtIndex > 0 ? textBeforeCursor[lastAtIndex - 1] : " ";
      const isValidTrigger = /\s/.test(charBeforeAt) || lastAtIndex === 0;

      if (!isValidTrigger) {
        if (mentionOpen) closeMention();
        return;
      }

      // Get the query after @
      const queryText = textBeforeCursor.slice(lastAtIndex + 1);

      // If there's a space in the query, close the mention
      if (queryText.includes(" ")) {
        if (mentionOpen) closeMention();
        return;
      }

      // Open or update mention
      if (!mentionOpen) {
        setMentionOpen(true);
        setMentionTriggerIndex(lastAtIndex);
      }
      setMentionQuery(queryText);
    },
    [adjustTextareaHeight, mentionOpen, closeMention],
  );

  const isDisabled = disabled || isSending;
  const canSend = message.trim().length > 0 && !isDisabled;

  return (
    <div
      className={cn("relative flex items-end gap-2 border-t bg-background px-4 py-3", className)}
    >
      {/* Mention popup */}
      {mentionOpen && (
        <div className="absolute bottom-full left-4 right-16 mb-1 z-50 rounded-lg border bg-popover shadow-lg">
          <Command shouldFilter={false}>
            <CommandList ref={listRef} className="max-h-48">
              <CommandEmpty className="py-3 text-xs text-muted-foreground">
                No files found
              </CommandEmpty>
              <CommandGroup heading={mentionQuery ? `Files matching "${mentionQuery}"` : "Files"}>
                {filteredFiles.map((file, index) => (
                  <CommandItem
                    key={file.path}
                    data-index={index}
                    onSelect={() => selectFile(file)}
                    className={cn(index === selectedIndex && "bg-accent")}
                  >
                    {file.type === "folder" ? (
                      <IconFolder className="size-4 text-muted-foreground" />
                    ) : (
                      <IconFile className="size-4 text-muted-foreground" />
                    )}
                    <span className="truncate">{file.path}</span>
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </div>
      )}

      <Textarea
        ref={textareaRef}
        value={message}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        disabled={isDisabled}
        rows={1}
        autoComplete="off"
        className="min-h-[36px] max-h-[320px] text-sm flex-1 [field-sizing:fixed] overflow-y-auto"
      />
      <Button size="icon" onClick={handleSend} disabled={!canSend} className="shrink-0 size-9">
        <Send className="size-4" />
      </Button>
    </div>
  );
}
