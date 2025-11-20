import type { NextConfig } from "next";
import createMDX from "@next/mdx";

const basePath = process.env.PAGES_BASE_PATH || "";
console.log("Building with basePath:", basePath);

const nextConfig: NextConfig = {
  pageExtensions: ["js", "jsx", "md", "mdx", "ts", "tsx"],
  output: "export",
  basePath: basePath,
};

const withMDX = createMDX({
  options: {
    remarkPlugins: [
      ["remark-gfm", { strict: true, throwOnError: true }],
      ["remark-toc", { heading: "The Table" }],
    ],
    rehypePlugins: [],
  },
});

export default withMDX(nextConfig);
