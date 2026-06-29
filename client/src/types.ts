export interface Session {
  token_id: string;
  created_at: string;
  user_agent?: string;
  ip_address?: string;
}

export interface Course {
  course_id: string;
  creator_id: string;
  title: string;
  slug: string;
  description?: string;
  price: number;
  currency: string;
  published: boolean;
  created_at: string;
}

export interface Lesson {
  lesson_id: string;
  section_id: string;
  title: string;
  content?: string;
  position: number;
}
export interface Section {
  section_id: string;
  course_id: string;
  title: string;
  position: number;
  lessons: Lesson[];
}

export interface CourseDetail extends Omit<Course, "creator_id"> {
  sections: Section[];
}

export interface FlatLesson {
  section_id: string;
  section_title: string;
  lesson_id: string;
  lesson_title: string;
}

export interface StudentCourse {
  course_id: string;
  title: string;
  instructor: string;
  enrolled_at: string;
}

export interface Student {
  id: string;
  name: string;
  email: string;
  enrolled_at: string;
}
export interface CreateCourseInput {
  title: string;
  slug: string;
  description?: string;
  price: number;
  currency: string;
}

export interface UpdateCourseInput extends CreateCourseInput {
  published: boolean;
}
