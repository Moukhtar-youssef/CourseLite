package handlers

import (
	"net/http"

	DB "github.com/Moukhtar-youssef/CourseLite/internal/db"
)

type CourseHandler struct {
	DB           *DB.Queries
	AccessSecret string
}

func (h *CourseHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	Courses, err := h.DB.GetAllCourses(r.Context())
	if err != nil {
		JsonError(w, "Error fetching all courses: "+err.Error(), http.StatusInternalServerError)
	}

	JsonResponse(w, Courses, http.StatusOK)
}
