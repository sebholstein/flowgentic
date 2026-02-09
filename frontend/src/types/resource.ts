// Resource types
export type ResourceType =
  | "markdown" // Docs, specs, notes
  | "code" // Source code files
  | "image" // PNG, JPG, SVG
  | "pdf" // PDF documents
  | "data" // JSON, CSV
  | "external"; // URL reference

export type ResourceStatus =
  | "draft" // Work in progress
  | "pending" // Awaiting review
  | "approved" // Ready for use
  | "superseded" // Replaced by newer version
  | "rejected"; // Failed validation

// Content storage (file refs or URLs, no inline)
export type ContentStorage =
  | { type: "file"; path: string; hash?: string }
  | { type: "url"; url: string };

// Who created this resource
export interface ResourceProvenance {
  createdBy:
    | { type: "user"; userId: string; userName: string }
    | { type: "execution"; taskId: string; executionId: string; agentId: string }
    | { type: "system"; source: string };
  createdAt: string;
}

// Core Resource interface
export interface Resource {
  id: string;
  name: string;
  description?: string;
  type: ResourceType;
  status: ResourceStatus;

  storage: ContentStorage;
  metadata: {
    mimeType?: string;
    size?: number;
    language?: string; // For code
    checksum?: string;
  };

  provenance: ResourceProvenance;
  version: number; // Simple version counter, no history

  // Scope
  threadId: string;
  taskId?: string; // If scoped to specific task

  // Assignment (user or agent)
  assignedTo?: {
    type: "user" | "agent";
    id: string;
    name: string;
  };

  // Timestamps
  createdAt: string;
  updatedAt: string;
  tags?: string[];
}

// Lightweight reference
export interface ResourceRef {
  resourceId: string;
  name: string;
  type: ResourceType;
  status: ResourceStatus;
}

// Validation schema types
export type ValidationSchema =
  | { type: "json-schema"; schema: object }
  | { type: "required-sections"; sections: string[] } // For markdown
  | { type: "file-type"; mimeTypes: string[] };

// Fulfillment status for requirements
export type FulfillmentStatus =
  | "unfulfilled" // No matching resource
  | "partial" // Exists but doesn't meet criteria
  | "fulfilled"; // Meets all criteria

// Base resource requirement
export interface ResourceRequirement {
  id: string;
  name: string;
  description: string;
  expectedType: ResourceType;
  isRequired: boolean;

  // Validation (optional - presence check is default)
  validationSchema?: ValidationSchema; // Enable content validation
  acceptanceCriteria?: string[]; // Human-readable checklist

  // Fulfillment tracking
  fulfillmentStatus: FulfillmentStatus;
  fulfilledByResourceId?: string;
  validationErrors?: string[]; // Populated if schema validation fails
}

// Input: what a task needs
export interface InputRequirement extends ResourceRequirement {
  direction: "input";
  sourceType: "inherited" | "dependency" | "any" | "manual";
  sourceDependencyId?: string;
}

// Output: what a task produces
export interface OutputRequirement extends ResourceRequirement {
  direction: "output";
  autoCompleteTask: boolean;
  requiresReview: boolean;
}

// Thread resource - a resource available at the thread level
export interface ThreadResource extends ResourceRef {
  sharedWithAllTasks: boolean;
  restrictedToTaskIds?: string[]; // If not shared, which tasks can see it
}

// Legacy alias for backwards compatibility
export type IssueResource = ThreadResource;

// Resource completion status for a task
export interface ResourceCompletionStatus {
  inputsFulfilled: number;
  inputsTotal: number;
  outputsFulfilled: number;
  outputsTotal: number;
  allRequirementsMet: boolean;
}
