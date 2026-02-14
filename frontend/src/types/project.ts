export interface Project {
  id: string;
  name: string;
  description?: string;
  color?: string; // For sidebar visual distinction
  defaultPlannerAgent?: string;
  defaultPlannerModel?: string;
  embeddedWorkerPath?: string;
  workerPaths?: Record<string, string>;
  agentPlanningTaskPreferences?: string;
  sortIndex?: number;
}
