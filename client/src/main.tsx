import "./index.css";
import { Layout } from "./layout.tsx";
import { BrowserRouter, Route, Routes } from "react-router";
import App from "./routes/App.tsx";
import { About } from "./routes/about.tsx";
import { Hello } from "./routes/hello.tsx";
import Login from "./routes/login.tsx";
import SignUp from "./routes/signup.tsx";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "./ThemeProvider.tsx";

const root = document.getElementById("root");

if (root == null) {
  throw new Error("Error: there is no Root in the index.html");
}

const queryClient = new QueryClient();

createRoot(root).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/" element={<Layout />}>
              <Route index element={<App />} />
              <Route path="about" element={<About />} />
              <Route path="hello" element={<Hello />} />
              <Route path="login" element={<Login />} />
              <Route path="signup" element={<SignUp />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </ThemeProvider>
    </QueryClientProvider>
  </StrictMode>,
);
