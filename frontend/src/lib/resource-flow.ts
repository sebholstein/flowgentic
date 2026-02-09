import type {
  Resource,
  ResourceRef,
  InputRequirement,
  OutputRequirement,
  FulfillmentStatus,
} from "@/types/resource";
import type { Task, Issue } from "@/types/inbox";

/**
 * Resource visibility rules:
 * | Source              | Visible to                    |
 * |---------------------|-------------------------------|
 * | Issue (shared)      | All tasks                     |
 * | Issue (restricted)  | Named tasks only              |
 * | Task output         | Dependent tasks (after done)  |
 */

/**
 * Resolve which resources a task can access based on:
 * - Issue-level shared resources
 * - Issue-level restricted resources (if task is in the list)
 * - Resources produced by completed dependency tasks
 */
export function resolveTaskResourceScope(
  task: Task,
  issue: Issue,
  allTasks: Task[],
  allResources: Resource[],
): ResourceRef[] {
  const availableResources: ResourceRef[] = [];
  const seenIds = new Set<string>();

  // 1. Add issue-level resources that are shared with all tasks
  if (issue.resources) {
    for (const issueResource of issue.resources) {
      if (issueResource.sharedWithAllTasks) {
        if (!seenIds.has(issueResource.resourceId)) {
          availableResources.push({
            resourceId: issueResource.resourceId,
            name: issueResource.name,
            type: issueResource.type,
            status: issueResource.status,
          });
          seenIds.add(issueResource.resourceId);
        }
      } else if (issueResource.restrictedToTaskIds?.includes(task.id)) {
        // Restricted resources available to this specific task
        if (!seenIds.has(issueResource.resourceId)) {
          availableResources.push({
            resourceId: issueResource.resourceId,
            name: issueResource.name,
            type: issueResource.type,
            status: issueResource.status,
          });
          seenIds.add(issueResource.resourceId);
        }
      }
    }
  }

  // 2. Add resources from completed dependency tasks
  for (const depId of task.dependencies) {
    const depTask = allTasks.find((t) => t.id === depId);
    if (depTask && depTask.status === "completed") {
      // Find resources produced by completed executions of this dependency
      const selectedExecution = depTask.executions?.find(
        (e) => e.id === depTask.selectedExecutionId,
      );
      if (selectedExecution?.producedResourceIds) {
        for (const resourceId of selectedExecution.producedResourceIds) {
          if (!seenIds.has(resourceId)) {
            const resource = allResources.find((r) => r.id === resourceId);
            if (resource) {
              availableResources.push({
                resourceId: resource.id,
                name: resource.name,
                type: resource.type,
                status: resource.status,
              });
              seenIds.add(resourceId);
            }
          }
        }
      }
    }
  }

  return availableResources;
}

/**
 * Check if a single input requirement is satisfied by available resources
 */
function checkInputFulfillment(
  requirement: InputRequirement,
  availableResources: ResourceRef[],
): { status: FulfillmentStatus; resourceId?: string; errors?: string[] } {
  // Find matching resources by type
  const matchingResources = availableResources.filter((r) => r.type === requirement.expectedType);

  if (matchingResources.length === 0) {
    return { status: "unfulfilled" };
  }

  // For now, check if any resource has approved status
  // In a full implementation, we'd also validate against the schema
  const approvedResource = matchingResources.find((r) => r.status === "approved");
  if (approvedResource) {
    return {
      status: "fulfilled",
      resourceId: approvedResource.resourceId,
    };
  }

  // Found resources but none are approved
  const pendingResource = matchingResources.find(
    (r) => r.status === "pending" || r.status === "draft",
  );
  if (pendingResource) {
    return {
      status: "partial",
      resourceId: pendingResource.resourceId,
      errors: ["Resource exists but is not yet approved"],
    };
  }

  return { status: "unfulfilled" };
}

/**
 * Validate all input requirements against available resources
 * Returns updated requirements with fulfillment status
 */
export function validateInputRequirements(
  requirements: InputRequirement[],
  availableResources: ResourceRef[],
): InputRequirement[] {
  return requirements.map((req) => {
    const result = checkInputFulfillment(req, availableResources);
    return {
      ...req,
      fulfillmentStatus: result.status,
      fulfilledByResourceId: result.resourceId,
      validationErrors: result.errors,
    };
  });
}

/**
 * Check if a single output requirement is satisfied by produced resources
 */
