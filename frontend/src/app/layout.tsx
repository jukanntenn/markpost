import type { Metadata } from "next";
import { ThemeProvider } from "@/components/theme-provider";
import { QueryProvider } from "@/components/providers/QueryProvider";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { UserInfoProvider } from "@/components/UserInfoProvider";
import "./globals.css";

export const metadata: Metadata = {
  title: "Markpost",
  description: "Markpost - A simple posting service",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className="min-h-screen bg-background font-sans antialiased">
        <UserInfoProvider>
          <QueryProvider>
            <ThemeProvider
              attribute="class"
              defaultTheme="system"
              enableSystem
              disableTransitionOnChange
            >
              <TooltipProvider delayDuration={200}>
                {children}
                <Toaster position="top-right" style={{ zIndex: 2000 }} />
              </TooltipProvider>
            </ThemeProvider>
          </QueryProvider>
        </UserInfoProvider>
      </body>
    </html>
  );
}
