import type { InboxItem, TaskExecution } from "@/types/inbox";
import type { PlanningData } from "@/components/inbox/PlanningApproval";
import type { ThreadReviewData } from "@/components/inbox/ThreadReview";

/**
 * Mock inbox items for development/demo purposes.
 * Includes various inbox item types: execution selections, questionnaires,
 * planning approvals, decision escalations, and direction clarifications.
 */
export const inboxItems: InboxItem[] = [
  {
    id: "1",
    type: "execution_selection",
    status: "pending",
    priority: "high",
    title: "Choose implementation",
    description: "Two agents completed the task with different approaches",
    createdAt: "5m ago",
    source: "thread_overseer",
    sourceName: "Alex Chen",
    threadId: "1",
    threadName: "User Authentication Flow",
    taskId: "t5",
    taskName: "Build Login UI",
    executionIds: ["exec-1", "exec-2"],
    overseerMessages: [
      {
        id: "m1",
        role: "overseer",
        content: "Requesting implementations from multiple agents for comparison...",
        timestamp: "10:30 AM",
      },
      {
        id: "m2",
        role: "agent",
        agentId: "claude-opus-4",
        executionId: "exec-1",
        content:
          "Here's my implementation using React Hook Form with built-in validation. The approach prioritizes type safety and minimal bundle size.",
        timestamp: "10:31 AM",
      },
      {
        id: "m3",
        role: "agent",
        agentId: "gpt-4",
        executionId: "exec-2",
        content:
          "I've created an implementation using Formik with Yup validation. This approach provides a declarative validation schema that's easy to maintain.",
        timestamp: "10:32 AM",
      },
      {
        id: "m4",
        role: "overseer",
        content:
          "Selected claude-opus-4 variant for better type safety and bundle size. React Hook Form has 94% smaller bundle than Formik.",
        timestamp: "10:33 AM",
      },
    ],
  },
  {
    id: "2",
    type: "thread_review",
    status: "pending",
    priority: "medium",
    title: "Review thread plan",
    description: "Auth Flow thread requires approval before execution",
    createdAt: "1h ago",
    source: "project_overseer",
    sourceName: "Project Coordinator",
    threadId: "1",
    threadName: "User Authentication Flow",
  },
  {
    id: "3",
    type: "execution_selection",
    status: "resolved",
    priority: "medium",
    title: "Choose implementation",
    description: "Single execution completed successfully",
    createdAt: "2h ago",
    source: "thread_overseer",
    sourceName: "Alex Chen",
    threadId: "1",
    threadName: "User Authentication Flow",
    taskId: "t2",
    taskName: "Create User Model",
    executionIds: ["exec-3"],
    selectedExecutionId: "exec-3",
    decidedBy: "overseer",
  },
  {
    id: "4",
    type: "planning_approval",
    status: "pending",
    priority: "low",
    title: "Approve task breakdown",
    description: "Dashboard Analytics thread planning complete",
    createdAt: "1d ago",
    source: "thread_overseer",
    sourceName: "Jordan Lee",
    threadId: "2",
    threadName: "Dashboard Analytics",
  },
  {
    id: "5",
    type: "questionnaire",
    status: "pending",
    priority: "high",
    title: "Authentication approach",
    description: "Choose how users should authenticate",
    createdAt: "2m ago",
    source: "thread_overseer",
    sourceName: "Alex Chen",
    threadId: "1",
    threadName: "User Authentication Flow",
    overseerMessages: [
      {
        id: "qm1",
        role: "overseer",
        content:
          "I'm planning the authentication system for your app. Before I proceed, I need some decisions from you.",
        timestamp: "2m ago",
      },
      {
        id: "qm2",
        role: "agent",
        agentId: "claude-opus-4",
        content:
          "Based on the project requirements, I'd recommend JWT for the API-first architecture you're building.",
        timestamp: "1m ago",
      },
    ],
    questions: [
      {
        id: "q1",
        header: "Auth method",
        question: "Which authentication method should we use?",
        multiSelect: false,
        options: [
          {
            id: "jwt",
            label: "JWT tokens (Recommended)",
            description:
              "Stateless authentication with JSON Web Tokens. Good for scalability and API access.",
          },
          {
            id: "session",
            label: "Session-based",
            description: "Traditional server-side sessions. Simpler but requires session storage.",
          },
          {
            id: "oauth",
            label: "OAuth 2.0 only",
            description: "Delegate authentication to external providers like Google or GitHub.",
          },
        ],
      },
      {
        id: "q2",
        header: "Social login",
        question: "Which social login providers should we support?",
        multiSelect: true,
        options: [
          {
            id: "google",
            label: "Google",
            description: "Most widely used social login option.",
          },
          {
            id: "github",
            label: "GitHub",
            description: "Popular with developer-focused applications.",
          },
          {
            id: "microsoft",
            label: "Microsoft",
            description: "Common in enterprise environments.",
          },
          {
            id: "none",
            label: "None",
            description: "Only support email/password authentication.",
          },
        ],
      },
    ],
  },
  {
    id: "6",
    type: "questionnaire",
    status: "pending",
    priority: "medium",
    title: "Database preference",
    description: "Select database technology for the project",
    createdAt: "15m ago",
    source: "project_overseer",
    sourceName: "Project Coordinator",
    threadId: "1",
    threadName: "User Authentication Flow",
    overseerMessages: [
      {
        id: "dm1",
        role: "overseer",
        content: "Setting up the data layer. Which database should we use?",
        timestamp: "15m ago",
      },
    ],
    questions: [
      {
        id: "q1",
        header: "Database",
        question: "Which database should we use for this project?",
        multiSelect: false,
        options: [
          {
            id: "postgres",
            label: "PostgreSQL (Recommended)",
            description:
              "Robust relational database with excellent JSON support and strong ecosystem.",
          },
          {
            id: "mysql",
            label: "MySQL",
            description: "Popular relational database, good for traditional web applications.",
          },
          {
            id: "mongodb",
            label: "MongoDB",
            description: "Document database, flexible schema for rapid iteration.",
          },
          {
            id: "sqlite",
            label: "SQLite",
            description: "Embedded database, perfect for development and small deployments.",
          },
        ],
      },
    ],
  },
  {
    id: "7",
    type: "questionnaire",
    status: "resolved",
    priority: "low",
    title: "Styling framework",
    description: "CSS approach for the frontend",
    createdAt: "3h ago",
    source: "task_agent",
    sourceName: "Claude Opus",
    threadId: "1",
    threadName: "User Authentication Flow",
    taskId: "t1",
    taskName: "Setup Frontend",
    questions: [
      {
        id: "q1",
        header: "CSS approach",
        question: "Which styling approach should we use?",
        multiSelect: false,
        selectedOptionIds: ["tailwind"],
        options: [
          {
            id: "tailwind",
            label: "Tailwind CSS",
            description: "Utility-first CSS framework for rapid UI development.",
          },
          {
            id: "css-modules",
            label: "CSS Modules",
            description: "Scoped CSS with standard syntax, good for component isolation.",
          },
          {
            id: "styled",
            label: "Styled Components",
            description: "CSS-in-JS solution with dynamic styling capabilities.",
          },
        ],
      },
    ],
  },
  {
    id: "8",
    type: "decision_escalation",
    status: "pending",
    priority: "high",
    title: "API versioning strategy",
    description: "Blocking decision needed before continuing",
    createdAt: "3m ago",
    source: "task_agent",
    sourceName: "Claude Opus",
    threadId: "1",
    threadName: "User Authentication Flow",
    taskId: "t3",
    taskName: "Implement API Endpoints",
    overseerMessages: [
      {
        id: "de1",
        role: "agent",
        agentId: "claude-opus-4",
        content:
          "I'm implementing the API endpoints and need to decide on a versioning strategy. This is a blocking decision that will affect all future API development.",
        timestamp: "3m ago",
      },
    ],
    decisionContext:
      "The API will be consumed by both the web frontend and potentially mobile apps in the future. We need to choose a versioning strategy that allows for backwards-compatible changes while enabling breaking changes when necessary.",
    decisionOptions: [
      {
        id: "url-versioning",
        label: "URL Path Versioning",
        description: "Version in the URL path, e.g., /api/v1/users",
        recommended: true,
        benefits: [
          "Clear and explicit versioning",
          "Easy to route different versions",
          "Good for documentation",
        ],
        risks: ["URLs change between versions", "Can lead to code duplication"],
      },
      {
        id: "header-versioning",
        label: "Header Versioning",
        description: "Version specified in request headers",
        benefits: ["Clean URLs", "Flexible version negotiation"],
        risks: [
          "Less discoverable",
          "Harder to test in browser",
          "Can be confusing for API consumers",
        ],
      },
      {
        id: "query-versioning",
        label: "Query Parameter Versioning",
        description: "Version as query param, e.g., /api/users?version=1",
        benefits: ["Easy to add to existing APIs", "Simple to implement"],
        risks: ["Can be accidentally omitted", "Clutters the URL", "Less semantic"],
      },
    ],
  },
  {
    id: "9",
    type: "direction_clarification",
    status: "pending",
    priority: "medium",
    title: "Error handling approach",
    description: "Need guidance on global error handling pattern",
    createdAt: "8m ago",
    source: "task_agent",
    sourceName: "GPT-4",
    threadId: "1",
    threadName: "User Authentication Flow",
    taskId: "t4",
    taskName: "Implement Signup API",
    overseerMessages: [
      {
        id: "dc1",
        role: "agent",
        agentId: "gpt-4",
        content:
          "I'm setting up the error handling for the signup API. I want to make sure I'm following the right patterns for this codebase.",
        timestamp: "8m ago",
      },
    ],
    clarificationContext: {
      currentUnderstanding:
        "I see that the codebase uses a mix of try-catch blocks and error boundaries. I'm not sure if there's a preferred approach for API routes specifically, or if we should implement a global error handler.",
      relevantCode: `// Current pattern in login.ts
try {
  const user = await authenticateUser(credentials);
  return { success: true, user };
} catch (error) {
  console.error('Login failed:', error);
  throw new Error('Authentication failed');
}`,
      relevantFiles: ["src/api/auth/login.ts", "src/middleware/errorHandler.ts"],
      approachOptions: [
        {
          id: "global-handler",
          description: "Implement a global error handler middleware that catches all errors",
          tradeoffs: "Centralized but may lose context-specific error handling",
        },
        {
          id: "local-handling",
          description: "Keep error handling local to each route with standardized error types",
          tradeoffs: "More boilerplate but better control over error responses",
        },
        {
          id: "hybrid",
          description: "Use local handling for known errors, global handler for unexpected errors",
          tradeoffs: "Best of both but more complex to maintain",
        },
      ],
    },
  },
  {
    id: "10",
    type: "task_plan_approval",
    status: "pending",
    priority: "medium",
    title: "Review task plan",
    description: "Password Reset Flow task plan needs your approval",
    createdAt: "5m ago",
    source: "thread_overseer",
    sourceName: "Alex Chen",
    threadId: "1",
    threadName: "User Authentication Flow",
    taskId: "t7",
    taskName: "Password Reset Flow",
  },
];