function checkOutputFulfillment(
  requirement: OutputRequirement,
  producedResources: ResourceRef[],
): { status: FulfillmentStatus; resourceId?: string; errors?: string[] } {
  // Find matching resources by type
  const matchingResources = producedResources.filter((r) => r.type === requirement.expectedType);

  if (matchingResources.length === 0) {
    return { status: "unfulfilled" };
  }

  // Check status based on whether review is required
  if (requirement.requiresReview) {
    const approvedResource = matchingResources.find((r) => r.status === "approved");
    if (approvedResource) {
      return {
        status: "fulfilled",
        resourceId: approvedResource.resourceId,
      };
    }

    const pendingResource = matchingResources.find(
      (r) => r.status === "pending" || r.status === "draft",
    );
    if (pendingResource) {
      return {
        status: "partial",
        resourceId: pendingResource.resourceId,
        errors: ["Resource produced but awaiting review"],
      };
    }
  } else {
    // No review required - any non-rejected resource is fine
    const validResource = matchingResources.find((r) => r.status !== "rejected");
    if (validResource) {
      return {
        status: "fulfilled",
        resourceId: validResource.resourceId,
      };
    }
  }

  return { status: "unfulfilled" };
}

/**
 * Validate all output requirements against produced resources
 * Returns updated requirements with fulfillment status
 */
export function validateOutputRequirements(
  requirements: OutputRequirement[],
  producedResources: ResourceRef[],
): OutputRequirement[] {
  return requirements.map((req) => {
    const result = checkOutputFulfillment(req, producedResources);
    return {
      ...req,
      fulfillmentStatus: result.status,
      fulfilledByResourceId: result.resourceId,
      validationErrors: result.errors,
    };
  });
}

/**
 * Calculate resource completion status for a task
 */
export function calculateResourceCompletionStatus(
  inputRequirements: InputRequirement[],
  outputRequirements: OutputRequirement[],
): {
  inputsFulfilled: number;
  inputsTotal: number;
  outputsFulfilled: number;
  outputsTotal: number;
  allRequirementsMet: boolean;
} {
  const requiredInputs = inputRequirements.filter((r) => r.isRequired);
  const requiredOutputs = outputRequirements.filter((r) => r.isRequired);

  const inputsFulfilled = requiredInputs.filter((r) => r.fulfillmentStatus === "fulfilled").length;
  const outputsFulfilled = requiredOutputs.filter(
    (r) => r.fulfillmentStatus === "fulfilled",
  ).length;

  return {
    inputsFulfilled,
    inputsTotal: requiredInputs.length,
    outputsFulfilled,
    outputsTotal: requiredOutputs.length,
    allRequirementsMet:
      inputsFulfilled === requiredInputs.length && outputsFulfilled === requiredOutputs.length,
  };
}

/**
 * Get a color class based on fulfillment ratio
 */
export function getFulfillmentColor(fulfilled: number, total: number): "green" | "yellow" | "red" {
  if (total === 0) return "green";
  const ratio = fulfilled / total;
  if (ratio >= 1) return "green";
  if (ratio > 0) return "yellow";
  return "red";
}

/**
 * Get icon/badge info for a resource type
 */
export function getResourceTypeInfo(type: Resource["type"]): {
  label: string;
  color: string;
  bgColor: string;
} {
  switch (type) {
    case "markdown":
      return { label: "Markdown", color: "text-blue-400", bgColor: "bg-blue-400/10" };
    case "code":
      return { label: "Code", color: "text-emerald-400", bgColor: "bg-emerald-400/10" };
    case "image":
      return { label: "Image", color: "text-purple-400", bgColor: "bg-purple-400/10" };
    case "pdf":
      return { label: "PDF", color: "text-red-400", bgColor: "bg-red-400/10" };
    case "data":
      return { label: "Data", color: "text-amber-400", bgColor: "bg-amber-400/10" };
    case "external":
      return { label: "External", color: "text-slate-400", bgColor: "bg-slate-400/10" };
    default:
      return { label: "Unknown", color: "text-slate-400", bgColor: "bg-slate-400/10" };
  }
}

/**
 * Get status badge info
 */
export function getResourceStatusInfo(status: Resource["status"]): {
  label: string;
  color: string;
  bgColor: string;
} {
  switch (status) {
    case "draft":
      return { label: "Draft", color: "text-slate-400", bgColor: "bg-slate-400/10" };
    case "pending":
      return { label: "Pending", color: "text-amber-400", bgColor: "bg-amber-400/10" };
    case "approved":
      return { label: "Approved", color: "text-emerald-400", bgColor: "bg-emerald-400/10" };
    case "superseded":
      return { label: "Superseded", color: "text-purple-400", bgColor: "bg-purple-400/10" };
    case "rejected":
      return { label: "Rejected", color: "text-red-400", bgColor: "bg-red-400/10" };
    default:
      return { label: "Unknown", color: "text-slate-400", bgColor: "bg-slate-400/10" };
  }
}
