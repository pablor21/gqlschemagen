"use client";

import { Button, ButtonProps } from "@heroui/react";
import Link, { LinkProps } from "next/link";

import { ArrowRight } from "lucide-react";

export default function NextButton({
  children,
  ...props
}: ButtonProps & LinkProps) {
  return (
    <div className="mt-12 flex justify-end not-prose">
      <Button
        color="primary"
        variant="flat"
        as={Link}
        className="min-h-16 h-auto flex flex-col items-start gap-0 no-underline text-lg"
        {...props}
      >
        <span className="text-sm text-gray-300">Next</span>
        <div className="flex gap-2 items-center">
          {typeof children === "string" ? <span>{children}</span> : children}
          <ArrowRight size={16} />
        </div>
      </Button>
    </div>
  );
}
