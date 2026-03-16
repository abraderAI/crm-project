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

export default clerkMiddleware(async (auth, request) => {
  // Public routes: no auth required.
  if (isPublicRoute(request)) {
    return NextResponse.next();
  }

  // All other routes require authentication.
  await auth.protect();

  const { pathname } = request.nextUrl;

  // Admin routes: redirect non-platform-admins to home.
  if (isAdminRoute(request)) {
    const tierHeader = request.headers.get("x-user-tier");
    if (tierHeader !== "6") {
      console.warn(`[middleware] Non-admin access attempt to ${pathname}`);
      return NextResponse.redirect(new URL("/", request.url));
    }
  }

  // Global leads API: reject non-DEFT org users with 403.
  if (isGlobalLeadsRoute(request)) {
    const tierHeader = request.headers.get("x-user-tier");
    if (tierHeader !== "4" && tierHeader !== "6") {
      console.warn(`[middleware] Non-DEFT access attempt to ${pathname}`);
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
