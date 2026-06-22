import { redirect } from "@remix-run/node";

export async function loader() {
  return redirect("/projects");
}

// The default export is required by Remix but unused — `loader` always
// redirects before any component renders.
export default function Index() {
  return null;
}
