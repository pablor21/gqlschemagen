"use client";

import { Accordion, AccordionItem } from "@heroui/react";
import { ReactNode, useId } from "react";

export const Collapsible = ({
  title,
  label,
  children,
}: {
  title: string;
  label: string;
  children: ReactNode;
}) => {
  const id = useId();

  return (
    <Accordion variant="splitted" className="not-prose">
      <AccordionItem key={id} aria-label={label} title={title}>
        {children}
      </AccordionItem>
    </Accordion>
  );
};

export { Accordion, AccordionItem, Alert, Link, Snippet } from "@heroui/react";
