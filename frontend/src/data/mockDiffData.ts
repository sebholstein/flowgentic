import type { ExecutionDiff, LineComment, FileDiff } from "@/types/code-review";

export const mockFileDiffs: FileDiff[] = [
  {
    path: "src/components/LoginForm.tsx",
    status: "added",
    language: "typescript",
    oldContent: "",
    newContent: `import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

interface LoginFormProps {
  onSubmit: (credentials: { email: string; password: string }) => void;
  isLoading?: boolean;
}

export function LoginForm({ onSubmit, isLoading = false }: LoginFormProps) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [errors, setErrors] = useState<{ email?: string; password?: string }>({});

  const validateForm = () => {
    const newErrors: { email?: string; password?: string } = {};

    if (!email) {
      newErrors.email = 'Email is required';
    } else if (!/^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/.test(email)) {
      newErrors.email = 'Please enter a valid email';
    }

    if (!password) {
      newErrors.password = 'Password is required';
    } else if (password.length < 8) {
      newErrors.password = 'Password must be at least 8 characters';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (validateForm()) {
      onSubmit({ email, password });
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="email">Email</Label>
        <Input
          id="email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="you@example.com"
          className={errors.email ? 'border-red-500' : ''}
        />
        {errors.email && (
          <p className="text-sm text-red-500">{errors.email}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="password">Password</Label>
        <Input
          id="password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="Enter your password"
          className={errors.password ? 'border-red-500' : ''}
        />
        {errors.password && (
          <p className="text-sm text-red-500">{errors.password}</p>
        )}
      </div>

      <Button type="submit" className="w-full" disabled={isLoading}>
        {isLoading ? 'Signing in...' : 'Sign In'}
      </Button>
    </form>
  );
}`,
    additions: 72,
    deletions: 0,
  },
  {
    path: "src/lib/auth.ts",
    status: "modified",
    language: "typescript",
    oldContent: `export function getCurrentUser() {
  return null;
}

export function isAuthenticated() {
  return false;
}`,
    newContent: `import { jwtDecode } from 'jwt-decode';

interface User {
  id: string;
  email: string;
  name: string;
  role: 'user' | 'admin';
}

interface JWTPayload {
  sub: string;
  email: string;
  name: string;
  role: 'user' | 'admin';
  exp: number;
}

const TOKEN_KEY = 'auth_token';

export function getCurrentUser(): User | null {
  const token = localStorage.getItem(TOKEN_KEY);
  if (!token) return null;

  try {
    const payload = jwtDecode<JWTPayload>(token);

    // Check if token is expired
    if (payload.exp * 1000 < Date.now()) {
      localStorage.removeItem(TOKEN_KEY);
      return null;
    }

    return {
      id: payload.sub,
      email: payload.email,
      name: payload.name,
      role: payload.role,
    };
  } catch {
    localStorage.removeItem(TOKEN_KEY);
    return null;
  }
}

export function isAuthenticated(): boolean {
  return getCurrentUser() !== null;
}

export async function login(email: string, password: string): Promise<User> {
  const response = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || 'Login failed');
  }

  const { token } = await response.json();
  localStorage.setItem(TOKEN_KEY, token);

  const user = getCurrentUser();
  if (!user) throw new Error('Failed to decode user from token');

  return user;
}

export function logout(): void {
  localStorage.removeItem(TOKEN_KEY);
}

export function getAuthHeader(): { Authorization: string } | {} {
  const token = localStorage.getItem(TOKEN_KEY);
  return token ? { Authorization: \`Bearer \${token}\` } : {};
}`,
    additions: 65,
    deletions: 6,
  },
  {
    path: "src/hooks/useAuth.ts",
    status: "added",
    language: "typescript",
    oldContent: "",
    newContent: `import { useState, useEffect, useCallback } from 'react';
import { getCurrentUser, login as authLogin, logout as authLogout } from '@/lib/auth';

interface User {
  id: string;
  email: string;
  name: string;
  role: 'user' | 'admin';
}

interface UseAuthReturn {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  error: string | null;
}

export function useAuth(): UseAuthReturn {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const currentUser = getCurrentUser();
    setUser(currentUser);
    setIsLoading(false);
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    setIsLoading(true);
    setError(null);

    try {
      const loggedInUser = await authLogin(email, password);
      setUser(loggedInUser);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Login failed';
      setError(message);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const logout = useCallback(() => {
    authLogout();
    setUser(null);
  }, []);

  return {
    user,
    isLoading,
    isAuthenticated: user !== null,
    login,
    logout,
    error,
  };
}`,
    additions: 54,
    deletions: 0,
  },
  {
    path: "src/types/auth.ts",
    status: "added",
    language: "typescript",
    oldContent: "",
    newContent: `export interface User {
  id: string;
  email: string;
  name: string;
  role: 'user' | 'admin';
}

export interface LoginCredentials {
  email: string;
  password: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}`,
    additions: 15,
    deletions: 0,
  },
  {
    path: "src/components/AuthProvider.tsx",
    status: "added",
    language: "typescript",
    oldContent: "",
    newContent: `import { createContext, useContext, type ReactNode } from 'react';
import { useAuth } from '@/hooks/useAuth';

interface User {
  id: string;
  email: string;
  name: string;
  role: 'user' | 'admin';
}

interface AuthContextValue {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  error: string | null;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const auth = useAuth();

  return (
    <AuthContext.Provider value={auth}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuthContext(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuthContext must be used within an AuthProvider');
  }
  return context;
}`,
    additions: 38,
    deletions: 0,
  },
];

