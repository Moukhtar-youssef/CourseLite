import { Outlet, NavLink } from "react-router";
import { ModeToggle } from "./components/mode-toggle";
import viteLogo from "/vite.svg";

type Route = {
  name: string;
  href: string;
};

const routes: Route[] = [
  { name: "Home", href: "/" },
  { name: "About", href: "/about" },
  { name: "Hello", href: "/hello" },
];

export function Layout() {
  return (
    <>
      <nav
        className="flex items-center justify-between gap-4 border-b 
        border-gray-200 bg-white/80 px-6 py-3 backdrop-blur 
        dark:border-gray-800 dark:bg-gray-950/80"
      >
        <img src={viteLogo} className="h-10" alt="Vite logo" />
        <div className="flex items-center justify-center gap-6">
          {routes.map((route) => (
            <NavLink
              key={route.href}
              to={route.href}
              className={({ isActive }) => (isActive ? "font-bold" : "")}
            >
              {route.name}
            </NavLink>
          ))}
        </div>
        <ModeToggle />
      </nav>
      <Outlet />
    </>
  );
}
