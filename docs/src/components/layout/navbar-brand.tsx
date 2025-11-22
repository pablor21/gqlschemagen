import { Avatar, NavbarBrand as HeroNavbarBrand } from "@heroui/react";

import Link from "next/link";

export default function NavbarBrand() {
  return (
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    <HeroNavbarBrand as={Link as any} href="/" color="foreground">
      <Avatar
        className="bg-white font-bold rounded-lg w-8 h-8 flex items-center justify-center mr-2"
        classNames={{
          img: "object-contain p-1",
        }}
        name="GQLSchemaGen"
        src={`${process.env.NEXT_PUBLIC_BASE_PATH ?? ""}/logo.png`}
      />
      <p className="font-bold">GQLSchemaGen</p>
    </HeroNavbarBrand>
  );
}
