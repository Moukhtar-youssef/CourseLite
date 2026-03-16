import axios from "axios";

const api = axios.create({
  baseURL: "/api",
  headers: {
    "Content-Type": "application/json",
  },
  withCredentials: true,
});
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // handle unauthorized globally
    }
    return Promise.reject(error);
  },
);

export async function fetchHello(): Promise<{ message: string }> {
  const { data } = await api.get<{ message: string }>("/hello");
  return data;
}

export async function LoginApi(
  email: string,
  password: string,
): Promise<{ message: string }> {
  const { data } = await api.post<{ message: string }>("/auth/login", {
    email,
    password,
  });
  return data;
}

export async function SignupApi(
  name: string,
  email: string,
  password: string,
): Promise<{ message: string }> {
  const { data } = await api.post<{ message: string }>("/auth/register", {
    name,
    email,
    password,
  });
  return data;
}

export default api;
