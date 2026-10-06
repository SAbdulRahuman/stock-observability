"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "next-themes";
import { useState } from "react";
import { PrivacyProvider } from "@/components/privacy";

export function Providers({ children }: { children: React.ReactNode }) {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { retry: 1, refetchOnWindowFocus: true, placeholderData: (prev: unknown) => prev },
        },
      }),
  );
  return (
    <ThemeProvider attribute="class" defaultTheme="dark" enableSystem disableTransitionOnChange>
      <QueryClientProvider client={client}>
        <PrivacyProvider>{children}</PrivacyProvider>
      </QueryClientProvider>
    </ThemeProvider>
  );
}
