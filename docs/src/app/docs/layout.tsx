import DocsNavbar, { Sidebar } from "./nav";

export default function DocsLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <main className="min-h-screen text-foreground">
      {/* Background Grids (copied from your homepage example) */}
      <div className="fixed inset-0 z-0 pointer-events-none">
        <div className="absolute inset-0 bg-[linear-gradient(to_right,#80808012_1px,transparent_1px),linear-gradient(to_bottom,#80808012_1px,transparent_1px)] bg-size-[24px_24px]"></div>
        <div className="absolute left-0 right-0 top-0 -z-10 m-auto h-[310px] w-[310px] rounded-full bg-primary-500 opacity-20 blur-[100px]"></div>
      </div>

      <DocsNavbar />

      {/* Main Content Area */}
      <div className="relative z-10 max-w-7xl mx-auto px-6 pt-8 pb-32 flex">
        <Sidebar />

        {/* Content Area for MDX Pages */}
        <main className="w-full lg:w-[calc(100%-16rem)] xl:w-[calc(100%-18rem)] pt-4">
          <article className="prose dark:prose-invert max-w-none">
            {children}
          </article>
        </main>
      </div>

      {/* Simple Footer (copied from your homepage example) */}
      <footer className="border-t border-white/10 py-12 text-center">
        <p className="text-default-400 text-sm">
          © 2025 GQLSchemaGen. Open Source MIT License.
        </p>
      </footer>
    </main>
  );
}
