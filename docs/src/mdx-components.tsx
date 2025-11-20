import { Alert, Snippet } from "./components";

import { CodeBlock } from "@/components/ui/code-block";
import type { MDXComponents } from "mdx/types";
import { cn } from "@heroui/react";

export function useMDXComponents(): MDXComponents {
  return {
    Snippet: ({ className, ...props }) => (
      <Snippet className={cn("not-prose", className)} {...props} />
    ),
    Alert: ({ classNames, ...props }) => (
      <Alert
        classNames={{
          ...classNames,
          base: cn("not-prose", classNames?.base),
          description: cn("prose", classNames?.description),
        }}
        {...props}
      />
    ),
    CodeBlock: ({ className, ...props }) => (
      <CodeBlock className={cn("not-prose", className)} {...props} />
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
