import axios from "axios";
import {
  type CourseDetail,
  type Course,
  type FlatLesson,
  type Session,
} from "./types";

const api = axios.create({
  baseURL: "/api",
  headers: {
    "Content-Type": "application/json",
  },
  withCredentials: true,
});

let isRefreshing = false;
let refreshQueue: Array<{
  resolve: (value: unknown) => void;
  reject: (reason?: unknown) => void;
}> = [];

function drainQueue(error: unknown) {
  refreshQueue.forEach(({ resolve, reject }) =>
    error ? reject(error) : resolve(null),
  );
  refreshQueue = [];
}

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const original = error.config;

    if (
      error.response?.status !== 401 ||
      original._retry ||
      original.url === "/auth/refresh"
    ) {
      return Promise.reject(error);
    }

    if (isRefreshing) {
      // Queue concurrent requests while a refresh is in flight
      return new Promise((resolve, reject) => {
        refreshQueue.push({ resolve, reject });
      }).then(() => api(original));
    }

    original._retry = true;
    isRefreshing = true;

    try {
      await api.post("/auth/refresh");
      drainQueue(null);
      return api(original);
    } catch (refreshError) {
      drainQueue(refreshError);
      window.location.href = "/login";
      return Promise.reject(refreshError);
    } finally {
      isRefreshing = false;
    }
  },
);

export async function fetchHello(): Promise<{ message: string }> {
  const { data } = await api.get<{ message: string }>("/hello");
  return data;
}

export const auth = {
  login: (email: string, password: string) =>
    api
      .post<{ user_id: string; email: string }>("/auth/login", {
        email,
        password,
      })
      .then((r) => r.data),

  register: (name: string, email: string, password: string) =>
    api
      .post<{ user_id: string; email: string }>("/auth/register", {
        name,
        email,
        password,
      })
      .then((r) => r.data),
  logout: () => api.post("/auth/logout").then(() => undefined),

  forgotPassword: (email: string) =>
    api
      .post<{ message: string }>("/auth/forgot-password", { email })
      .then((r) => r.data),

  resetPassword: (token: string, password: string) =>
    api
      .post<{ message: string }>("/auth/reset-password", { token, password })
      .then((r) => r.data),

  sessions: () => api.get<Session[]>("/auth/sessions").then((r) => r.data),
};

export const courses = {
  getAll: () => api.get<Course[]>("/courses/").then((r) => r.data),

  getBySlug: (slug: string) =>
    api.get<CourseDetail>(`/courses/${slug}/`).then((r) => r.data),

  getLessons: (slug: string) =>
    api.get<FlatLesson[]>(`/courses/${slug}/lessons`).then((r) => r.data),

  isEnrolled: (slug: string) =>
    api
      .get<{ enrolled: boolean }>(`/courses/${slug}/enrolled`)
      .then((r) => r.data),

  enroll: (slug: string) =>
    api.post(`/courses/${slug}/enroll`).then(() => undefined),

  unenroll: (slug: string) =>
    api.delete(`/courses/${slug}/enroll`).then(() => undefined),
};

export default api;
