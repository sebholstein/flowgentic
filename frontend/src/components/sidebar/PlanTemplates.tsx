import { ScrollArea } from "@/components/ui/scroll-area";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { FileCode, Sparkles, Bot, GitBranch } from "lucide-react";

const templates = [
  {
    id: "1",
    name: "Feature Request",
    description: "Standard template for requesting new features",
    icon: Sparkles,
    tags: ["planning", "feature"],
  },
  {
    id: "2",
    name: "Bug Report",
    description: "Template for reporting and tracking bugs",
    icon: FileCode,
    tags: ["bug", "tracking"],
  },
  {
    id: "3",
    name: "Code Review",
    description: "Template for reviewing code changes",
    icon: GitBranch,
    tags: ["review", "code"],
  },
  {
    id: "4",
    name: "AI Assistant",
    description: "Template with AI-powered planning mode",
    icon: Bot,
    tags: ["ai", "planning"],
  },
];

export function PlanTemplates() {
  return (
    <ScrollArea className="flex-1 overflow-hidden px-2 pt-2">
      <div className="space-y-3 p-2">
        {templates.map((template) => {
          const Icon = template.icon;
          return (
            <Card key={template.id} className="cursor-pointer hover:bg-muted/50 transition-colors">
              <CardHeader className="p-3 pb-2">
                <div className="flex items-start gap-3">
                  <div className="p-2 rounded-md bg-primary/10">
                    <Icon className="size-4 text-primary" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <CardTitle className="text-sm font-medium">{template.name}</CardTitle>
                    <CardDescription className="text-xs mt-0.5">
                      {template.description}
                    </CardDescription>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="p-3 pt-0">
                <div className="flex gap-1 flex-wrap">
                  {template.tags.map((tag) => (
                    <Badge key={tag} variant="secondary" className="text-[0.65rem] px-1.5 py-0">
                      {tag}
                    </Badge>
                  ))}
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </ScrollArea>
  );
}
