"use client";

import type { BundledLanguage, BundledTheme } from "shiki";
import { Check, Copy, FileIcon } from "lucide-react";
import { Skeleton, cn } from "@heroui/react";
import {
  transformerNotationDiff,
  transformerNotationFocus,
  transformerNotationHighlight,
} from "@shikijs/transformers";
import { useEffect, useState } from "react";

interface CodeHighlightProps {
  children: string;
  language: BundledLanguage;
  theme?: BundledTheme;
  className?: string;
}

export const CodeHighlight = ({
  children,
  language,
  theme = "github-dark",
  className,
}: CodeHighlightProps) => {
  const [html, setHtml] = useState("");
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    const run = async () => {
      try {
        setIsLoading(true);

        const { codeToHtml } = await import("shiki");

        const rendered = await codeToHtml(children, {
          lang: language,
          theme,
          transformers: [
            transformerNotationHighlight({ matchAlgorithm: "v3" }),
            transformerNotationDiff({ matchAlgorithm: "v3" }),
            transformerNotationFocus({ matchAlgorithm: "v3" }),
          ],
        });

        if (!cancelled) {
          setHtml(rendered);
          setIsLoading(false);
        }
      } catch {
        if (!cancelled) {
          setHtml(`<pre class="shiki ${theme}"><code>${children}</code></pre>`);
          setIsLoading(false);
        }
      }
    };

    run();
    return () => {
      cancelled = true;
    };
  }, [children, language, theme]);

  if (isLoading)
    return (
      <div className="p-4 space-y-3">
        <Skeleton className="h-4 w-1/3 rounded" />
        <Skeleton className="h-4 w-3/4 rounded" />
        <Skeleton className="h-4 w-1/2 rounded" />
        <Skeleton className="h-4 w-2/3 rounded" />
      </div>
    );

  return (
    <div
      className={className}
      dangerouslySetInnerHTML={{ __html: html }}
      style={{ backgroundColor: "transparent" }}
    />
  );
};

interface CodeBlockProps {
  children: string;
  language: BundledLanguage;
  filename: string;
  theme?: BundledTheme;
  className?: string;
  label?: string; // Added to show "before" or "after" labels
  hideCopyButton?: boolean;
}

export const CodeBlock = ({
  children,
  language,
  filename,
  theme = "github-dark",
  className,
  label,
  hideCopyButton = false,
}: CodeBlockProps) => {
  const [isCopied, setIsCopied] = useState(false);

  const handleCopy = async () => {
    if (!children) return;

    try {
      await navigator.clipboard.writeText(children);
      setIsCopied(true);
      setTimeout(() => setIsCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  };

  return (
    <div
      className={cn(
        "relative w-full group overflow-hidden bg-zinc-900 text-white",
        "rounded-xl border border-zinc-800 shadow-2xl max-w-3xl mx-auto",
        className
      )}
    >
      <div className="flex items-center justify-between px-4 py-3 border-b border-zinc-800 bg-zinc-950/70 h-12">
        <div className="flex items-center gap-2 text-sm text-zinc-400">
          <FileIcon className="w-4 h-4 text-blue-400" />
          <span className="font-medium font-mono">{filename}</span>
          {label && (
            <span className="text-zinc-600 ml-1 text-xs">({label})</span>
          )}
        </div>

        {!hideCopyButton && (
          <button
            onClick={handleCopy}
            className="p-1.5 rounded-md hover:bg-zinc-800 transition-all text-zinc-500 hover:text-zinc-200"
          >
            {isCopied ? (
              <Check className="w-4 h-4 text-green-400" />
            ) : (
              <Copy className="w-4 h-4" />
            )}
          </button>
        )}
      </div>

      <div className="relative overflow-x-auto">
        <CodeHighlight language={language} theme={theme}>
          {children}
        </CodeHighlight>
      </div>

      <div className="absolute -inset-1 bg-linear-to-r from-blue-500/20 to-purple-500/20 blur-3xl -z-10 opacity-20" />
    </div>
  );
};

interface CodeComparisonProps {
  beforeCode: string;
  afterCode: string;
  // We use BundledLanguage from shiki, but export a string type for ease of use
  language: BundledLanguage | string;
  filename: string;
  theme?: BundledTheme;
  className?: string;
}

export function CodeComparison({
  beforeCode,
  afterCode,
  // Cast language to BundledLanguage for CodeBlock component
  language,
  filename,
  theme = "github-dark",
  className,
}: CodeComparisonProps) {
  // We don't need to manually run the highlighting logic here anymore,
  // as CodeBlock handles it internally.

  return (
    <div className={cn("mx-auto w-full max-w-5xl my-8", className)}>
      <div className="relative w-full overflow-hidden rounded-xl border border-zinc-800 shadow-2xl bg-[#0d1117]">
        <div className="relative grid md:grid-cols-2 divide-y md:divide-y-0 md:divide-x divide-zinc-800">
          {/* Left Side (Before) */}
          <div className="relative">
            <CodeBlock
              // Cast language for CodeBlock props
              language={language as BundledLanguage}
              filename={filename}
              theme={theme}
              label="before"
              // Override the default CodeBlock styles to fit the grid
              className="rounded-none border-none shadow-none bg-transparent"
              hideCopyButton
            >
              {beforeCode}
            </CodeBlock>
          </div>

          {/* Right Side (After) */}
          <div className="relative">
            <CodeBlock
              // Cast language for CodeBlock props
              language={language as BundledLanguage}
              filename={filename}
              theme={theme}
              label="after"
              // Override the default CodeBlock styles to fit the grid
              className="rounded-none border-none shadow-none bg-transparent"
            >
              {afterCode}
            </CodeBlock>
          </div>
        </div>

        {/* VS Badge */}
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 flex h-8 w-8 items-center justify-center rounded-full bg-zinc-800 border border-zinc-700 text-xs font-bold text-zinc-400 shadow-lg z-10">
          VS
        </div>
      </div>
    </div>
  );
}
