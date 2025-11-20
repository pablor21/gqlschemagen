import type { NextConfig } from "next";
import createMDX from "@next/mdx";

const nextConfig: NextConfig = {
  pageExtensions: ["js", "jsx", "md", "mdx", "ts", "tsx"],
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