export const mockExecutionDiff: ExecutionDiff = {
  executionId: "exec-1",
  taskId: "t5",
  agentName: "Claude Opus 4",
  agentId: "claude-opus-4",
  files: mockFileDiffs,
  totalAdditions: mockFileDiffs.reduce((sum, f) => sum + f.additions, 0),
  totalDeletions: mockFileDiffs.reduce((sum, f) => sum + f.deletions, 0),
  timestamp: "2025-01-15T10:30:00Z",
};

export const mockComments: LineComment[] = [
  {
    id: "c1",
    executionId: "exec-1",
    filePath: "src/components/LoginForm.tsx",
    lineNumber: 7,
    author: {
      type: "user",
      id: "u1",
      name: "Sebastian",
    },
    content:
      "Consider adding email validation here before submit. Maybe use zod for schema validation?",
    createdAt: "2025-01-15T10:35:00Z",
    resolved: false,
    actionType: "suggestion",
    replies: [
      {
        id: "c2",
        executionId: "exec-1",
        filePath: "src/components/LoginForm.tsx",
        lineNumber: 7,
        parentId: "c1",
        author: {
          type: "agent",
          id: "claude-opus-4",
          name: "Claude Opus 4",
        },
        content:
          "Good point! I've added basic validation in the validateForm function. For production, zod would be a great addition. Want me to add that in the next iteration?",
        createdAt: "2025-01-15T10:36:00Z",
        resolved: false,
      },
    ],
  },
  {
    id: "c3",
    executionId: "exec-1",
    filePath: "src/components/LoginForm.tsx",
    lineNumber: 35,
    author: {
      type: "overseer",
      id: "overseer",
      name: "Overseer",
    },
    content:
      "The form validation looks good. Consider adding rate limiting on the API side to prevent brute force attacks.",
    createdAt: "2025-01-15T10:40:00Z",
    resolved: true,
    actionType: "approve",
  },
  {
    id: "c4",
    executionId: "exec-1",
    filePath: "src/lib/auth.ts",
    lineNumber: 24,
    author: {
      type: "user",
      id: "u1",
      name: "Sebastian",
    },
    content:
      "Should we handle the case where jwtDecode throws? The try-catch is good but maybe add more specific error handling.",
    createdAt: "2025-01-15T10:42:00Z",
    resolved: false,
    actionType: "question",
  },
  {
    id: "c5",
    executionId: "exec-1",
    filePath: "src/lib/auth.ts",
    lineNumber: 45,
    author: {
      type: "user",
      id: "u1",
      name: "Sebastian",
    },
    content:
      "We should add retry logic here for network failures. Consider using something like fetch-retry or implementing exponential backoff.",
    createdAt: "2025-01-15T10:45:00Z",
    resolved: false,
    actionType: "request_change",
  },
];

