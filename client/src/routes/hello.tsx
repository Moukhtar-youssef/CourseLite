import { useQuery } from "@tanstack/react-query";
import { fetchHello } from "../api";

export function Hello() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["hello"],
    queryFn: fetchHello,
  });

  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <h1 className="text-3xl font-bold mb-4">Hello</h1>
      {isLoading && <p>Loading...</p>}
      {error && <p>Error: {error.message}</p>}
      {data && <p>{data.message}</p>}
    </div>
  );
}
