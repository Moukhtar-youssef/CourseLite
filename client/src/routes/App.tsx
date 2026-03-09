import { useState } from "react";
import reactLogo from "../assets/react.svg";
import viteLogo from "/vite.svg";

function App() {
  const [count, setCount] = useState(0);
  return (
    <div className="mx-auto max-w-4xl px-4 py-10 text-center">
      <div className="flex justify-center gap-6 mb-6">
        <a href="https://vite.dev" target="_blank">
          <img src={viteLogo} className="h-16" alt="Vite logo" />
        </a>
        <a href="https://react.dev" target="_blank">
          <img src={reactLogo} className="h-16" alt="React logo" />
        </a>
      </div>
      <h1 className="text-3xl font-bold mb-6">Vite + React</h1>
      <div className="mb-4">
        <button
          onClick={() => setCount((count) => count + 1)}
          className="px-4 py-2 rounded-md bg-gray-100 hover:bg-gray-200 dark:bg-gray-800 dark:hover:bg-gray-700"
        >
          count is {count}
        </button>
      </div>
      <p className="text-gray-500 dark:text-gray-400">
        Edit <code className="font-mono">src/App.tsx</code> and save to test HMR
      </p>
    </div>
  );
}

export default App;
