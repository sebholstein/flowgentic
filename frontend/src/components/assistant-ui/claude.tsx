"use client";

import {
  ActionBarPrimitive,
  AuiIf,
  AttachmentPrimitive,
  ComposerPrimitive,
  MessagePrimitive,
  ThreadPrimitive,
  useAuiState,
} from "@assistant-ui/react";
import * as Avatar from "@radix-ui/react-avatar";
import {
  ArrowUpIcon,
  ChevronDownIcon,
  Cross2Icon,
  MixerHorizontalIcon,
  Pencil1Icon,
  PlusIcon,
  ReloadIcon,
} from "@radix-ui/react-icons";
import { useEffect, useState, type FC } from "react";
import { useShallow } from "zustand/shallow";
import { MarkdownText } from "./markdown-text";
import { ScrollArea } from "../ui/scroll-area";

export const Claude: FC = () => {
  return (
    <ThreadPrimitive.Root className="flex h-full flex-col items-stretch p-4 pt-16">
      <ScrollArea className="flex grow flex-col">
        <ThreadPrimitive.Messages components={{ Message: ChatMessage }} />
        <div aria-hidden="true" className="h-4" />
      </ScrollArea>
      <ComposerPrimitive.Root className="mx-4 flex w-auto flex-col rounded-2xl border border-border bg-card p-0.5 shadow-lg transition-shadow duration-200 focus-within:shadow-xl hover:shadow-xl">
        <div className="m-3.5 flex flex-col gap-3.5">
          <div className="relative">
            <div className="wrap-break-word max-h-96 w-full overflow-y-auto">
              <ComposerPrimitive.Input
                placeholder="How can I help you today?"
                autoFocus
                className="block min-h-6 w-full resize-none bg-transparent text-foreground outline-none placeholder:text-muted-foreground"
              />
            </div>
          </div>
          <div className="flex w-full items-center gap-2">
            <div className="relative flex min-w-0 flex-1 shrink items-center gap-2">
              <ComposerPrimitive.AddAttachment className="flex h-8 min-w-8 items-center justify-center overflow-hidden rounded-lg border border-border bg-transparent px-1.5 text-muted-foreground transition-all hover:bg-muted hover:text-foreground active:scale-[0.98]">
                <PlusIcon width={16} height={16} />
              </ComposerPrimitive.AddAttachment>
              <button
                type="button"
                className="flex h-8 min-w-8 items-center justify-center overflow-hidden rounded-lg border border-border bg-transparent px-1.5 text-muted-foreground transition-all hover:bg-muted hover:text-foreground active:scale-[0.98]"
                aria-label="Open tools menu"
              >
                <MixerHorizontalIcon width={16} height={16} />
              </button>
              <button
                type="button"
                className="flex h-8 min-w-8 shrink-0 items-center justify-center overflow-hidden rounded-lg border border-border bg-transparent px-1.5 text-muted-foreground transition-all hover:bg-muted hover:text-foreground active:scale-[0.98]"
                aria-label="Extended thinking"
              >
                <ReloadIcon width={16} height={16} />
              </button>
            </div>
            <button
              type="button"
              className="flex h-8 min-w-16 items-center justify-center gap-1 whitespace-nowrap rounded-md px-2 pr-2 pl-2.5 text-foreground text-xs transition duration-300 ease-[cubic-bezier(0.165,0.85,0.45,1)] hover:bg-muted active:scale-[0.985]"
            >
              <span className="text-[14px]">Sonnet 4.5</span>
              <ChevronDownIcon width={20} height={20} className="opacity-75" />
            </button>
            <ComposerPrimitive.Send className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary transition-colors hover:bg-primary/90 active:scale-95 disabled:pointer-events-none disabled:opacity-50">
              <ArrowUpIcon width={16} height={16} className="text-primary-foreground" />
            </ComposerPrimitive.Send>
          </div>
        </div>
        <AuiIf condition={(s) => s.composer.attachments.length > 0}>
          <div className="overflow-hidden rounded-b-2xl">
            <div className="overflow-x-auto rounded-b-2xl border-t border-border bg-muted p-3.5">
              <div className="flex flex-row gap-3">
                <ComposerPrimitive.Attachments components={{ Attachment: ClaudeAttachment }} />
              </div>
            </div>
          </div>
        </AuiIf>
      </ComposerPrimitive.Root>
    </ThreadPrimitive.Root>
  );
};