/**
 * Mock task execution data for demonstration.
 */
export const executionsData: TaskExecution[] = [
  {
    id: "exec-1",
    taskId: "t5",
    agentId: "claude-opus-4",
    agentName: "Claude Opus 4",
    status: "completed",
    createdAt: "10:30 AM",
    startedAt: "10:30 AM",
    completedAt: "10:31 AM",
    duration: "2m 34s",
    tokens: {
      input: 1250,
      output: 890,
      total: 2140,
    },
    output: {
      summary:
        "Implementation using React Hook Form with built-in validation. Features type-safe form handling, minimal re-renders, and a 94% smaller bundle than alternatives.",
      code: `import { useForm } from 'react-hook-form';

type LoginForm = {
  email: string;
  password: string;
};

export function LoginForm() {
  const { register, handleSubmit, formState: { errors } } = useForm<LoginForm>();

  const onSubmit = (data: LoginForm) => {
    // Handle login
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <input {...register('email', { required: true })} />
      {errors.email && <span>Email is required</span>}
      <input type="password" {...register('password', { required: true })} />
      {errors.password && <span>Password is required</span>}
      <button type="submit">Login</button>
    </form>
  );
}`,
    },
    evaluation: {
      score: 92,
      pros: [
        "Type-safe form handling with TypeScript",
        "Minimal bundle size (8kb gzipped)",
        "Excellent performance with minimal re-renders",
      ],
      cons: ["Less declarative validation syntax"],
    },
  },
  {
    id: "exec-2",
    taskId: "t5",
    agentId: "gpt-4",
    agentName: "GPT-4",
    status: "completed",
    createdAt: "10:30 AM",
    startedAt: "10:30 AM",
    completedAt: "10:32 AM",
    duration: "3m 12s",
    tokens: {
      input: 1480,
      output: 1120,
      total: 2600,
    },
    output: {
      summary:
        "Implementation using Formik with Yup validation schema. Provides declarative validation rules and comprehensive form state management.",
      code: `import { Formik, Form, Field } from 'formik';
import * as Yup from 'yup';

const LoginSchema = Yup.object().shape({
  email: Yup.string().email('Invalid email').required('Required'),
  password: Yup.string().min(8, 'Too short').required('Required'),
});

export function LoginForm() {
  return (
    <Formik
      initialValues={{ email: '', password: '' }}
      validationSchema={LoginSchema}
      onSubmit={(values) => {
        // Handle login
      }}
    >
      {({ errors, touched }) => (
        <Form>
          <Field name="email" />
          {errors.email && touched.email && <span>{errors.email}</span>}
          <Field name="password" type="password" />
          {errors.password && touched.password && <span>{errors.password}</span>}
          <button type="submit">Login</button>
        </Form>
      )}
    </Formik>
  );
}`,
    },
    evaluation: {
      score: 85,
      pros: [
        "Declarative validation with Yup schemas",
        "Comprehensive form state management",
        "Good ecosystem and documentation",
      ],
      cons: ["Larger bundle size (~45kb gzipped)", "More re-renders than React Hook Form"],
    },
  },
  {
    id: "exec-3",
    taskId: "t2",
    agentId: "claude-opus-4",
    agentName: "Claude Opus 4",
    status: "completed",
    createdAt: "10:33 AM",
    startedAt: "10:33 AM",
    completedAt: "10:34 AM",
    duration: "1m 12s",
    tokens: {
      input: 820,
      output: 456,
      total: 1276,
    },
    output: {
      summary:
        "Created User model with Zod validation, supporting email, password hash, and profile fields with proper TypeScript types.",
    },
  },
];

