import { Alert, Link, Snippet } from "./components";

import { CodeBlock } from "@/components/ui/code-block";
import type { MDXComponents } from "mdx/types";
import NextLink from "next/link";
import { cn } from "@heroui/react";

export function useMDXComponents(): MDXComponents {
  return {
    Snippet: ({ className, classNames, ...props }) => (
      <Snippet
        classNames={{
          ...classNames,
          base: cn("max-w-full my-2", classNames?.base),
          pre: cn("flex gap-2 whitespace-pre-wrap leading-5", classNames?.pre),
        }}
        className={cn("not-prose", className)}
        {...props}
      />
    ),
    Alert: ({ classNames, ...props }) => (
      <Alert
        classNames={{
          ...classNames,
          base: cn("not-prose my-2", classNames?.base),
          description: cn("prose", classNames?.description),
        }}
        {...props}
      />
    ),
    CodeBlock: ({ className, ...props }) => (
      <CodeBlock className={cn("not-prose my-2", className)} {...props} />
    ),
    pre: ({ children }) => (
      <pre className="bg-content1 text-zinc-100 p-4 rounded-lg border border-content2 overflow-x-auto text-sm leading-relaxed my-5">
        <code className="block font-mono">{children}</code>
      </pre>
    ),
    a: ({ href, ...props }) =>
      href.startsWith("/") ? (
        <NextLink href={href} {...props} />
      ) : (
        <Link href={href} isExternal {...props} />
      ),
  };
}
