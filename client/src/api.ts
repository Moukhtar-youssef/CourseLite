export async function fetchHello(): Promise<{ message: string }> {
  const res = await fetch("/api/hello");
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}
