import "./globals.css";

import { Exo_2, Raleway } from "next/font/google";

import type { Metadata } from "next";
import { Providers } from "./providers";
import Script from "next/script";

const raleway = Raleway({
  variable: "--font-raleway",
  subsets: ["latin"],
});

const exo = Exo_2({
  variable: "--font-exo",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: {
    template: "%s | GQLSchemaGen",
    default: "GQLSchemaGen",
  },
  description:
    "Documentation for GQLSchemaGen, the Go-based GraphQL schema generator. Learn how to install, configure, and extend the tool with practical examples.",
  authors: [
    {
      name: "Pablo R.",
      url: "https://github.com/pablor21",
    },
    {
      name: "RaviAnand M.",
      url: "https://github.com/dan6erbond",
    },
  ],
  keywords: [
    "GraphQL",
    "Go",
    "Schema Generation",
    "GQLSchemaGen",
    "gqlgen",
    "GraphQL tooling",
  ],
  openGraph: {
    title: "GQLSchemaGen Documentation",
    description:
      "The official documentation for GQLSchemaGen — generate GraphQL schemas directly from Go models.",
    type: "website",
    locale: "en_US",
    siteName: "GQLSchemaGen",
  },
  metadataBase: new URL(
    process.env.PAGES_ORIGIN ?? `http://localhost:${process.env.PORT ?? 3000}`
  ),
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">

      <body
        className={`${raleway.variable} ${exo.variable} antialiased font-sans dark:bg-black`}
      >
        {process.env.NODE_ENV === "production" && (
          <Script
            defer
            src="https://analytics.pramirez.dev/script.js"
            data-website-id="0a9146bb-75d8-447c-80a1-3f37938fdc84"
          />
        )}
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
