import { TextContentPartComponent } from "@assistant-ui/react";

export const MarkdownText: TextContentPartComponent = ({ text }) => {
  return <span className="whitespace-pre-wrap">{text}</span>;
};
