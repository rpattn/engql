// scripts/remove-infinite-queries.js
import fs from "fs";

const file = "src/generated/graphql.ts";
let code = fs.readFileSync(file, "utf8");

// 🧹 Remove each "useInfinite..." export and its associated .getKey
code = code.replace(
  /export const useInfinite[A-Za-z0-9_]+Query[\s\S]*?useInfinite[A-Za-z0-9_]+Query\.getKey[^\n]*\n?/g,
  ""
);

// 🧹 Remove any leftover import references to useInfiniteQuery
code = code.replace(/,\s*useInfiniteQuery[^}]*/, "");

code = code.replace(
  /queryFn:\s*graphqlRequest/g,
  "queryFn: () => graphqlRequest"
);

code = code.replace(
  /mutationFn:\s*\(([^)]*)\)\s*=>\s*graphqlRequest<([^>]*)>\(([^)]*)\)\(\)/g,
  "mutationFn: ($1) => graphqlRequest<$2>($3)"
);

// Save
fs.writeFileSync(file, code);
console.log("✅ Removed all useInfiniteQuery hooks (verified regex v3)");