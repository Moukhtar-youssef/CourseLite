import { useQuery } from "@tanstack/react-query";
import { fetchHello } from "../api";

export function Hello() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["hello"],
    queryFn: fetchHello,
  });

  return (
    <div>
      <section>
        <h1 className="text-3xl font-bold mb-4">Hello</h1>

        {isLoading && <p>Loading...</p>}
        {error && <p>Error: {error.message}</p>}
        {data && <p>{data.message}</p>}
      </section>
    </div>
  );
}
