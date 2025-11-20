import { Avatar, NavbarBrand as HeroNavbarBrand } from "@heroui/react";

import Link from "next/link";

export default function NavbarBrand() {
  return (
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    <HeroNavbarBrand as={Link as any} href="/" color="foreground">
      <Avatar
        className="bg-linear-to-br from-blue-500 to-purple-600 font-bold rounded-lg w-8 h-8 flex items-center justify-center mr-2"
        name="GQLSchemaGen"
      />
      <p className="font-bold">GQLSchemaGen</p>
    </HeroNavbarBrand>
  );
}
