import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { SignupApi } from "@/api";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { NavLink } from "react-router";
import { Label } from "@/components/ui/label";

export default function SignUp() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [remember, setRemember] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: () => SignupApi(name, email, password),
    onSuccess: (data) => {
      console.log("Signup successful:", data);
    },
    onError: (err: Error) => {
      setError(err.message);
    },
  });

  const handleSubmit: React.SubmitEventHandler<HTMLFormElement> = (e) => {
    e.preventDefault();

    if (password !== confirmPassword) {
      setError("Passwords do not match");
      return;
    }

    setError(null);
    mutation.mutate();
  };

  return (
    <div className="min-w-full min-h-screen flex items-center justify-center">
      <div className="w-full max-w-md">
        <Card className="rounded-xl">
          <CardHeader>
            <CardTitle className="text-center text-2xl font-semibold">
              Create an account
            </CardTitle>
            <CardDescription className="text-center text-base">
              Sign up to start using the platform.
            </CardDescription>
          </CardHeader>
          <form onSubmit={handleSubmit}>
            <CardContent>
              <div className="flex flex-col gap-6">
                <div className="grid gap-2">
                  <Label htmlFor="name">Name</Label>
                  <Input
                    id="name"
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Your name"
                    required
                  />
                </div>
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
                  <Label htmlFor="password">Password</Label>
                  <Input
                    id="password"
                    type="password"
                    placeholder="Password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="confirmPassword">Re-type password</Label>
                  <Input
                    id="confirmPassword"
                    type="password"
                    placeholder="Re-type password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    required
                  />
                </div>
                <div className="flex items-center gap-2 cursor-pointer">
                  <Checkbox
                    id="remember"
                    className="rounded-sm"
                    checked={remember}
                    onCheckedChange={(checked) => setRemember(checked === true)}
                  />
                  <Label
                    htmlFor="remember"
                    className="cursor-pointer font-normal"
                  >
                    Remember me
                  </Label>
                </div>
                {error && <p className="text-sm text-red-500">{error}</p>}
              </div>
            </CardContent>
            <CardFooter className="flex-col gap-2 mt-4">
              <Button
                type="submit"
                className="w-full"
                disabled={mutation.isPending}
              >
                {mutation.isPending ? "Creating account..." : "Sign Up"}
              </Button>
            </CardFooter>
          </form>
        </Card>

        <Card className="rounded-xl mt-4">
          <CardContent className="flex items-center justify-center gap-1 ">
            <CardDescription className="text-base">
              Already have an account?
            </CardDescription>
            <NavLink
              to="/login"
              className="text-sm font-medium underline-offset-4 hover:underline ml-1"
            >
              Login
            </NavLink>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
