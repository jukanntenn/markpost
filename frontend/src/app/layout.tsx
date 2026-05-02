import type { Metadata } from "next";
import { getLocale, getMessages } from "next-intl/server";
import { ThemeProvider } from "@/components/theme-provider";
import { QueryProvider } from "@/components/providers/QueryProvider";
import { LocaleProvider } from "@/components/providers/LocaleProvider";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import "./globals.css";

export const metadata: Metadata = {
  title: "Markpost",
  description: "Markpost - A simple posting service",
};

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const locale = await getLocale();
  const messages = await getMessages();

  return (
    <html lang={locale} suppressHydrationWarning>
      <body className="min-h-screen bg-background font-sans antialiased">
        <LocaleProvider serverLocale={locale} serverMessages={messages}>
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
        </LocaleProvider>
      </body>
    </html>
  );
}
