// Package handlers contains HTTP handlers for the CourseLite service layer.
package handlers

import (
	"net/http"

	DB "github.com/Moukhtar-youssef/CourseLite/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CourseHandler struct {
	DB           *DB.Queries
	AccessSecret string
}

type LessonResponse struct {
	LessonID string `json:"lesson_id"`
	Title    string `json:"title"`
	Content  string `json:"content,omitempty"`
	Position int32  `json:"position"`
}

type SectionResponse struct {
	SectionID string           `json:"section_id"`
	Title     string           `json:"title"`
	Position  int32            `json:"position"`
	Lessons   []LessonResponse `json:"lessons"`
}

type CourseResponse struct {
	CourseID    string            `json:"course_id"`
	Title       string            `json:"title"`
	Slug        string            `json:"slug"`
	Description string            `json:"description,omitempty"`
	Price       int32             `json:"price"`
	Currency    string            `json:"currency"`
	Published   bool              `json:"published"`
	Sections    []SectionResponse `json:"sections"`
}

type CourseSearchResponse struct{}

type CouresResponse struct {
	Courses []CourseSearchResponse `json:"courses"`
}

// GetAll handles the HTTP request to retrieve all courses.
// It fetches all courses from the database and returns them as JSON.
func (h *CourseHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	Courses, err := h.DB.GetAllCourses(r.Context())
	if err != nil {
		JSONError(w, "Error fetching all courses", http.StatusInternalServerError)
		return
	}

	JSONResponse(w, Courses, http.StatusOK)
}

func (h *CourseHandler) GetPaginatedOrderByTitle(w http.ResponseWriter, r *http.Request) {
	Courses, err := h.DB.GetAllCoursesPaginatedByTitle(
		r.Context(),
		DB.GetAllCoursesPaginatedByTitleParams{
			Limit:  10,
			Offset: 10,
		},
	)
	if err != nil {
		JSONError(w, "Error fetching courses", http.StatusInternalServerError)
		return
	}
	JSONResponse(w, Courses, http.StatusOK)
}

// GetCourse handles the HTTP request to retrieve a single course by its slug.
// It fetches the course details, its sections, and lessons for each section,
// then returns the complete course data as JSON.
func (h *CourseHandler) GetCourse(w http.ResponseWriter, r *http.Request) {
	SlugParam := chi.URLParam(r, "slug")

	course, err := h.DB.GetCourseBySlug(r.Context(), SlugParam)
	if err != nil {
		JSONError(w, "Error fetching course", http.StatusInternalServerError)
	}

	sections, err := h.DB.GetCourseSections(r.Context(), course.CourseID)
	if err != nil {
		JSONError(w, "Error fetching course's lessons",
			http.StatusInternalServerError)
	}

	sectionResponses := make([]SectionResponse, 0, len(sections))
	for _, section := range sections {
		lessons, err := h.DB.GetSectionLessons(r.Context(), section.SectionID)
		if err != nil {
			JSONError(w, "error fetching lessons", http.StatusInternalServerError)
			return
		}
		lessonResponses := make([]LessonResponse, 0, len(lessons))
		for _, lesson := range lessons {
			lr := LessonResponse{
				LessonID: lesson.LessonID.String(),
				Title:    lesson.Title,
				Position: lesson.Position,
			}
			if lesson.Content != nil {
				lr.Content = *lesson.Content
			}
			lessonResponses = append(lessonResponses, lr)
		}
		sectionResponses = append(sectionResponses, SectionResponse{
			SectionID: section.SectionID.String(),
			Title:     section.Title,
			Position:  section.Position,
			Lessons:   lessonResponses,
		})
	}

	resp := CourseResponse{
		CourseID:  course.CourseID.String(),
		Title:     course.Title,
		Slug:      course.Slug,
		Price:     course.Price,
		Currency:  course.Currency,
		Published: course.Published,
		Sections:  sectionResponses,
	}
	if course.Description != nil {
		resp.Description = *course.Description
	}
	JSONResponse(w, resp, http.StatusOK)
}

// IsEnrolled handles the HTTP request to check if the authenticated student
// is enrolled in a specific course. It returns a JSON response indicating
// whether the student is enrolled.
func (h *CourseHandler) IsEnrolled(w http.ResponseWriter, r *http.Request) {
	claims, err := claimsFromCookie(r, h.AccessSecret)
	if err != nil {
		JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	studentID, err := uuid.Parse(claims.UserID)
	if err != nil {
		JSONError(w, "invalid token subject", http.StatusUnauthorized)
		return
	}
	SlugParam := chi.URLParam(r, "slug")
	coursesid, err := h.DB.GetCourseIDBySlug(r.Context(), SlugParam)
	if err != nil {
		JSONError(w, "Error fetching course", http.StatusInternalServerError)
	}
	enrolled, err := h.DB.IsStudentEnrolled(r.Context(),
		DB.IsStudentEnrolledParams{
			UserID:   studentID,
			CourseID: coursesid,
		})
	if err != nil {
		JSONError(w, "error checking enrollment", http.StatusInternalServerError)
		return
	}
	JSONResponse(w, map[string]bool{"enrolled": enrolled}, http.StatusOK)
}

// Enroll handles the HTTP request to enroll the student in a specific course.
// It return a JSON response indicating that the student is now enrolled.
func (h *CourseHandler) Enroll(w http.ResponseWriter, r *http.Request) {
	claims, err := claimsFromCookie(r, h.AccessSecret)
	if err != nil {
		JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	studentID, err := uuid.Parse(claims.ID)
	if err != nil {
		JSONError(w, "invalid token subject", http.StatusUnauthorized)
		return
	}
	SlugParam := chi.URLParam(r, "slug")
	courseid, err := h.DB.GetCourseIDBySlug(r.Context(), SlugParam)
	if err != nil {
		JSONError(w, "Error fetching course", http.StatusInternalServerError)
	}
	err = h.DB.EnrollStudent(r.Context(), DB.EnrollStudentParams{
		UserID:   studentID,
		CourseID: courseid,
	})
	if err != nil {
		JSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	JSONResponse(w, "Student enrolled", http.StatusOK)
}