// Alternative execution for comparison
export const mockExecutionDiff2: ExecutionDiff = {
  executionId: "exec-2",
  taskId: "t5",
  agentName: "GPT-4",
  agentId: "gpt-4",
  files: [
    {
      path: "src/components/LoginForm.tsx",
      status: "added",
      language: "typescript",
      oldContent: "",
      newContent: `import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

const loginSchema = z.object({
  email: z.string().email('Invalid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
});

type LoginFormData = z.infer<typeof loginSchema>;

interface LoginFormProps {
  onSubmit: (data: LoginFormData) => void;
  isLoading?: boolean;
}

export function LoginForm({ onSubmit, isLoading }: LoginFormProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
  });

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div>
        <Input
          {...register('email')}
          type="email"
          placeholder="Email"
        />
        {errors.email && (
          <span className="text-red-500 text-sm">{errors.email.message}</span>
        )}
      </div>

      <div>
        <Input
          {...register('password')}
          type="password"
          placeholder="Password"
        />
        {errors.password && (
          <span className="text-red-500 text-sm">{errors.password.message}</span>
        )}
      </div>

      <Button type="submit" disabled={isLoading}>
        {isLoading ? 'Loading...' : 'Login'}
      </Button>
    </form>
  );
}`,
      additions: 52,
      deletions: 0,
    },
    {
      path: "src/lib/auth.ts",
      status: "modified",
      language: "typescript",
      oldContent: `export function getCurrentUser() {
  return null;
}

export function isAuthenticated() {
  return false;
}`,
      newContent: `const API_URL = '/api/auth';

export async function getCurrentUser() {
  const token = localStorage.getItem('token');
  if (!token) return null;

  const res = await fetch(\`\${API_URL}/me\`, {
    headers: { Authorization: \`Bearer \${token}\` },
  });

  if (!res.ok) return null;
  return res.json();
}

export function isAuthenticated() {
  return !!localStorage.getItem('token');
}

export async function login(email: string, password: string) {
  const res = await fetch(\`\${API_URL}/login\`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });

  if (!res.ok) throw new Error('Login failed');

  const { token } = await res.json();
  localStorage.setItem('token', token);
}

export function logout() {
  localStorage.removeItem('token');
}`,
      additions: 32,
      deletions: 6,
    },
  ],
  totalAdditions: 84,
  totalDeletions: 6,
  timestamp: "2025-01-15T10:32:00Z",
};

// Helper to get file icon based on path
export function getFileIcon(path: string): string {
  const ext = path.split(".").pop()?.toLowerCase();
  switch (ext) {
    case "ts":
    case "tsx":
      return "typescript";
    case "js":
    case "jsx":
      return "javascript";
    case "css":
      return "css";
    case "html":
      return "html";
    case "json":
      return "json";
    case "py":
      return "python";
    case "md":
      return "markdown";
    default:
      return "file";
  }
}

// Helper to get language from file path
export function getLanguageFromPath(path: string): string {
  const ext = path.split(".").pop()?.toLowerCase();
  switch (ext) {
    case "ts":
    case "tsx":
      return "typescript";
    case "js":
    case "jsx":
      return "javascript";
    case "css":
      return "css";
    case "html":
      return "html";
    case "json":
      return "json";
    case "py":
      return "python";
    default:
      return "text";
  }
}
