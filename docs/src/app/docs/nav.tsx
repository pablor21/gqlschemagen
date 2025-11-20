"use client";

import {
  Button,
  Link,
  Navbar,
  NavbarContent,
  NavbarItem,
  NavbarMenu,
  NavbarMenuItem,
  NavbarMenuToggle,
} from "@heroui/react";
import React, { ReactNode, useState } from "react";

import { Github } from "lucide-react";
import NavbarBrand from "@/components/layout/navbar-brand";
import { usePathname } from "next/navigation";

const docsNavItems = [
  { label: "Introduction", href: "/docs" },
  { label: "Getting Started", href: "/docs/getting-started" },
  { label: "Configuration", href: "/docs/configuration" },
  {
    label: "Features",
    items: [
      { label: "GQL Types", href: "/docs/features/gql-types" },
      { label: "GQL Inputs", href: "/docs/features/gql-inputs" },
      { label: "GQL Enums", href: "/docs/features/gql-enums" },
      { label: "Namespaces", href: "/docs/features/namespaces" },
    ],
  },
  {
    label: "Integrations",
    items: [{ label: "GQLGen", href: "/docs/integrations/gqlgen" }],
  },
  { label: "CLI Reference", href: "/docs/cli-reference" },
];

const getIsActive = (href: string, current: string) => current === href;

const SidebarLink = ({
  href,
  children,
  isSub = false,
}: {
  href: string;
  children: ReactNode;
  isSub?: boolean;
}) => {
  const pathname = usePathname();
  const isActive = getIsActive(href, pathname);

  return (
    <Link
      href={href}
      data-active={isActive}
      className={[
        "w-full text-left text-sm transition-colors",
        isSub ? "pl-4" : "pl-0",
        isActive
          ? "text-primary font-semibold"
          : isSub
          ? "text-default-500"
          : "text-default-700 hover:text-primary",
      ].join(" ")}
      color="foreground"
    >
      {children}
    </Link>
  );
};

export function Sidebar() {
  return (
    <aside className="hidden lg:block w-64 xl:w-72 pt-8 pr-8 sticky top-16 h-[calc(100vh-4rem)] overflow-y-auto">
      <div className="flex flex-col space-y-4">
        {docsNavItems.map((item, index) => (
          <React.Fragment key={index}>
            {item.href ? (
              <SidebarLink href={item.href}>{item.label}</SidebarLink>
            ) : (
              <div className="flex flex-col space-y-2">
                <h4 className="text-md font-semibold text-foreground">
                  {item.label}
                </h4>
                {item.items?.map((subItem, i) => (
                  <SidebarLink key={i} href={subItem.href} isSub>
                    {subItem.label}
                  </SidebarLink>
                ))}
              </div>
            )}
          </React.Fragment>
        ))}
      </div>
    </aside>
  );
}

export default function DocsNavbar() {
  const pathname = usePathname();
  const [isMenuOpen, setIsMenuOpen] = useState(false);

  const isDocsActive = pathname.startsWith("/docs");
  const isExamplesActive = pathname.startsWith("/examples");

  return (
    <Navbar
      maxWidth="xl"
      isMenuOpen={isMenuOpen}
      onMenuOpenChange={setIsMenuOpen}
      className="bg-transparent backdrop-blur-md border-b border-white/5"
      classNames={{
        item: [
          "flex",
          "relative",
          "h-full",
          "items-center",
          "data-[active=true]:after:content-['']",
          "data-[active=true]:after:absolute",
          "data-[active=true]:after:bottom-0",
          "data-[active=true]:after:left-0",
          "data-[active=true]:after:right-0",
          "data-[active=true]:after:h-[2px]",
          "data-[active=true]:after:rounded-[2px]",
          "data-[active=true]:after:bg-primary",
          "data-[active=true]:font-semibold",
          "data-[active=true]:text-primary",
        ],
      }}
    >
      <NavbarContent>
        <NavbarMenuToggle aria-label="Toggle menu" className="lg:hidden" />

        <NavbarBrand />
      </NavbarContent>

      <NavbarContent className="hidden sm:flex gap-4" justify="center">
        <NavbarItem data-active={isDocsActive}>
          <Link href="/docs" color="foreground" className="text-sm">
            Documentation
          </Link>
        </NavbarItem>

        <NavbarItem data-active={isExamplesActive}>
          <Link href="/examples" color="foreground" className="text-sm">
            Examples
          </Link>
        </NavbarItem>
      </NavbarContent>

      <NavbarContent justify="end">
        <NavbarItem>
          <Button
            as={Link}
            href="https://github.com/pablor21/gqlschemagen"
            variant="ghost"
            startContent={<Github size={18} />}
            isExternal
          >
            GitHub
          </Button>
        </NavbarItem>
      </NavbarContent>

      {/* Mobile menu */}
      <NavbarMenu>
        <div className="flex flex-col gap-4 pt-4">
          <div className="flex flex-col gap-2">
            <NavbarMenuItem data-active={isDocsActive}>
              <h3 className="text-xl font-bold mb-3">Documentation</h3>
            </NavbarMenuItem>

            {docsNavItems.map((item, index) => (
              <React.Fragment key={index}>
                {item.href ? (
                  <NavbarMenuItem
                    data-active={pathname === item.href}
                    className="pl-0"
                  >
                    <Link href={item.href} size="lg" color="foreground">
                      {item.label}
                    </Link>
                  </NavbarMenuItem>
                ) : (
                  <div className="flex flex-col">
                    <h4 className="text-md font-semibold mt-4">{item.label}</h4>
                    {item.items?.map((sub, i) => (
                      <NavbarMenuItem
                        key={i}
                        data-active={pathname === sub.href}
                        className="pl-4"
                      >
                        <Link href={sub.href} color="foreground">
                          {sub.label}
                        </Link>
                      </NavbarMenuItem>
                    ))}
                  </div>
                )}
              </React.Fragment>
            ))}
          </div>

          <NavbarMenuItem data-active={isExamplesActive}>
            <Link href="/examples" size="lg" color="foreground">
              Examples
            </Link>
          </NavbarMenuItem>
        </div>
      </NavbarMenu>
    </Navbar>
  );
}
