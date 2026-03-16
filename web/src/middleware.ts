import { clerkMiddleware, createRouteMatcher } from "@clerk/nextjs/server";
import { NextResponse } from "next/server";

/** Routes that do not require authentication at all. */
const isPublicRoute = createRouteMatcher([
  "/sign-in(.*)",
  "/sign-up(.*)",
  "/api/webhooks(.*)",
  "/api/healthz",
  "/docs(.*)",
  "/forum(.*)",
]);

/** Routes restricted to platform admins only. */
const isAdminRoute = createRouteMatcher(["/admin(.*)"]);

/** API routes restricted to DEFT org members. */
const isGlobalLeadsRoute = createRouteMatcher(["/api/v1/global-leads(.*)"]);

const API_BASE =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/**
 * Resolves the current user's tier by calling the backend tier endpoint.
 * Returns 1 (anonymous) on error, timeout, or missing token.
 */
async function resolveUserTier(token: string): Promise<number> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 2000);
  try {
    const res = await fetch(`${API_BASE}/v1/me/tier`, {
      headers: { Authorization: `Bearer ${token}` },
      signal: controller.signal,
    });
    clearTimeout(timeout);
    if (!res.ok) return 1;
    const data = (await res.json()) as { tier?: unknown };
    return typeof data.tier === "number" ? data.tier : 1;
  } catch {
    clearTimeout(timeout);
    return 1;
  }
}

export default clerkMiddleware(async (auth, request) => {
  // Public routes: no auth required.
  if (isPublicRoute(request)) {
    return NextResponse.next();
  }

  // All other routes require authentication.
  await auth.protect();

  const { pathname } = request.nextUrl;
  const needsTierCheck = isAdminRoute(request) || isGlobalLeadsRoute(request);

  if (needsTierCheck) {
    const authObj = await auth();
    const token = await authObj.getToken();
    const tier = token ? await resolveUserTier(token) : 1;

    // Admin routes: redirect non-platform-admins to home.
    if (isAdminRoute(request) && tier !== 6) {
      console.warn(
        `[middleware] Non-admin access attempt to ${pathname} (tier=${tier})`,
      );
      return NextResponse.redirect(new URL("/", request.url));
    }

    // Global leads API: reject non-DEFT org users with 403.
    if (isGlobalLeadsRoute(request) && tier !== 4 && tier !== 6) {
      console.warn(
        `[middleware] Non-DEFT access attempt to ${pathname} (tier=${tier})`,
      );
      return NextResponse.json(
        {
          type: "about:blank",
          title: "Forbidden",
          status: 403,
          detail: "Access restricted to DEFT organization members",
        },
        { status: 403 },
      );
    }
  }

  return NextResponse.next();
});

export const config = {
  matcher: [
    // Skip Next.js internals and all static files, unless found in search params.
    "/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)",
    // Always run for API routes.
    "/(api|trpc)(.*)",
  ],
};
