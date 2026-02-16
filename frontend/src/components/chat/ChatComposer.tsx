import { useState, useRef } from "react";
import { AnimatePresence, motion } from "motion/react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Loader2, ImagePlus, ArrowUp, ChevronDown, Map } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuLabel,
} from "@/components/ui/dropdown-menu";
import type { ModelInfo } from "@/proto/gen/worker/v1/system_service_pb";

export function ChatComposer({
  onSend,
  isTyping,
  selectedModel,
  availableModels,
  onModelChange,
  sessionMode = "code",
  onSessionModeChange,
}: {
  onSend: (message: string) => void;
  isTyping: boolean;
  selectedModel?: string;
  availableModels?: ModelInfo[];
  onModelChange?: (model: string) => void;
  sessionMode?: string;
  onSessionModeChange?: (mode: string) => void;
}) {
  const [inputValue, setInputValue] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleSend = () => {
    if (!inputValue.trim()) return;
    onSend(inputValue.trim());
    setInputValue("");
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Tab" && e.shiftKey && onSessionModeChange) {
      e.preventDefault();
      onSessionModeChange(sessionMode === "code" ? "plan" : "code");
      return;
    }
    if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleImageClick = () => fileInputRef.current?.click();

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (files && files.length > 0) {
      console.log("Selected files:", files);
      e.target.value = "";
    }
  };

  return (
    <div className="px-6 lg:px-10 pb-3 pt-2">
      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        onChange={handleFileChange}
        className="hidden"
        multiple
      />

      <div className="relative rounded-xl border border-input bg-muted/30 focus-within:ring-1 focus-within:ring-ring/50 focus-within:border-ring/50">
        <textarea
          ref={textareaRef}
          placeholder={`How can I help you today?`}
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={isTyping}
          rows={4}
          className="w-full resize-none bg-transparent px-4 pt-3 pb-10 text-sm min-h-[120px] placeholder:text-muted-foreground/50 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
        />
        <div className="absolute bottom-2 left-2 right-2 flex items-center justify-between">
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 rounded-lg"
              onClick={handleImageClick}
              disabled={isTyping}
              title="Attach image"
            >
              <ImagePlus className="h-3.5 w-3.5 text-muted-foreground" />
            </Button>
          </div>
          <div className="flex items-center gap-1">
            {availableModels && availableModels.length > 0 && onModelChange && (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <button
                    type="button"
                    className="flex items-center gap-0.5 rounded-md px-1.5 h-7 text-[11px] text-muted-foreground hover:text-foreground hover:bg-accent transition-colors cursor-pointer"
                  >
                    <span className="max-w-[120px] truncate">
                      {(availableModels.find((m) => m.id === selectedModel)?.displayName || selectedModel || "Model").replace(/\s*\(recommended\)/i, "")}
                    </span>
                    <ChevronDown className="size-3 shrink-0 opacity-60" />
                  </button>
                </DropdownMenuTrigger>
                <DropdownMenuContent side="top" align="end" className="min-w-[180px]">
                  <DropdownMenuLabel>Model</DropdownMenuLabel>
                  <DropdownMenuRadioGroup value={selectedModel} onValueChange={onModelChange}>
                    {availableModels.map((model) => (
                      <DropdownMenuRadioItem key={model.id} value={model.id}>
                        <div className="flex flex-col">
                          <span>{(model.displayName || model.id).replace(/\s*\(recommended\)/i, "")}</span>
                          {model.description && (
                            <span className="text-[10px] text-muted-foreground">{model.description}</span>
                          )}
                        </div>
                      </DropdownMenuRadioItem>
                    ))}
                  </DropdownMenuRadioGroup>
                </DropdownMenuContent>
              </DropdownMenu>
            )}

            {onSessionModeChange && (
              <button
                type="button"
                onClick={() => onSessionModeChange(sessionMode === "code" ? "plan" : "code")}
                className={cn(
                  "flex items-center gap-1 rounded-md px-1.5 h-7 text-[11px] transition-colors cursor-pointer",
                  sessionMode === "plan"
                    ? "text-amber-400 bg-amber-500/10 hover:bg-amber-500/20"
                    : "text-muted-foreground hover:text-foreground hover:bg-accent",
                )}
                title="Toggle plan mode (Shift+Tab)"
              >
                <Map className="size-3 shrink-0" />
                <AnimatePresence>
                  {sessionMode === "plan" && (
                    <motion.span
                      initial={{ width: 0, opacity: 0 }}
                      animate={{ width: "auto", opacity: 1 }}
                      exit={{ width: 0, opacity: 0 }}
                      transition={{ duration: 0.2, ease: "easeInOut" }}
                      className="overflow-hidden whitespace-nowrap"
                    >
                      Plan mode active
                    </motion.span>
                  )}
                </AnimatePresence>
              </button>
            )}
            <Button
              size="icon"
              className="h-7 w-7 rounded-lg"
              onClick={handleSend}
              disabled={!inputValue.trim() || isTyping}
            >
              {isTyping ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <ArrowUp className="h-3.5 w-3.5" />
              )}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
