import { Alert, Link, Snippet } from "./components";
import { CodeBlock, CodeHighlight } from "@/components/ui/code-block";

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
    pre: ({ children }) => {
      const codeEl = children.props;

      // codeEl.className → "language-ts", "language-yaml", etc.
      const lang =
        codeEl.className
          ?.split(" ")
          ?.find((c: string) => c.startsWith("language-"))
          ?.replace("language-", "") ?? "txt";

      const code = codeEl.children ?? "";

      return <CodeHighlight language={lang}>{code}</CodeHighlight>;
    },
    code: (props) => <code className="font-mono wrap-break-word" {...props} />,
    a: ({ href, ...props }) =>
      href.startsWith("/") ? (
        <NextLink href={href} {...props} />
      ) : (
        <Link href={href} isExternal {...props} />
      ),
    table: (props) => (
      <div className="w-full overflow-x-auto">
        <table className="w-full table-auto" {...props} />
      </div>
    ),
  };
}