/**
 * Planning data for Dashboard Analytics thread (inbox item 4).
 */
export const planningDataByItemId: Record<string, PlanningData> = {
  "4": {
    issueTitle: "Dashboard Analytics",
    issueDescription:
      "Build a comprehensive analytics dashboard for monitoring key business metrics with real-time updates, interactive charts, and export functionality.",
    totalTasks: 12,
    parallelGroups: 5,
    estimatedDuration: "~4h",
    planSummary:
      "I've analyzed the requirements and broken down the dashboard into 12 tasks across 5 execution steps. The plan maximizes parallelization where possible — setup and schema design can run concurrently at the start, followed by parallel API development, then UI components in parallel, and finally integration tasks.",
    considerations: [
      "Recharts library chosen for charts — lightweight and React-native",
      "WebSocket for real-time updates will require backend infrastructure",
      "Export to PDF may need a headless browser (Puppeteer) for accurate rendering",
      "Date range filter should support custom ranges beyond presets",
    ],
    tasks: [
      {
        id: "t1",
        name: "Setup Chart Library",
        description: "Install and configure Recharts with proper TypeScript types",
        agent: "setup-agent",
        workers: [{ id: "claude-opus-4", name: "Claude Opus 4" }],
        dependencies: [],
        plannerPrompt:
          "Plan the chart library setup including package selection and TypeScript configuration.",
        planApproval: "auto" as const,
      },
      {
        id: "t2",
        name: "Design Data Schema",
        description: "Define analytics data models for metrics, time series, and aggregations",
        agent: "model-agent",
        workers: [{ id: "claude-opus-4", name: "Claude Opus 4" }],
        dependencies: [],
        plannerPrompt:
          "Design the data schema for analytics including time series data, aggregations, and metric definitions.",
        planApproval: "overseer" as const,
      },
      {
        id: "t3",
        name: "Build Metrics API",
        description: "Create REST endpoints for fetching analytics data with date filtering",
        agent: "api-agent",
        workers: [
          { id: "claude-opus-4", name: "Claude Opus 4" },
          { id: "gpt-4", name: "GPT-4" },
        ],
        dependencies: ["t2"],
        plannerPrompt:
          "Plan the metrics API endpoints including data fetching, filtering, aggregation, and caching strategy.",
        planApproval: "user" as const,
      },
      {
        id: "t4",
        name: "Real-time WebSocket",
        description: "Implement WebSocket server for pushing live metric updates",
        agent: "api-agent",
        workers: [{ id: "gpt-4o", name: "GPT-4o" }],
        dependencies: ["t2"],
      },
      {
        id: "t5",
        name: "Create Line Charts",
        description: "Build responsive line chart components for time series data",
        agent: "ui-agent",
        workers: [
          { id: "claude-opus-4", name: "Claude Opus 4" },
          { id: "claude-sonnet", name: "Claude Sonnet" },
        ],
        dependencies: ["t1", "t3"],
      },
      {
        id: "t6",
        name: "Create Bar Charts",
        description: "Build bar chart components for categorical comparisons",
        agent: "ui-agent",
        workers: [{ id: "claude-sonnet", name: "Claude Sonnet" }],
        dependencies: ["t1", "t3"],
      },
      {
        id: "t7",
        name: "Create Pie Charts",
        description: "Build pie/donut charts for distribution visualization",
        agent: "ui-agent",
        workers: [{ id: "gemini-pro", name: "Gemini Pro" }],
        dependencies: ["t1", "t3"],
      },
      {
        id: "t8",
        name: "Metrics Cards",
        description: "Build summary cards showing KPIs with trend indicators",
        agent: "ui-agent",
        workers: [
          { id: "claude-opus-4", name: "Claude Opus 4" },
          { id: "gpt-4", name: "GPT-4" },
          { id: "gemini-pro", name: "Gemini Pro" },
        ],
        dependencies: ["t3"],
      },
      {
        id: "t9",
        name: "Dashboard Layout",
        description: "Compose responsive grid layout with all chart components",
        agent: "ui-agent",
        workers: [{ id: "claude-opus-4", name: "Claude Opus 4" }],
        dependencies: ["t5", "t6", "t7", "t8"],
      },
      {
        id: "t10",
        name: "Live Updates UI",
        description: "Integrate WebSocket connection with chart components for real-time updates",
        agent: "ui-agent",
        workers: [{ id: "gpt-4o", name: "GPT-4o" }],
        dependencies: ["t4", "t9"],
      },
      {
        id: "t11",
        name: "Date Range Filter",
        description: "Add date picker with preset ranges and custom date selection",
        agent: "ui-agent",
        workers: [{ id: "claude-sonnet", name: "Claude Sonnet" }],
        dependencies: ["t9"],
      },
      {
        id: "t12",
        name: "Export to PDF",
        description: "Implement dashboard export using html2canvas and jsPDF",
        agent: "export-agent",
        workers: [{ id: "gpt-4", name: "GPT-4" }],
        dependencies: ["t9"],
      },
    ],
  },
};

