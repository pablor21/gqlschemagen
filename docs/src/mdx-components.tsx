import { Alert, Snippet } from "./components";

import { CodeBlock } from "@/components/ui/code-block";
import type { MDXComponents } from "mdx/types";
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
    /* pre: ({ children }) => (
      <CodeBlock
        filename=""
        // extract language from className like "language-yaml"
        language={children.props.className?.split(" ")[0].split("-")[1]}
      >
        {children.props.children}
      </CodeBlock>
    ), */
  };
}
