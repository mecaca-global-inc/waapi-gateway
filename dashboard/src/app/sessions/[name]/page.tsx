// Static export cannot enumerate unknown dynamic params. This route is kept
// as a build-time placeholder; live navigation goes to /sessions/detail?name=...
export const dynamic = "force-static";
export const dynamicParams = false;

export function generateStaticParams() {
  return [] as { name: string }[];
}

export default function StubPage() {
  return null;
}
