import type { Metadata } from "next";
import { Playfair_Display, Source_Sans_3, Fira_Code } from "next/font/google";
import { getLocale, getMessages } from "next-intl/server";
import { ThemeProvider } from "@/components/theme-provider";
import { QueryProvider } from "@/components/providers/QueryProvider";
import { LocaleProvider } from "@/components/providers/LocaleProvider";
import { ToastProvider } from "@/components/ui/toast";
import "./globals.css";

const playfairDisplay = Playfair_Display({
  subsets: ["latin"],
  variable: "--font-playfair-display",
  display: "swap",
});

const sourceSans3 = Source_Sans_3({
  subsets: ["latin"],
  variable: "--font-source-sans",
  display: "swap",
});

const firaCode = Fira_Code({
  subsets: ["latin"],
  variable: "--font-fira-code",
  display: "swap",
});

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
    <html
      lang={locale}
      className={`${playfairDisplay.variable} ${sourceSans3.variable} ${firaCode.variable}`}
      suppressHydrationWarning
    >
      <body className="min-h-screen bg-background font-body antialiased">
        <LocaleProvider serverLocale={locale} serverMessages={messages}>
          <QueryProvider>
            <ThemeProvider
              attribute="class"
              defaultTheme="system"
              enableSystem
              disableTransitionOnChange
            >
              <ToastProvider>
                {children}
              </ToastProvider>
            </ThemeProvider>
          </QueryProvider>
        </LocaleProvider>
      </body>
    </html>
  );
}