const ChatMessage: FC = () => {
  return (
    <MessagePrimitive.Root className="group relative mx-auto mt-1 mb-1 block w-full max-w-3xl">
      <AuiIf condition={({ message }) => message.role === "user"}>
        <div className="group/user wrap-break-word relative inline-flex max-w-[75ch] flex-col gap-2 rounded-xl bg-muted py-2.5 pr-6 pl-2.5 text-foreground transition-all">
          <div className="relative flex flex-row gap-2">
            <div className="shrink-0 self-start transition-all duration-300">
              <Avatar.Root className="flex h-7 w-7 shrink-0 select-none items-center justify-center rounded-full bg-primary font-bold text-[12px] text-primary-foreground">
                <Avatar.AvatarFallback>U</Avatar.AvatarFallback>
              </Avatar.Root>
            </div>
            <div className="flex-1">
              <div className="relative grid grid-cols-1 gap-2 py-0.5">
                <div className="wrap-break-word whitespace-pre-wrap font-mono text-sm">
                  <MessagePrimitive.Parts components={{ Text: MarkdownText }} />
                </div>
              </div>
            </div>
          </div>
          <div className="pointer-events-none absolute right-2 bottom-0">
            <ActionBarPrimitive.Root
              autohide="not-last"
              className="pointer-events-auto min-w-max translate-x-1 translate-y-4 rounded-lg border border-border bg-background/80 p-0.5 opacity-0 shadow-sm backdrop-blur-sm transition group-hover/user:translate-x-0.5 group-hover/user:opacity-100"
            >
              <div className="flex items-center text-muted-foreground">
                <ActionBarPrimitive.Reload className="flex h-8 w-8 items-center justify-center rounded-md transition duration-300 ease-[cubic-bezier(0.165,0.85,0.45,1)] hover:bg-muted active:scale-95">
                  <ReloadIcon width={20} height={20} />
                </ActionBarPrimitive.Reload>
                <ActionBarPrimitive.Edit className="flex h-8 w-8 items-center justify-center rounded-md transition duration-300 ease-[cubic-bezier(0.165,0.85,0.45,1)] hover:bg-muted active:scale-95">
                  <Pencil1Icon width={20} height={20} />
                </ActionBarPrimitive.Edit>
              </div>
            </ActionBarPrimitive.Root>
          </div>
        </div>
      </AuiIf>

      <AuiIf condition={({ message }) => message.role === "assistant"}>
        <div className="relative mb-4">
          <div className="relative leading-[1.65rem]">
            <div className="grid grid-cols-1 gap-2.5">
              <div className="wrap-break-word whitespace-normal pr-8 pl-2 font-mono text-sm text-foreground">
                <MessagePrimitive.Parts components={{ Text: MarkdownText }} />
              </div>
            </div>
          </div>
        </div>
      </AuiIf>
    </MessagePrimitive.Root>
  );
};

const useFileSrc = (file: File | undefined) => {
  const [src, setSrc] = useState<string | undefined>(undefined);

  useEffect(() => {
    if (!file) {
      setSrc(undefined);
      return;
    }

    const objectUrl = URL.createObjectURL(file);
    setSrc(objectUrl);

    return () => {
      URL.revokeObjectURL(objectUrl);
    };
  }, [file]);

  return src;
};

const useAttachmentSrc = () => {
  const { file, src } = useAuiState(
    useShallow(({ attachment }): { file?: File; src?: string } => {
      if (attachment.type !== "image") return {};
      if (attachment.file) return { file: attachment.file };
      const src = attachment.content?.filter((c) => c.type === "image")[0]?.image;
      if (!src) return {};
      return { src };
    }),
  );

  return useFileSrc(file) ?? src;
};

const ClaudeAttachment: FC = () => {
  const isImage = useAuiState(({ attachment }) => attachment.type === "image");
  const src = useAttachmentSrc();

  return (
    <AttachmentPrimitive.Root className="group/thumbnail relative">
      <div
        className="can-focus-within overflow-hidden rounded-lg border border-border shadow-sm hover:border-border hover:shadow-md"
        style={{
          width: "120px",
          height: "120px",
          minWidth: "120px",
          minHeight: "120px",
        }}
      >
        <button
          type="button"
          className="relative bg-card"
          style={{ width: "120px", height: "120px" }}
        >
          {isImage && src ? (
            <img
              className="h-full w-full object-cover opacity-100 transition duration-400"
              alt="Attachment"
              src={src}
            />
          ) : (
            <div className="flex h-full w-full items-center justify-center text-muted-foreground">
              <AttachmentPrimitive.unstable_Thumb className="text-xs" />
            </div>
          )}
        </button>
      </div>
      <AttachmentPrimitive.Remove
        className="absolute -top-2 -left-2 flex h-5 w-5 items-center justify-center rounded-full border border-border bg-background/90 text-muted-foreground opacity-0 backdrop-blur-sm transition-all hover:bg-background hover:text-foreground group-focus-within/thumbnail:opacity-100 group-hover/thumbnail:opacity-100"
        aria-label="Remove attachment"
      >
        <Cross2Icon width={12} height={12} />
      </AttachmentPrimitive.Remove>
    </AttachmentPrimitive.Root>
  );
};