/**
 * Thread review data for User Authentication Flow thread (inbox item 2).
 */
export const threadReviewDataByItemId: Record<string, ThreadReviewData> = {
  "2": {
    threadTitle: "User Authentication Flow",
    threadDescription: `Implement a **complete user authentication system** with the following features:

### Core Features
- **Login** — Email/password + OAuth providers (*Google*, *GitHub*)
- **Signup** — Email verification and strong password requirements
- **Password Reset** — Secure email tokens with expiration
- **Session Management** — JWT access tokens + refresh tokens

### Security Requirements
> The system must follow security best practices

1. Rate limiting on all auth endpoints
2. Password hashing with \`bcrypt\` (cost factor 12)
3. HTTPS-only cookies for tokens
4. CSRF protection on forms`,
    threadStatus: "pending",
    totalTasks: 8,
    parallelGroups: 5,
    estimatedDuration: "~3h",
    planSummary:
      "The authentication system has been broken down into 8 tasks across 5 execution steps. We start with provider setup, then model creation, followed by parallel API development for login/signup, then UI components, and finally integration tests.",
    securityConsiderations: [
      "All passwords must be hashed with bcrypt (cost factor 12) before storage",
      "JWT tokens should use short expiration (15min access, 7d refresh)",
      "Rate limiting required: 5 attempts/min for login, 3 for password reset",
      "OAuth state parameter must be validated to prevent CSRF attacks",
    ],
    technicalNotes: [
      "Using React Hook Form for type-safe form handling",
      "Zod schemas for runtime validation on both client and server",
      "OAuth flow will use PKCE for added security",
      "Session storage in Redis for horizontal scaling",
    ],
    tasks: [
      {
        id: "t1",
        name: "Setup Auth Provider",
        description: "Configure authentication provider with OAuth2 support for Google and GitHub",
        agent: "setup-agent",
        dependencies: [],
      },
      {
        id: "t2",
        name: "Create User Model",
        description:
          "Define user schema with email, password hash, and profile fields using Zod validation",
        agent: "model-agent",
        dependencies: ["t1"],
      },
      {
        id: "t3",
        name: "Implement Login API",
        description:
          "Build login endpoint with rate limiting, validation, and JWT token generation",
        agent: "api-agent",
        dependencies: ["t2"],
      },
      {
        id: "t4",
        name: "Implement Signup API",
        description:
          "Build signup endpoint with email verification and password strength requirements",
        agent: "api-agent",
        dependencies: ["t2"],
      },
      {
        id: "t5",
        name: "Build Login UI",
        description: "Create login form component with validation, error states, and OAuth buttons",
        agent: "ui-agent",
        dependencies: ["t3"],
      },
      {
        id: "t6",
        name: "Build Signup UI",
        description:
          "Create signup form with field validation, password strength meter, and terms checkbox",
        agent: "ui-agent",
        dependencies: ["t4"],
      },
      {
        id: "t7",
        name: "Password Reset Flow",
        description: "Implement forgot password and reset functionality with secure email tokens",
        agent: "api-agent",
        dependencies: ["t3", "t4"],
      },
      {
        id: "t8",
        name: "Integration Tests",
        description: "Write end-to-end tests for all auth flows including edge cases",
        agent: "test-agent",
        dependencies: ["t5", "t6", "t7"],
      },
    ],
  },
};
