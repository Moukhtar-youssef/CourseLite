-- name: GetStudentCourses :many
SELECT
    c.course_id,
    c.title,
    c.description,
    c.creator_id,
    ce.enrolled_at
FROM courses c
JOIN course_enrollments ce
ON ce.course_id = c.course_id
WHERE ce.user_id = $1
ORDER BY ce.enrolled_at DESC;

-- name: GetStudentCoursesWithInstructor :many
SELECT
    c.course_id,
    c.title,
    u.name AS instructor,
    ce.enrolled_at
FROM course_enrollments ce
JOIN courses c
    ON ce.course_id = c.course_id
JOIN users u
    ON u.id = c.creator_id
WHERE ce.user_id = $1;

-- name: EnrollStudent :exec
INSERT INTO course_enrollments (
    user_id,
    course_id
) VALUES ($1, $2);

-- name: GetCourseStudents :many
SELECT
    u.id,
    u.name,
    u.email,
    ce.enrolled_at
FROM users u
JOIN course_enrollments ce
ON ce.user_id = u.id
WHERE ce.course_id = $1;

-- name: IsStudentEnrolled :one
SELECT EXISTS (
    SELECT 1
    FROM course_enrollments
    WHERE user_id = $1
    AND course_id = $2
);

-- name: GetCourseLessons :many
SELECT
    s.section_id,
    s.title AS section_title,
    l.lesson_id,
    l.title AS lesson_title
FROM sections s
JOIN lessons l
ON l.section_id = s.section_id
WHERE s.course_id = $1
ORDER BY s.position, l.position;

