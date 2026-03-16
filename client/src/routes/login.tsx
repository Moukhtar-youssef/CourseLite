import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { LoginApi } from "@/api";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { NavLink } from "react-router";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";

export default function Login() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [remember, setRemember] = useState(false);

  const mutation = useMutation({
    mutationFn: () => LoginApi(email, password),
    onSuccess: (data) => {
      setSuccess("Signin Successful");
      console.log("Signin successful:", data);
    },
    onError: (err: Error) => {
      setError(err.message);
    },
  });

  const handleSubmit: React.SubmitEventHandler<HTMLFormElement> = (e) => {
    e.preventDefault();
    setError(null);
    mutation.mutate();
  };

  return (
    <div className="min-w-full min-h-screen flex items-center justify-center">
      <div className="w-full max-w-md">
        <Card className="rounded-xl">
          <CardHeader>
            <CardTitle className="text-center text-2xl font-semibold">
              Welcome back!
            </CardTitle>
            <CardDescription className="text-center text-base">
              Sign in to your account to continue.
            </CardDescription>
          </CardHeader>
          <form onSubmit={handleSubmit}>
            <CardContent>
              <div className="flex flex-col gap-6">
                <div className="grid gap-2">
                  <Label htmlFor="email">Email</Label>
                  <Input
                    id="email"
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="m@example.com"
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <div className="flex items-center">
                    <Label htmlFor="password">Password</Label>
                    <NavLink
                      to="#"
                      className="text-xs ml-auto underline-offset-4 hover:underline"
                    >
                      Forgot password?
                    </NavLink>
                  </div>
                  <Input
                    id="password"
                    type="password"
                    placeholder="Password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                  />
                </div>
                <div className="flex items-center gap-2 cursor-pointer">
                  <Checkbox
                    id="remember-me"
                    className="rounded-sm"
                    checked={remember}
                    onCheckedChange={(checked) => setRemember(checked === true)}
                  />
                  <Label
                    htmlFor="remember-me"
                    className="cursor-pointer font-normal"
                  >
                    Remember me
                  </Label>
                </div>
                {error && <p className="text-sm text-red-500">{error}</p>}
                {success && <p className="text-sm text-green-500">{success}</p>}
              </div>
            </CardContent>
            <CardFooter className="flex-col gap-2 mt-4">
              <Button
                type="submit"
                className="w-full"
                disabled={mutation.isPending}
              >
                {mutation.isPending ? "Signing in..." : "Sign in"}
              </Button>
            </CardFooter>
          </form>
        </Card>

        <Card className="rounded-xl mt-4">
          <CardContent className="flex items-center justify-center gap-1 ">
            <CardDescription className="text-base">
              Not a member?
            </CardDescription>
            <NavLink
              to="/signup"
              className="text-sm font-medium underline-offset-4 hover:underline ml-1"
            >
              Create an account
            </NavLink>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
