-- name: GetAllCourses :many
SELECT * FROM courses;

-- name: GetCourseByID :one
SELECT * FROM courses
WHERE course_id = $1;

-- name: GetCourseBySlug :one
SELECT * FROM courses
WHERE slug = $1;

-- name: GetInstructorCourses :many
SELECT * FROM courses
WHERE creator_id = $1
ORDER BY created_at DESC;

-- name: CreateCourse :one
INSERT INTO courses (
    creator_id,
    title,
    slug,
    description,
    price,
    currency,
    published
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateCourse :one
UPDATE courses SET
    title       = $2,
    slug        = $3,
    description = $4,
    price       = $5,
    currency    = $6,
    published   = $7
WHERE course_id = $1
RETURNING *;

-- name: DeleteCourse :exec
DELETE FROM courses
WHERE course_id = $1;

-- name: PublishCourse :exec
UPDATE courses SET published = true
WHERE course_id = $1;

-- name: UnpublishCourse :exec
UPDATE courses SET published = false
WHERE course_id = $1;

-- name: GetStudentCourses :many
SELECT
    c.course_id,
    c.title,
    c.description,
    c.creator_id,
    ce.enrolled_at
FROM courses c
JOIN course_enrollments ce ON ce.course_id = c.course_id
WHERE ce.user_id = $1
ORDER BY ce.enrolled_at DESC;

-- name: GetStudentCoursesWithInstructor :many
SELECT
    c.course_id,
    c.title,
    u.name AS instructor,
    ce.enrolled_at
FROM course_enrollments ce
JOIN courses c ON ce.course_id = c.course_id
JOIN users u   ON u.id = c.creator_id
WHERE ce.user_id = $1;

-- name: EnrollStudent :exec
INSERT INTO course_enrollments (
    user_id,
    course_id
) VALUES ($1, $2);

-- name: UnenrollStudent :exec
DELETE FROM course_enrollments
WHERE user_id = $1 AND course_id = $2;

-- name: GetCourseStudents :many
SELECT
    u.id,
    u.name,
    u.email,
    ce.enrolled_at
FROM users u
JOIN course_enrollments ce ON ce.user_id = u.id
WHERE ce.course_id = $1;

-- name: IsStudentEnrolled :one
SELECT EXISTS (
    SELECT 1
    FROM course_enrollments
    WHERE user_id  = $1
    AND   course_id = $2
);

-- name: CountCourseStudents :one
SELECT COUNT(*) FROM course_enrollments
WHERE course_id = $1;

-- name: GetCourseSections :many
SELECT * FROM sections
WHERE course_id = $1
ORDER BY position;

-- name: GetSectionByID :one
SELECT * FROM sections
WHERE section_id = $1;

-- name: CreateSection :one
INSERT INTO sections (
    course_id,
    title,
    position
) VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateSection :one
UPDATE sections SET
    title    = $2,
    position = $3
WHERE section_id = $1
RETURNING *;

-- name: DeleteSection :exec
DELETE FROM sections
WHERE section_id = $1;

-- name: GetCourseLessons :many
SELECT
    s.section_id,
    s.title      AS section_title,
    l.lesson_id,
    l.title      AS lesson_title
FROM sections s
JOIN lessons l ON l.section_id = s.section_id
WHERE s.course_id = $1
ORDER BY s.position, l.position;

-- name: GetSectionLessons :many
SELECT * FROM lessons
WHERE section_id = $1
ORDER BY position;

-- name: GetLessonByID :one
SELECT * FROM lessons
WHERE lesson_id = $1;

-- name: CreateLesson :one
INSERT INTO lessons (
    section_id,
    title,
    content,
    position
) VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateLesson :one
UPDATE lessons SET
    title    = $2,
    content  = $3,
    position = $4
WHERE lesson_id = $1
RETURNING *;

-- name: DeleteLesson :exec
DELETE FROM lessons
WHERE lesson_id = $1;

-- name: GetCourseWithSectionsAndLessons :many
SELECT
    c.course_id,
    c.title        AS course_title,
    c.description,
    c.published,
    s.section_id,
    s.title        AS section_title,
    s.position     AS section_position,
    l.lesson_id,
    l.title        AS lesson_title,
    l.position     AS lesson_position,
    l.content
FROM courses c
LEFT JOIN sections s ON s.course_id    = c.course_id
LEFT JOIN lessons  l ON l.section_id   = s.section_id
WHERE c.course_id = $1
ORDER BY s.position, l.position;
