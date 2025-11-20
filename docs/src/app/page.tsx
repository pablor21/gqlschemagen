"use client";

import { ArrowRight, BookOpen, Code2, Github, Layers, Zap } from "lucide-react";
import {
  Button,
  Card,
  CardBody,
  Link,
  Navbar,
  NavbarContent,
  NavbarItem,
  NavbarMenu,
  NavbarMenuItem,
  NavbarMenuToggle,
  Snippet,
} from "@heroui/react";

import { BorderBeam } from "@/components/ui/border-beam";
import { CodeBlock } from "@/components/ui/code-block";
// Ensure you have this component or remove the import if not using it
import NavbarBrand from "@/components/layout/navbar-brand";
import { useState } from "react";

export default function Home() {
  const [isMenuOpen, setIsMenuOpen] = useState(false);

  return (
    <main className="min-h-screen text-foreground">
      {/* Background Grids */}
      <div className="fixed inset-0 z-0 pointer-events-none">
        <div className="absolute inset-0 bg-[linear-gradient(to_right,#80808012_1px,transparent_1px),linear-gradient(to_bottom,#80808012_1px,transparent_1px)] bg-size-[24px_24px]"></div>
        <div className="absolute left-0 right-0 top-0 -z-10 m-auto h-[310px] w-[310px] rounded-full bg-primary-500 opacity-20 blur-[100px]"></div>
      </div>

      {/* Navigation */}
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
          <NavbarItem>
            <Link href="/docs" color="foreground" className="text-sm">
              Documentation
            </Link>
          </NavbarItem>

          <NavbarItem>
            <Link href="/examples" color="foreground" className="text-sm">
              Examples
            </Link>
          </NavbarItem>
        </NavbarContent>

        <NavbarContent justify="end">
          <NavbarItem>
            <Button
              as={Link}
              href="[https://github.com/pablor21/gqlschemagen](https://github.com/pablor21/gqlschemagen)"
              variant="ghost"
              startContent={<Github size={18} />}
            >
              GitHub
            </Button>
          </NavbarItem>
        </NavbarContent>

        <NavbarMenu>
          <NavbarMenuItem>
            <Link href="/docs" size="lg" color="foreground">
              Documentation
            </Link>
          </NavbarMenuItem>
          <NavbarMenuItem>
            <Link href="/examples" size="lg" color="foreground">
              Examples
            </Link>
          </NavbarMenuItem>
        </NavbarMenu>
      </Navbar>

      <main className="relative z-10 max-w-7xl mx-auto px-6 pt-20 pb-32">
        {/* Hero Section */}
        <div className="flex flex-col items-center text-center mb-32">
          {/* <Chip
            variant="flat"
            color="primary"
            className="mb-6 border border-primary/20 bg-primary/10"
            startContent={<Zap size={14} className="ml-1" />}
          >
            v1.0.0 Now Available
          </Chip> */}

          <h1 className="text-5xl md:text-7xl font-bold tracking-tight mb-6">
            Type-safe GraphQL <br />
            <span className="text-transparent bg-clip-text bg-linear-to-r from-blue-400 via-purple-400 to-pink-400">
              from your Go models.
            </span>
          </h1>

          <p className="text-lg md:text-xl text-default-500 max-w-2xl mb-10">
            Stop writing schemas by hand. Annotate your standard Go structs and
            generate production-ready GraphQL schemas instantly.
          </p>

          <div className="flex flex-col sm:flex-row gap-4">
            <Button
              color="primary"
              size="lg"
              endContent={<ArrowRight size={18} />}
              className="font-semibold"
              as={Link}
              href="/docs"
            >
              Read the Docs
            </Button>
          </div>
        </div>

        {/* Features Grid (Quick Intro) */}
        <div className="grid md:grid-cols-3 gap-6 mb-32">
          {[
            {
              title: "Declarative",
              icon: <Code2 className="text-blue-400" />,
              desc: "Define schema using simple Go struct tags.",
            },
            {
              title: "Zero Boilerplate",
              icon: <Layers className="text-purple-400" />,
              desc: "No need to maintain separate .graphql files.",
            },
            {
              title: "Go Native",
              icon: <Zap className="text-yellow-400" />,
              desc: "Works seamlessly with your existing domain entities.",
            },
          ].map((feature, idx) => (
            <Card
              key={idx}
              className="bg-default-50/50 border border-default-100 backdrop-blur-sm"
            >
              <CardBody className="p-6">
                <div className="mb-4 p-2 bg-default-100 w-fit rounded-lg">
                  {feature.icon}
                </div>
                <h3 className="text-xl font-semibold mb-2">{feature.title}</h3>
                <p className="text-default-500">{feature.desc}</p>
              </CardBody>
            </Card>
          ))}
        </div>

        {/* 3 Steps Section */}
        <div className="max-w-5xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="text-3xl font-bold mb-4">Get started in seconds</h2>
            <p className="text-default-500">
              Three simple steps to automate your schema workflow.
            </p>
          </div>

          {/* Step 1 */}
          <div className="flex flex-col md:flex-row gap-8 items-start mb-20 relative">
            <div className="md:w-1/3 flex flex-col md:items-end md:text-right mt-4">
              <div className="text-6xl font-bold text-default-100 absolute -top-8 md:-right-6 -z-10 select-none">
                1
              </div>
              <h3 className="text-xl font-bold text-primary mb-2">Install</h3>
              <p className="text-default-400">
                Get the latest version of the CLI tool directly from standard Go
                modules.
              </p>
            </div>
            <div className="md:w-2/3 w-full">
              <Card className="bg-[#0d1117] border-default-200">
                <CardBody className="p-0">
                  <Snippet
                    symbol="$"
                    className="w-full bg-transparent text-default-300 font-mono py-6 px-6"
                  >
                    go install github.com/pablor21/gqlschemagen@latest
                  </Snippet>
                </CardBody>
              </Card>
            </div>
          </div>

          {/* Step 2 */}
          <div className="flex flex-col md:flex-row gap-8 items-start mb-20 relative">
            <div className="md:w-1/3 flex flex-col md:items-end md:text-right mt-4">
              <div className="text-6xl font-bold text-default-100 absolute -top-8 md:-right-6 -z-10 select-none">
                2
              </div>
              <h3 className="text-xl font-bold text-primary mb-2">Annotate</h3>
              <p className="text-default-400">
                Add the{" "}
                <code className="text-xs bg-default-100 px-1 py-0.5 rounded">
                  @gqlType
                </code>{" "}
                directive to your Go structs to expose them to the schema.
              </p>
            </div>
            <div className="md:w-2/3 w-full">
              <Card className="bg-[#0d1117] border-default-200 relative overflow-hidden">
                {/* MagicUI Border Beam */}
                <BorderBeam
                  size={250}
                  duration={12}
                  delay={9}
                  borderWidth={1.5}
                />

                <div className="flex items-center justify-between px-4 py-2 bg-default-50/5 border-b border-white/5">
                  <div className="flex gap-2">
                    <div className="w-3 h-3 rounded-full bg-red-500/20"></div>
                    <div className="w-3 h-3 rounded-full bg-yellow-500/20"></div>
                    <div className="w-3 h-3 rounded-full bg-green-500/20"></div>
                  </div>
                  <span className="text-xs text-default-500">
                    internal/domain/user.go
                  </span>
                </div>
                <CardBody className="p-6 font-mono text-sm overflow-x-auto">
                  <CodeBlock
                    language="go"
                    filename="user.go"
                    label="GraphQL Type Definition"
                  >{`/**
 * @gqlType(name: "UserProfile", description: "Represents a user")
 */

type User struct {
    ID string
    Name string
}`}</CodeBlock>
                </CardBody>
              </Card>
            </div>
          </div>

          {/* Step 3 */}
          <div className="flex flex-col md:flex-row gap-8 items-start relative">
            <div className="md:w-1/3 flex flex-col md:items-end md:text-right mt-4">
              <div className="text-6xl font-bold text-default-100 absolute -top-8 md:-right-6 -z-10 select-none">
                3
              </div>
              <h3 className="text-xl font-bold text-primary mb-2">Generate</h3>
              <p className="text-default-400">
                Run the generator to build your GraphQL schema file
                automatically.
              </p>
            </div>
            <div className="md:w-2/3 w-full">
              <Card className="bg-[#0d1117] border-default-200">
                <CardBody className="p-0">
                  <Snippet
                    symbol="$"
                    className="w-full bg-transparent text-default-300 font-mono py-6 px-6"
                  >
                    gqlschemagen generate -p ./internal/domain -o ./graph
                  </Snippet>
                </CardBody>
              </Card>
            </div>
          </div>
        </div>

        {/* Final CTA */}
        <div className="mt-32 text-center">
          <Card className="max-w-3xl mx-auto bg-linear-to-b from-default-50 to-transparent border border-default-100">
            <CardBody className="py-12 px-8">
              <h2 className="text-2xl font-bold text-white mb-4">
                Ready to speed up development?
              </h2>
              <p className="text-default-500 mb-8">
                Join other Go developers who are automating their GraphQL
                workflow.
              </p>
              <Button
                as={Link}
                href="/docs"
                color="primary"
                size="lg"
                startContent={<BookOpen size={18} />}
              >
                Check the Documentation
              </Button>
            </CardBody>
          </Card>
        </div>
      </main>

      {/* Simple Footer */}
      <footer className="border-t border-white/10 py-12 text-center">
        <p className="text-default-400 text-sm">
          © 2025 GQLSchemaGen. Open Source MIT License.
        </p>
      </footer>
    </main>
  );
}
