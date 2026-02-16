import type { DemoComment } from "@/data/mockFlowgenticData";

export const mockThreadPatchData: Record<string, string> = {
  "src/middleware/auth.ts": `--- a/src/middleware/auth.ts
+++ b/src/middleware/auth.ts
@@ -1,15 +1,32 @@
-import { Request, Response, NextFunction } from "express";
+import { type Request, type Response, type NextFunction } from "express";
+import { verifyToken } from "../services/jwt";
+import { Result } from "../types/result";

-export function authMiddleware(req: Request, res: Response, next: NextFunction) {
-  const token = req.headers.authorization;
-  if (!token) {
-    return res.status(401).json({ error: "Unauthorized" });
+export function authMiddleware(
+  req: Request,
+  res: Response,
+  next: NextFunction,
+) {
+  const header = req.headers.authorization;
+  if (!header?.startsWith("Bearer ")) {
+    return res.status(401).json({ error: "Missing or malformed token" });
   }
-  // TODO: verify token
-  next();
+
+  const token = header.slice(7);
+  const result = verifyToken(token);
+
+  if (!result.ok) {
+    return res.status(401).json({ error: result.error.message });
+  }
+
+  req.user = result.value;
+  next();
 }
`,
  "src/services/jwt.ts": `--- /dev/null
+++ b/src/services/jwt.ts
@@ -0,0 +1,45 @@
+import jwt from "jsonwebtoken";
+import { Result, ok, err } from "../types/result";
+
+const SECRET = process.env.JWT_SECRET!;
+const ACCESS_TTL = "15m";
+const REFRESH_TTL = "7d";
+
+interface TokenPayload {
+  userId: string;
+  roles: string[];
+}
+
+export function signAccessToken(payload: TokenPayload): string {
+  return jwt.sign(payload, SECRET, { expiresIn: ACCESS_TTL });
+}
+
+export function signRefreshToken(payload: TokenPayload): string {
+  return jwt.sign(payload, SECRET, { expiresIn: REFRESH_TTL });
+}
+
+export function verifyToken(token: string): Result<TokenPayload, Error> {
+  try {
+    const decoded = jwt.verify(token, SECRET) as TokenPayload;
+    return ok(decoded);
+  } catch (e) {
+    return err(e instanceof Error ? e : new Error("Invalid token"));
+  }
+}
`,
  "src/models/user.ts": `--- a/src/models/user.ts
+++ b/src/models/user.ts
@@ -5,6 +5,8 @@
 export interface User {
   id: string;
   email: string;
+  passwordHash: string;
+  roles: string[];
   name: string;
   createdAt: Date;
   updatedAt: Date;
`,
};

export const mockThreadComments: DemoComment[] = [
  {
    id: "tc-1",
    lineNumber: 8,
    side: "additions",
    author: "Overseer",
    content: "Good — using Bearer scheme validation before extraction.",
    timestamp: "2026-02-15T10:15:00Z",
    file: "src/middleware/auth.ts",
  },
];
